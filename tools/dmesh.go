package tools

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	dmeshClient "github.com/dfuse-io/dmesh/client"
	"github.com/golang/protobuf/ptypes"
	tspb "github.com/golang/protobuf/ptypes/timestamp"
	pbdashboard "github.com/streamingfast/dlauncher/dashboard/pb"

	_ "github.com/dfuse-io/kvdb/store/badger"
	_ "github.com/dfuse-io/kvdb/store/bigkv"
	_ "github.com/dfuse-io/kvdb/store/tikv"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var dmeshCmd = &cobra.Command{Use: "dmesh", Short: "List current search peers in dmesh etcd only", RunE: dmeshE}

func init() {
	Cmd.AddCommand(dmeshCmd)

	dmeshCmd.PersistentFlags().String("dsn", "etcd://etcd.dmesh:2379/eos-mainnet", "Etcd connection string with namespace")
	dmeshCmd.PersistentFlags().String("service-version", "v3", "Dmesh service version")
}

func dmeshE(cmd *cobra.Command, args []string) (err error) {
	dsn := viper.GetString("dsn")
	if dsn == "" {
		return fmt.Errorf("A DSN is required to connect to etcd")
	}

	meshClient, err := dmeshClient.New(dsn)
	if err != nil {
		return fmt.Errorf("unable to create dmesh client with dsn %q: %w", dsn, err)
	}

	service := fmt.Sprintf("/%s/search", viper.GetString("service-version"))
	err = meshClient.Start(cmd.Context(), []string{service})
	if err != nil {
		return fmt.Errorf("unable to start dmesh client: %w", err)
	}

	// fmt.Println(fmt.Sprintf("Looking up search peers @ %q", fmt.Sprintf("%s%s", extractNamespace(dsn), service)))
	searchPeers := meshClient.Peers()
	sort.Slice(searchPeers, func(i, j int) bool {
		return searchPeers[i].TierLevel < searchPeers[j].TierLevel
	})

	if len(searchPeers) == 0 {
		fmt.Println("Did not find any search peers")
	}
	for _, peer := range searchPeers {
		err := printPeer(&pbdashboard.DmeshClient{
			Host:               peer.Host,
			Ready:              peer.Ready,
			Boot:               timeToProtoTimestamp(peer.Boot),
			ServesResolveForks: peer.ServesResolveForks,
			ServesReversible:   peer.ServesReversible,
			HasMovingHead:      peer.HasMovingHead,
			HasMovingTail:      peer.HasMovingTail,
			ShardSize:          peer.ShardSize,
			TierLevel:          peer.TierLevel,
			TailBlockNum:       peer.TailBlock,
			TailBlockId:        peer.TailBlockID,
			IrrBlockNum:        peer.IrrBlock,
			IrrBlockId:         peer.IrrBlockID,
			HeadBlockNum:       peer.HeadBlock,
			HeadBlockId:        peer.HeadBlockID,
		})
		if err != nil {
			return fmt.Errorf("unable to print peer: %w", err)
		}

	}
	return nil
}

func printPeer(peer *pbdashboard.DmeshClient) error {
	cnt, err := json.Marshal(peer)
	if err != nil {
		return err
	}
	fmt.Println(string(cnt))
	return nil
}

func timeToProtoTimestamp(t *time.Time) *tspb.Timestamp {
	out, _ := ptypes.TimestampProto(*t)
	return out
}

func extractNamespace(dsn string) string {
	chunks := strings.Split(dsn, "/")
	return chunks[len(chunks)-1]
}
