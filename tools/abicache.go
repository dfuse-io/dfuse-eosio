package tools

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/streamingfast/bstream"
	dfuse "github.com/streamingfast/client-go"
	"github.com/dfuse-io/dfuse-eosio/abicodec"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/streamingfast/dgrpc"
	"github.com/streamingfast/dstore"
	pbbstream "github.com/streamingfast/pbgo/dfuse/bstream/v1"
	"github.com/eoscanada/eos-go"
	eossnapshot "github.com/eoscanada/eos-go/snapshot"
	"github.com/golang/protobuf/ptypes"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/oauth"
)

var statusFrequency = 15 * time.Second

var abicacheCmd = &cobra.Command{Use: "abi", Short: ""}
var exportCmd = &cobra.Command{
	Use:   "export {block_num} {export_filename}",
	Short: "export",
	Args:  cobra.ExactArgs(2),
	RunE:  exportE,
}

func init() {
	Cmd.AddCommand(abicacheCmd)

	abicacheCmd.AddCommand(exportCmd)
	exportCmd.Flags().String("abi-snapshot-store-url", "", "If not snapshot path is provided,we will attempt to find the closest snapshot to the desired block")
	exportCmd.Flags().String("abi-snapshot-path", "", "Snapshot path to start from")
	exportCmd.Flags().String("firehose-grpc-addr", "blocks.mainnet.eos.dfuse.io:443/", "Firehose GRPC serving address")
	exportCmd.Flags().Bool("secure", false, "When set, skips certification verification")
}

func exportE(cmd *cobra.Command, args []string) (err error) {
	ctx := cmd.Context()
	fmt.Println("")
	outputStore := args[1]
	blockNum, err := strconv.ParseUint(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("Unable to parse %q as block number: %w", args[0], err)
	}

	snapshotURL := ""
	if viper.GetString("abi-snapshot-path") != "" {
		snapshotURL = viper.GetString("abi-snapshot-path")
		fmt.Printf("Starting ABI export from given snapshot: %q\n", snapshotURL)
	} else if viper.GetString("abi-snapshot-store-url") != "" {
		snapshotStoreURL := viper.GetString("abi-snapshot-store-url")
		fmt.Printf("Searching for closest snapshot to block number %d from store: %q\n", blockNum, snapshotStoreURL)
		file, err := getSnapshotPath(blockNum, snapshotStoreURL)
		if err != nil {
			return fmt.Errorf("Unable to retrieve preferred snapshot: %w", err)
		}
		fmt.Printf("Found closest snapshot to desired block num: %q\n", file.Source)

		snapshotURL = snapshotStoreURL
		if !strings.HasSuffix(snapshotURL, "/") {
			snapshotURL += "/"
		}
		snapshotURL += file.Source
	} else {
		return fmt.Errorf("You must either specify a snapshot path (--abi-snapshot-path) or a snapthost store URL to proceed (--abi-snapshot-store-url)")
	}

	e := &exporter{
		desiredBlockNum: blockNum,
		snapshotPath:    snapshotURL,
		abis: &abiCache{
			Abis: map[string]*abicodec.ABICacheItem{},
		},
	}
	if err = e.export(); err != nil {
		return fmt.Errorf("ABI export from snapshot %q failed: %w", snapshotURL, err)
	}

	if e.snapshotBlockRef.Num() == blockNum {
		fmt.Printf("Snapshot block number (%d) matches your desired block number (%d). You are done!\n", e.snapshotBlockRef.Num(), blockNum)
		e.abis.BlockId = e.snapshotBlockRef.ID()
		e.abis.BlockNum = blockNum
	} else {
		fmt.Printf("Snapshot block number (%d) does not match your desired block number (%d)\n", e.snapshotBlockRef.Num(), blockNum)
		blcokRef, err := e.firehose(ctx, e.snapshotBlockRef.Num()+1, blockNum)
		if err != nil {
			return fmt.Errorf("unable to complete firehose: %w", err)

		}
		fmt.Printf("Fire hose completed! Total ABI count: %d\n", len(e.abis.Abis))
		e.abis.BlockId = blcokRef.ID()
		e.abis.BlockNum = blcokRef.Num()
	}

	if err = e.output(ctx, outputStore); err != nil {
		return fmt.Errorf("Unable to output ABIS: %w", err)
	}

	return nil
}

type exporter struct {
	desiredBlockNum  uint64
	snapshotPath     string
	snapshotBlockRef bstream.BlockRef
	abis             *abiCache
}

type abiCache struct {
	BlockId  string `json:"block_id"`
	BlockNum uint64 `json:"block_num"`
	Abis     map[string]*abicodec.ABICacheItem
}

func (e *exporter) firehose(ctx context.Context, startBlockNum, stopBlockNum uint64) (bstream.BlockRef, error) {
	apiKey := os.Getenv("DFUSE_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("the environment variable DFUSE_API_KEY must be set to a valid dfuse API key value")
	}

	endpoint := viper.GetString("firehose-grpc-addr")
	filter := ""
	blockRange := &BlockRange{
		Start: startBlockNum,
		Stop:  stopBlockNum,
	}

	var dialOptions []grpc.DialOption
	//if viper.GetBool("secure") {
	//	dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{InsecureSkipVerify: true}))}
	//}

	conn, err := dgrpc.NewExternalClient(endpoint, dialOptions...)
	if err != nil {
		return nil, fmt.Errorf("unable to create external gRPC client to %q: %w", endpoint, err)
	}

	dfuse, err := dfuse.NewClient(endpoint, apiKey)
	streamClient := pbbstream.NewBlockStreamV2Client(conn)

	stats := newStats()
	nextStatus := time.Now().Add(statusFrequency)

	cursor := ""
	lastBlockRef := bstream.BlockRefEmpty

	fmt.Printf("Starting firehose %q from %d to %d\n", endpoint, blockRange.Start, blockRange.Stop)

	for {
		tokenInfo, err := dfuse.GetAPITokenInfo(ctx)
		if err != nil {
			fmt.Errorf("unable to retrieve dfuse API token: %w", err)
		}

		credentials := oauth.NewOauthAccess(&oauth2.Token{AccessToken: tokenInfo.Token, TokenType: "Bearer"})
		stream, err := streamClient.Blocks(context.Background(), &pbbstream.BlocksRequestV2{
			StartBlockNum:     int64(blockRange.Start),
			StartCursor:       cursor,
			StopBlockNum:      blockRange.Stop,
			ForkSteps:         []pbbstream.ForkStep{pbbstream.ForkStep_STEP_IRREVERSIBLE},
			IncludeFilterExpr: filter,
			Details:           pbbstream.BlockDetails_BLOCK_DETAILS_LIGHT,
		}, grpc.PerRPCCredentials(credentials))
		if err != nil {
			fmt.Errorf("unable to start blocks stream: %w", err)
		}

		for {
			zlog.Debug("waiting for message to reach us")
			response, err := stream.Recv()
			if err != nil {
				if err == io.EOF {
					fmt.Printf("Firehose completed. Ended at block num: %d with id: %s \n", lastBlockRef.Num(), lastBlockRef.ID())
					return lastBlockRef, nil
				}

				zlog.Error("Stream encountered a remote error, going to retry",
					zap.String("cursor", cursor),
					zap.Stringer("last_block", lastBlockRef),
					zap.Error(err),
				)
				break
			}
			zlog.Debug("Decoding received message's block")
			block := &pbcodec.Block{}
			err = ptypes.UnmarshalAny(response.Block, block)
			if err != nil {
				fmt.Errorf("unable to unmarshal received block payload %w", err)
			}

			cursor = response.Cursor
			lastBlockRef = block.AsRef()

			for _, trx := range block.TransactionTraces() {
				for _, action := range trx.ActionTraces {
					e.handleABIAction(block.AsRef(), trx.Id, action)
				}
			}

			now := time.Now()
			if now.After(nextStatus) {
				fmt.Printf("> Stream blocks progress at block %d -> blocks: %s, bytes: %s\n", block.Number, stats.blockReceived.String(), stats.bytesReceived.String())
				nextStatus = now.Add(statusFrequency)
			}
			stats.recordBlock(int64(response.XXX_Size()))
		}
	}
	return nil, fmt.Errorf("unable to run firehose")
}

func (e *exporter) handleABIAction(blockRef bstream.BlockRef, trxID string, actionTrace *pbcodec.ActionTrace) error {
	account := actionTrace.GetData("account").String()
	hexABI := actionTrace.GetData("abi")

	if !hexABI.Exists() {
		zlog.Warn("'setabi' action data payload not present", zap.String("account", account), zap.String("transaction_id", trxID))
		return nil
	}

	hexData := hexABI.String()
	if hexData == "" {
		zlog.Info("empty ABI in 'setabi' action", zap.String("account", account), zap.String("transaction_id", trxID))
		return nil
	}

	abiData, err := hex.DecodeString(hexData)
	if err != nil {
		zlog.Info("failed to hex decode abi string", zap.String("account", account), zap.String("transaction_id", trxID), zap.Error(err))
		return nil // do not return the error. Worker will retry otherwise
	}

	var abi *eos.ABI
	err = eos.UnmarshalBinary(abiData, &abi)
	if err != nil {
		abiHexCutAt := math.Min(50, float64(len(hexData)))

		zlog.Info("failed to unmarshal abi from binary",
			zap.String("account", account),
			zap.String("transaction_id", trxID),
			zap.String("abi_hex_prefix", hexData[0:int(abiHexCutAt)]),
			zap.Error(err),
		)

		return nil
	}

	zlog.Debug("setting new abi", zap.String("account", account), zap.Stringer("transaction_id", blockRef), zap.Stringer("block", blockRef))

	e.abis.Abis[account] = &abicodec.ABICacheItem{
		ABI:      abi,
		BlockNum: uint32(blockRef.Num()),
	}
	return nil
}

func (e *exporter) output(ctx context.Context, outputStoreURL string) error {
	fmt.Println("Saving ABIS")
	filename := "output.json.zst"
	store, err := dstore.NewStore(outputStoreURL, "", "zstd", true)
	if err != nil {
		return fmt.Errorf("error creating export store: %w", err)
	}

	data, err := json.Marshal(e.abis)
	if err != nil {
		return fmt.Errorf("error marshalling abis: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	err = store.WriteObject(ctx, filename, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("exporting cache: %w", err)
	}

	fmt.Printf("Abis Exported %q\n", filepath.Join(outputStoreURL, filename))

	return nil

}
func (e *exporter) export() error {
	fmt.Printf("Exporting ABIs from EOS snapshot @ %q\n", e.snapshotPath)

	reader, err := eossnapshot.NewDefaultReader(e.snapshotPath)
	if err != nil {
		return fmt.Errorf("unable to create a snapshot reader: %w", err)
	}
	defer func() {
		reader.Close()
	}()

	for {
		err := reader.NextSection()
		if err == io.EOF {
			break
		}
		if err != nil {
			zlog.Error("failed reading snapshot",
				zap.String("snapshot_path", e.snapshotPath),
				zap.Error(err),
			)
			return err
		}

		currentSection := reader.CurrentSection
		switch currentSection.Name {
		case eossnapshot.SectionNameAccountObject:
			zlog.Info("reading snapshot account objects")
			err = reader.ProcessCurrentSection(e.processSnapshotAccountObject)
		case eossnapshot.SectionNameBlockState:
			zlog.Info("reading snapshot block state objects")
			err = reader.ProcessCurrentSection(e.processBlockState)
		}
		if err != nil {
			return fmt.Errorf("failed processing snapshot section %q: %w", currentSection.Name, err)
		}
	}
	fmt.Printf("Snpashot ABI export completed, found %d ABIs\n", len(e.abis.Abis))
	return nil
}

func (e *exporter) processSnapshotAccountObject(obj interface{}) (err error) {
	acc, ok := obj.(eossnapshot.AccountObject)
	if !ok {
		return fmt.Errorf("failed processing account object: unexpected object type: %T", obj)
	}
	if len(acc.RawABI) == 0 {
		return nil
	}

	if _, found := e.abis.Abis[acc.Name.String()]; found {
		return fmt.Errorf("failed processing account object: received seen ABI %q", string(acc.Name))
	}

	zlog.Info("adding ABI", zap.Stringer("contract", acc.Name))
	abi := new(eos.ABI)

	err = eos.UnmarshalBinary(acc.RawABI, abi)
	if err != nil {
		return nil
	}

	e.abis.Abis[acc.Name.String()] = &abicodec.ABICacheItem{
		ABI:      abi,
		BlockNum: 0,
	}
	return nil
}

func (e *exporter) processBlockState(obj interface{}) error {
	state, ok := obj.(eossnapshot.BlockState)
	if !ok {
		return fmt.Errorf("failed processing block state: unexpected object type: %T", obj)
	}

	if e.snapshotBlockRef != nil {
		return fmt.Errorf("snapshot block ref already specified")
	}

	if uint64(state.BlockNum) > e.desiredBlockNum {
		return fmt.Errorf("snapshot block number %d cannot execeed desired block bumber %d", state.BlockNum, e.abis.BlockNum)
	}

	e.snapshotBlockRef = bstream.NewBlockRef(string(state.BlockID), uint64(state.BlockNum))
	return nil
}

func getSnapshotPath(desiredBlockNum uint64, snapshotStoreURL string) (out *stateFile, err error) {
	zlog.Info(
		"downloading EOS state snaphosts list",
		zap.String("snapshot_Store_url", snapshotStoreURL),
	)
	stateFiles, err := getStateFiles(snapshotStoreURL)
	if err != nil {
		return nil, fmt.Errorf("get state files: %s", err)
	}

	for _, file := range stateFiles {
		if out == nil {
			out = file
			continue
		}

		if file.StartBlock > desiredBlockNum {
			continue
		}

		if file.StartBlock > out.StartBlock {
			out = file
		}
	}
	return
}

func getStateFiles(bucket string) (out []*stateFile, err error) {
	output, err := exec.Command("gsutil", "ls", bucket).Output()
	if err != nil {
		return nil, fmt.Errorf("gsutil failed: %s", err)
	}

	re := regexp.MustCompile(`/((\d{10})-[a-z0-9]+-snapshot.bin)`)

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		match := re.FindStringSubmatch(line)
		if match == nil {
			continue
		}

		blockNum, err := strconv.ParseUint(match[2], 10, 64)
		if err != nil {
			log.Fatalln("invalid block num:", line)
		}

		out = append(out, &stateFile{
			StartBlock: blockNum,
			Source:     match[1],
		})
	}

	return
}
