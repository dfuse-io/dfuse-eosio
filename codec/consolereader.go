// Copyright 2019 dfuse Platform Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package codec

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/eoscanada/eos-go"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

var supportedVersions = []string{"12", "13"}

type ConsoleReaderOption interface {
	apply(reader *ConsoleReader)
}

type consoleReaderOptionFunc func(reader *ConsoleReader)

func (f consoleReaderOptionFunc) apply(reader *ConsoleReader) {
	f(reader)
}

// LimitConsoleLength ensure that `Console` field on `pbcodec.ActionTrace` are
// never bigger than `maxCharacterCount` characters.
//
// This is sadly incomplete as failing deferred transaction can still log out of band
// via the standard nodeos logging mecanism.
func LimitConsoleLength(maxCharacterCount int) ConsoleReaderOption {
	return consoleReaderOptionFunc(func(reader *ConsoleReader) {
		reader.ctx.maxConsoleLengthInCharacter = maxCharacterCount
	})
}

// ConsoleReader is what reads the `nodeos` output directly. It builds
// up some LogEntry objects. See `LogReader to read those entries .
type ConsoleReader struct {
	src        io.Reader
	scanner    *bufio.Scanner
	close      func()
	readBuffer chan string
	done       chan interface{}

	ctx *parseCtx
}

// TODO: At some point, the interface of a ConsoleReader should be re-done.
//       Indeed, the `ConsoleReader` could simply receive each line already split
//       since the upstream caller is already doing this job it self. This way, we
//       would have a single split job instead of two. Only the upstream would split
//       the line and the console reader would simply process each line, one at a time.
func NewConsoleReader(reader io.Reader, opts ...ConsoleReaderOption) (*ConsoleReader, error) {
	l := &ConsoleReader{
		src:   reader,
		close: func() {},
		ctx:   newParseCtx(),
		done:  make(chan interface{}),
	}

	for _, opt := range opts {
		opt.apply(l)
	}

	l.setupScanner()
	return l, nil
}

func (l *ConsoleReader) setupScanner() {
	maxTokenSize := uint64(50 * 1024 * 1024)
	if maxBufferSize := os.Getenv("MINDREADER_MAX_TOKEN_SIZE"); maxBufferSize != "" {
		bs, err := strconv.ParseUint(maxBufferSize, 10, 64)
		if err != nil {
			zlog.Error("MINDREADER_MAX_TOKEN_SIZE is set but invalid parse uint", zap.Error(err))
		} else {
			zlog.Info("setting max_token_size from environment variable MINDREADER_MAX_TOKEN_SIZE", zap.Uint64("max_token_size", bs))
			maxTokenSize = bs
		}
	}
	buf := make([]byte, maxTokenSize)
	scanner := bufio.NewScanner(l.src)
	scanner.Buffer(buf, len(buf))
	l.scanner = scanner
	l.readBuffer = make(chan string, 2000)

	go func() {
		for l.scanner.Scan() {
			line := l.scanner.Text()
			if !strings.HasPrefix(line, "DMLOG ") {
				continue
			}
			l.readBuffer <- line
		}

		err := l.scanner.Err()
		if err != nil && err != io.EOF {
			zlog.Error("console read line scanner encountered an error", zap.Error(err))
		}

		close(l.readBuffer)
		close(l.done)
	}()
}

func (l *ConsoleReader) Done() <-chan interface{} {
	return l.done
}

func (l *ConsoleReader) Close() {
	l.close()
}

type parseCtx struct {
	abiDecoder     *ABIDecoder
	block          *pbcodec.Block
	activeBlockNum int64

	trx         *pbcodec.TransactionTrace
	creationOps []*creationOp

	maxConsoleLengthInCharacter int
}

func newParseCtx() *parseCtx {
	return &parseCtx{
		abiDecoder: newABIDecoder(),
		block:      &pbcodec.Block{},
		trx:        &pbcodec.TransactionTrace{},
	}
}

func (l *ConsoleReader) Read() (out interface{}, err error) {
	ctx := l.ctx

	for line := range l.readBuffer {
		line = line[6:]

		if traceEnabled {
			zlog.Debug("extracing deep mind data from line", zap.String("line", line))
		}

		// Order of conditions is based (approximately) on those that will appear more often
		switch {
		case strings.HasPrefix(line, "RAM_OP"):
			err = ctx.readRAMOp(line)

		case strings.HasPrefix(line, "CREATION_OP"):
			err = ctx.readCreationOp(line)

		case strings.HasPrefix(line, "DB_OP"):
			err = ctx.readDBOp(line)

		case strings.HasPrefix(line, "RLIMIT_OP"):
			err = ctx.readRlimitOp(line)

		case strings.HasPrefix(line, "TRX_OP"):
			err = ctx.readTrxOp(line)

		case strings.HasPrefix(line, "APPLIED_TRANSACTION"):
			err = ctx.readAppliedTransaction(line)

		case strings.HasPrefix(line, "TBL_OP"):
			err = ctx.readTableOp(line)

		case strings.HasPrefix(line, "PERM_OP"):
			err = ctx.readPermOp(line)

		case strings.HasPrefix(line, "DTRX_OP CREATE"):
			err = ctx.readCreateOrCancelDTrxOp("CREATE", line)

		case strings.HasPrefix(line, "DTRX_OP MODIFY_CREATE"):
			err = ctx.readCreateOrCancelDTrxOp("MODIFY_CREATE", line)

		case strings.HasPrefix(line, "DTRX_OP MODIFY_CANCEL"):
			err = ctx.readCreateOrCancelDTrxOp("MODIFY_CANCEL", line)

		case strings.HasPrefix(line, "RAM_CORRECTION_OP"):
			err = ctx.readRAMCorrectionOp(line)

		case strings.HasPrefix(line, "DTRX_OP PUSH_CREATE"):
			err = ctx.readCreateOrCancelDTrxOp("PUSH_CREATE", line)

		case strings.HasPrefix(line, "DTRX_OP CANCEL"):
			err = ctx.readCreateOrCancelDTrxOp("CANCEL", line)

		case strings.HasPrefix(line, "DTRX_OP FAILED"):
			err = ctx.readFailedDTrxOp(line)

		case strings.HasPrefix(line, "ACCEPTED_BLOCK"):
			block, err := ctx.readAcceptedBlock(line)
			if err != nil {
				return nil, l.formatError(line, err)
			}

			return block, nil

		case strings.HasPrefix(line, "START_BLOCK"):
			err = ctx.readStartBlock(line)

		case strings.HasPrefix(line, "FEATURE_OP ACTIVATE"):
			err = ctx.readFeatureOpActivate(line)

		case strings.HasPrefix(line, "FEATURE_OP PRE_ACTIVATE"):
			err = ctx.readFeatureOpPreActivate(line)

		case strings.HasPrefix(line, "SWITCH_FORK"):
			zlog.Info("fork signal, restarting state accumulation from beginning")
			ctx.resetBlock()

		case strings.HasPrefix(line, "ABIDUMP START"):
			err = ctx.readABIStart(line)
		case strings.HasPrefix(line, "ABIDUMP ABI"):
			err = ctx.readABIDump(line)
		case strings.HasPrefix(line, "ABIDUMP END"):
			//noop

		case strings.HasPrefix(line, "DEEP_MIND_VERSION"):
			err = ctx.readDeepmindVersion(line)

		default:
			zlog.Info("unknown log line", zap.String("line", line))
		}

		if err != nil {
			return nil, l.formatError(line, err)
		}
	}

	if l.scanner.Err() == nil {
		return nil, io.EOF
	}

	return nil, l.scanner.Err()
}

func (l *ConsoleReader) formatError(line string, err error) error {
	chunks := strings.SplitN(line, " ", 2)
	return fmt.Errorf("%s: %s (line %q)", chunks[0], err, line)
}

type creationOp struct {
	kind        string // ROOT, NOTIFY, CFA_INLINE, INLINE
	actionIndex int
}

func (ctx *parseCtx) resetBlock() {
	// The nodeos bootstrap phase at chain initialization happens before the first block is ever
	// produced. As such, those operations needs to be attached to initial block. Hence, let's
	// reset recorded ops only if a block existed previously.
	if ctx.activeBlockNum != 0 {
		ctx.resetTrx()
	}

	ctx.block = &pbcodec.Block{}
}

func (ctx *parseCtx) resetTrx() {
	ctx.trx = &pbcodec.TransactionTrace{}
	ctx.creationOps = nil
}

func (ctx *parseCtx) recordCreationOp(operation *creationOp) {
	ctx.creationOps = append(ctx.creationOps, operation)
}

func (ctx *parseCtx) recordDBOp(operation *pbcodec.DBOp) {
	ctx.trx.DbOps = append(ctx.trx.DbOps, operation)
}

func (ctx *parseCtx) recordDTrxOp(transaction *pbcodec.DTrxOp) {
	ctx.trx.DtrxOps = append(ctx.trx.DtrxOps, transaction)

	if transaction.Operation == pbcodec.DTrxOp_OPERATION_FAILED {
		ctx.revertOpsDueToFailedTransaction()
	}
}

func (ctx *parseCtx) recordFeatureOp(operation *pbcodec.FeatureOp) {
	ctx.trx.FeatureOps = append(ctx.trx.FeatureOps, operation)
}

func (ctx *parseCtx) recordPermOp(operation *pbcodec.PermOp) {
	ctx.trx.PermOps = append(ctx.trx.PermOps, operation)
}

func (ctx *parseCtx) recordRAMOp(operation *pbcodec.RAMOp) {
	ctx.trx.RamOps = append(ctx.trx.RamOps, operation)
}

func (ctx *parseCtx) recordRAMCorrectionOp(operation *pbcodec.RAMCorrectionOp) {
	ctx.trx.RamCorrectionOps = append(ctx.trx.RamCorrectionOps, operation)
}

func (ctx *parseCtx) recordRlimitOp(operation *pbcodec.RlimitOp) {
	if operation.IsGlobalKind() {
		ctx.block.RlimitOps = append(ctx.block.RlimitOps, operation)
	} else if operation.IsLocalKind() {
		ctx.trx.RlimitOps = append(ctx.trx.RlimitOps, operation)
	}
}

func (ctx *parseCtx) recordTableOp(operation *pbcodec.TableOp) {
	ctx.trx.TableOps = append(ctx.trx.TableOps, operation)
}

func (ctx *parseCtx) recordTrxOp(operation *pbcodec.TrxOp) {
	ctx.block.UnfilteredImplicitTransactionOps = append(ctx.block.UnfilteredImplicitTransactionOps, operation)
}

func (ctx *parseCtx) recordTransaction(trace *pbcodec.TransactionTrace) error {
	failedTrace := trace.FailedDtrxTrace
	if failedTrace != nil {
		// Having a `FailedDtrxTrace` means the `trace` we got is an `onerror` handler.
		// In this block, we perform all the logic to correctly record the `onerror`
		// handler trace and the actual deferred transaction trace that failed.

		// The deferred transaction removal RAM op needs to be attached to the failed trace, not the onerror handler
		ctx.trx.RamOps = ctx.transferDeferredRemovedRAMOp(ctx.trx.RamOps, failedTrace)

		// The only possibilty to have failed deferred trace, is when the deferred execution
		// resulted in a subjetive failure, which is really a soft fail. So, when the receipt is
		// not set, let's re-create it here with soft fail status only.
		if failedTrace.Receipt == nil {
			failedTrace.Receipt = &pbcodec.TransactionReceiptHeader{
				Status: pbcodec.TransactionStatus_TRANSACTIONSTATUS_SOFTFAIL,
			}
		}

		// We add the failed deferred trace first, before the "real" trace (the `onerror` handler)
		// since it was ultimetaly ran first. There is no ops possible on the trace expect the
		// transferred RAM op, so it's all good to attach it directly.
		ctx.block.UnfilteredTransactionTraces = append(ctx.block.UnfilteredTransactionTraces, failedTrace)

		if err := ctx.abiDecoder.processTransaction(failedTrace); err != nil {
			return fmt.Errorf("abi decoding failed trace: %w", err)
		}

		// When the `onerror` `trace` receipt is `soft_fail`, it means the `onerror` handler
		// succeed. But when it's `hard_fail` it means either no handler was defined, or the one
		// defined failed to execute properly. So in the `hard_fail` case, let's reset all ops.
		// However, we do keep `RLimitOps` as they seems to be billed regardeless of transaction
		// execution status
		if trace.Receipt == nil || trace.Receipt.Status == pbcodec.TransactionStatus_TRANSACTIONSTATUS_HARDFAIL {
			ctx.revertOpsDueToFailedTransaction()
		}
	}

	// All this stiching of ops into trace must be performed after `if` because the if can revert them all
	creationTreeRoots, err := computeCreationTree(ctx.creationOps)
	if err != nil {
		return fmt.Errorf("compute creation tree: %s", err)
	}

	trace.CreationTree = CreationTreeToDEOS(toFlatTree(creationTreeRoots...))
	trace.DtrxOps = ctx.trx.DtrxOps
	trace.DbOps = ctx.trx.DbOps
	trace.FeatureOps = ctx.trx.FeatureOps
	trace.PermOps = ctx.trx.PermOps
	trace.RamOps = ctx.trx.RamOps
	trace.RamCorrectionOps = ctx.trx.RamCorrectionOps
	trace.RlimitOps = ctx.trx.RlimitOps
	trace.TableOps = ctx.trx.TableOps

	ctx.block.UnfilteredTransactionTraces = append(ctx.block.UnfilteredTransactionTraces, trace)

	if err := ctx.abiDecoder.processTransaction(trace); err != nil {
		return fmt.Errorf("abi decoding trace: %w", err)
	}

	ctx.resetTrx()
	return nil
}

func (ctx *parseCtx) revertOpsDueToFailedTransaction() {
	// We must keep the deferred removal, as this RAM changed is **not** reverted by nodeos, unlike all other ops
	// as well as the RLimitOps, which happens at a location that does not revert.
	toRestoreRlimitOps := ctx.trx.RlimitOps

	var deferredRemovalRAMOp *pbcodec.RAMOp
	for _, op := range ctx.trx.RamOps {
		if op.Namespace == pbcodec.RAMOp_NAMESPACE_DEFERRED_TRX && op.Action == pbcodec.RAMOp_ACTION_REMOVE {
			deferredRemovalRAMOp = op
			break
		}
	}

	ctx.resetTrx()
	ctx.trx.RlimitOps = toRestoreRlimitOps
	if deferredRemovalRAMOp != nil {
		ctx.trx.RamOps = []*pbcodec.RAMOp{deferredRemovalRAMOp}
	}
}

func (ctx *parseCtx) transferDeferredRemovedRAMOp(initialRAMOps []*pbcodec.RAMOp, target *pbcodec.TransactionTrace) (filteredRAMOps []*pbcodec.RAMOp) {
	for _, ramOp := range initialRAMOps {
		if ramOp.Namespace == pbcodec.RAMOp_NAMESPACE_DEFERRED_TRX && ramOp.Action == pbcodec.RAMOp_ACTION_REMOVE {
			target.RamOps = append(target.RamOps, ramOp)
		} else {
			filteredRAMOps = append(filteredRAMOps, ramOp)
		}
	}

	return filteredRAMOps
}

// Line format:
//   START_BLOCK ${block_num}
func (ctx *parseCtx) readStartBlock(line string) error {
	chunks := strings.Split(line, " ")
	if len(chunks) != 2 {
		return fmt.Errorf("expected 2 fields, got %d", len(chunks))
	}

	blockNum, err := strconv.ParseInt(chunks[1], 10, 64)
	if err != nil {
		return fmt.Errorf("block_num not a valid string, got: %q", chunks[1])
	}

	ctx.resetBlock()
	ctx.activeBlockNum = blockNum

	if err := ctx.abiDecoder.startBlock(uint64(blockNum)); err != nil {
		return fmt.Errorf("abi decoder: %w", err)
	}

	return nil
}

// Line format:
//   ACCEPTED_BLOCK ${block_num} ${block_state_hex}
func (ctx *parseCtx) readAcceptedBlock(line string) (*pbcodec.Block, error) {
	chunks := strings.SplitN(line, " ", 3)
	if len(chunks) != 3 {
		return nil, fmt.Errorf("expected 3 fields, got %d", len(chunks))
	}

	blockNum, err := strconv.ParseInt(chunks[1], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("block_num not a valid string, got: %q", chunks[1])
	}

	if ctx.activeBlockNum != blockNum {
		return nil, fmt.Errorf("block_num %d doesn't match the active block num (%d)", blockNum, ctx.activeBlockNum)
	}

	blockStateHex, err := hex.DecodeString(chunks[2])
	if err != nil {
		return nil, fmt.Errorf("unable to decode block %d state hex: %w", blockNum, err)
	}

	blockState := &eos.BlockState{}
	err = unmarshalBinary(blockStateHex, blockState)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling binary block state: %w", err)
	}

	signedBlock := blockState.SignedBlock

	ctx.block.Id = blockState.BlockID.String()
	ctx.block.Number = blockState.BlockNum
	// Version 1: Added the total counts (ExecutedInputActionCount, ExecutedTotalActionCount,
	// TransactionCount, TransactionTraceCount)
	ctx.block.Version = 1
	ctx.block.Header = BlockHeaderToDEOS(&signedBlock.BlockHeader)
	ctx.block.BlockExtensions = ExtensionsToDEOS(signedBlock.BlockExtensions)
	ctx.block.DposIrreversibleBlocknum = blockState.DPoSIrreversibleBlockNum
	ctx.block.DposProposedIrreversibleBlocknum = blockState.DPoSProposedIrreversibleBlockNum
	ctx.block.Validated = blockState.Validated
	ctx.block.BlockrootMerkle = BlockrootMerkleToDEOS(blockState.BlockrootMerkle)
	ctx.block.ProducerToLastProduced = ProducerToLastProducedToDEOS(blockState.ProducerToLastProduced)
	ctx.block.ProducerToLastImpliedIrb = ProducerToLastImpliedIrbToDEOS(blockState.ProducerToLastImpliedIRB)
	ctx.block.ActivatedProtocolFeatures = ActivatedProtocolFeaturesToDEOS(blockState.ActivatedProtocolFeatures)
	ctx.block.ProducerSignature = signedBlock.ProducerSignature.String()

	ctx.block.ConfirmCount = make([]uint32, len(blockState.ConfirmCount))
	for i, count := range blockState.ConfirmCount {
		ctx.block.ConfirmCount[i] = uint32(count)
	}

	if blockState.PendingSchedule != nil {
		ctx.block.PendingSchedule = PendingScheduleToDEOS(blockState.PendingSchedule)
	}

	/// Specific versions handling

	blockSigningKey := blockState.BlockSigningKeyV1
	schedule := blockState.ActiveSchedule
	signingAuthority := blockState.ValidBlockSigningAuthorityV2

	// Only in EOSIO 1.x
	if blockSigningKey != nil {
		ctx.block.BlockSigningKey = blockSigningKey.String()
	}

	if schedule.V1 != nil {
		ctx.block.ActiveScheduleV1 = ProducerScheduleToDEOS(schedule.V1)
	}

	// Only in EOSIO 2.x
	if signingAuthority != nil {
		ctx.block.ValidBlockSigningAuthorityV2 = BlockSigningAuthorityToDEOS(signingAuthority)
	}

	if schedule.V2 != nil {
		ctx.block.ActiveScheduleV2 = ProducerAuthorityScheduleToDEOS(schedule.V2)
	}

	// End (versions)

	ctx.block.UnfilteredTransactionCount = uint32(len(signedBlock.Transactions))
	for idx, transaction := range signedBlock.Transactions {
		deosTransaction := TransactionReceiptToDEOS(&transaction)
		deosTransaction.Index = uint64(idx)

		ctx.block.UnfilteredTransactions = append(ctx.block.UnfilteredTransactions, deosTransaction)
	}

	ctx.block.UnfilteredTransactionTraceCount = uint32(len(ctx.block.UnfilteredTransactionTraces))
	for idx, t := range ctx.block.UnfilteredTransactionTraces {
		t.Index = uint64(idx)
		t.BlockTime = ctx.block.Header.Timestamp
		t.ProducerBlockId = ctx.block.Id
		t.BlockNum = uint64(ctx.block.Number)

		for _, actionTrace := range t.ActionTraces {
			ctx.block.UnfilteredExecutedTotalActionCount++
			if actionTrace.IsInput() {
				ctx.block.UnfilteredExecutedInputActionCount++
			}
		}
	}

	block := ctx.block

	zlog.Debug("blocking until abi decoder has decoded every transaction pushed to it")
	err = ctx.abiDecoder.endBlock(ctx.block)
	if err != nil {
		return nil, fmt.Errorf("abi decoding post-process failed: %w", err)
	}

	zlog.Debug("abi decoder terminated all decoding operations, resetting block")
	ctx.resetBlock()
	return block, nil
}

// Line format:
//   APPLIED_TRANSACTION ${block_num} ${trace_hex}
func (ctx *parseCtx) readAppliedTransaction(line string) error {
	chunks := strings.SplitN(line, " ", 3)
	if len(chunks) != 3 {
		return fmt.Errorf("expected 3 fields, got %d", len(chunks))
	}

	blockNum, err := strconv.ParseInt(chunks[1], 10, 64)
	if err != nil {
		return fmt.Errorf("block_num not a valid number, got: %q", chunks[1])
	}

	if ctx.activeBlockNum != blockNum {
		return fmt.Errorf("saw transactions from block %d while active block is %d", blockNum, ctx.activeBlockNum)
	}

	trxTraceHex, err := hex.DecodeString(chunks[2])
	if err != nil {
		return fmt.Errorf("unable to decode transaction trace hex at block num %d: %w", blockNum, err)
	}

	trxTrace := &eos.TransactionTrace{}
	err = unmarshalBinary(trxTraceHex, trxTrace)
	if err != nil {
		return fmt.Errorf("unmarshalling binary transaction trace: %w", err)
	}

	var options []conversionOption
	if ctx.maxConsoleLengthInCharacter > 0 {
		options = append(options, limitConsoleLengthConversionOption(ctx.maxConsoleLengthInCharacter))
	}

	return ctx.recordTransaction(TransactionTraceToDEOS(trxTrace, options...))
}

// Line formats:
//  CREATION_OP ROOT ${action_id}
//  CREATION_OP NOTIFY ${action_id}
//  CREATION_OP INLINE ${action_id}
//  CREATION_OP CFA_INLINE ${action_id}
func (ctx *parseCtx) readCreationOp(line string) error {
	chunks := strings.SplitN(line, " ", 3)
	if len(chunks) != 3 {
		return fmt.Errorf("expected 3 fields, got %d", len(chunks))
	}

	kind := chunks[1]
	if kind != "ROOT" && kind != "NOTIFY" && kind != "INLINE" && kind != "CFA_INLINE" {
		return fmt.Errorf("kind must be one of ROOT, NOTIFY, CFA_INLINE or INLINE, got: %q", kind)
	}

	actionIndex, err := strconv.Atoi(chunks[2])
	if err != nil {
		return fmt.Errorf("action_index is not a valid number, got: %q", chunks[2])
	}

	ctx.recordCreationOp(&creationOp{
		kind: kind,
		// FIXME: this index is 0-based, whereas `action_ordinal` is 1-based, where 0 means a virtual root node.
		// This is a BIG problem as now we unpack the traces and simply keep that `action_ordinal` field.. so in `eosws`, we need to re-map all of this together.
		// Perhaps we can simply ditch all of this since we'll have the `closest unnotified ancestor`,.. and we could *NOT* compute our own thing anymore.. and always use theirs..
		// then simply re-map their model into ours at the edge (in `eosws`).
		actionIndex: actionIndex,
	})

	return nil
}

// Line formats:
//   DB_OP INS ${action_id} ${payer} ${table_code} ${scope} ${table_name} ${primkey} ${ndata}
//   DB_OP UPD ${action_id} ${opayer}:${npayer} ${table_code} ${scope} ${table_name} ${primkey} ${odata}:${ndata}
//   DB_OP REM ${action_id} ${payer} ${table_code} ${scope} ${table_name} ${primkey} ${odata}
func (ctx *parseCtx) readDBOp(line string) error {
	chunks := strings.SplitN(line, " ", 9)
	if len(chunks) != 9 {
		return fmt.Errorf("expected 9 fields, got %d", len(chunks))
	}

	actionIndex, err := strconv.Atoi(chunks[2])
	if err != nil {
		return fmt.Errorf("action_index is not a valid number, got: %q", chunks[2])
	}

	opString := chunks[1]

	op := pbcodec.DBOp_OPERATION_UNKNOWN
	var oldData, newData string
	var oldPayer, newPayer string
	switch opString {
	case "INS":
		op = pbcodec.DBOp_OPERATION_INSERT
		newData = chunks[8]
		newPayer = chunks[3]
	case "UPD":
		op = pbcodec.DBOp_OPERATION_UPDATE

		dataChunks := strings.SplitN(chunks[8], ":", 2)
		if len(dataChunks) != 2 {
			return fmt.Errorf("should have old and new data in field 8, found only one")
		}

		oldData = dataChunks[0]
		newData = dataChunks[1]

		payerChunks := strings.SplitN(chunks[3], ":", 2)
		if len(payerChunks) != 2 {
			return fmt.Errorf("should have two payers in field 3, separated by a ':', found only one")
		}

		oldPayer = payerChunks[0]
		newPayer = payerChunks[1]
	case "REM":
		op = pbcodec.DBOp_OPERATION_REMOVE
		oldData = chunks[8]
		oldPayer = chunks[3]
	default:
		return fmt.Errorf("unknown operation: %q", opString)
	}

	var oldBytes, newBytes []byte
	if len(oldData) != 0 {
		oldBytes, err = hex.DecodeString(oldData)
		if err != nil {
			return fmt.Errorf("couldn't decode old_data: %s", err)
		}
	}

	if len(newData) != 0 {
		newBytes, err = hex.DecodeString(newData)
		if err != nil {
			return fmt.Errorf("couldn't decode new_data: %s", err)
		}
	}

	ctx.recordDBOp(&pbcodec.DBOp{
		Operation:   op,
		ActionIndex: uint32(actionIndex),
		OldPayer:    oldPayer,
		NewPayer:    newPayer,
		Code:        chunks[4],
		Scope:       chunks[5],
		TableName:   chunks[6],
		PrimaryKey:  chunks[7],
		OldData:     oldBytes,
		NewData:     newBytes,
	})

	return nil
}

// Line formats:
//   DTRX_OP MODIFY_CANCEL ${action_id} ${sender} ${sender_id} ${payer} ${published} ${delay} ${expiration} ${trx_id} ${trx}
//   DTRX_OP MODIFY_CREATE ${action_id} ${sender} ${sender_id} ${payer} ${published} ${delay} ${expiration} ${trx_id} ${trx}
//   DTRX_OP CREATE        ${action_id} ${sender} ${sender_id} ${payer} ${published} ${delay} ${expiration} ${trx_id} ${trx}
//   DTRX_OP CANCEL        ${action_id} ${sender} ${sender_id} ${payer} ${published} ${delay} ${expiration} ${trx_id} ${trx}
//   DTRX_OP PUSH_CREATE   ${action_id} ${sender} ${sender_id} ${payer} ${published} ${delay} ${expiration} ${trx_id} ${trx}
func (ctx *parseCtx) readCreateOrCancelDTrxOp(tag string, line string) error {
	chunks := strings.SplitN(line, " ", 11)
	if len(chunks) != 11 {
		return fmt.Errorf("expected 11 fields, got %d", len(chunks))
	}

	opString := chunks[1]
	rawOp, ok := pbcodec.DTrxOp_Operation_value["OPERATION_"+opString]
	if !ok {
		return fmt.Errorf("operation %q unknown", opString)
	}

	op := pbcodec.DTrxOp_Operation(rawOp)

	actionIndex, err := strconv.Atoi(chunks[2])
	if err != nil {
		return fmt.Errorf("action_index is not a valid number, got: %q", chunks[2])
	}

	trxHex, err := hex.DecodeString(chunks[10])
	if err != nil {
		return fmt.Errorf("unable to decode signed transaction hex: %w", err)
	}

	var signedTrx *eos.SignedTransaction
	if op == pbcodec.DTrxOp_OPERATION_PUSH_CREATE {
		signedTrx = new(eos.SignedTransaction)
		err = unmarshalBinary(trxHex, signedTrx)
		if err != nil {
			return fmt.Errorf("unmarshal binary signed transaction: %w", err)
		}
	} else {
		trx := &eos.Transaction{}
		err = unmarshalBinary(trxHex, trx)
		if err != nil {
			return fmt.Errorf("unmarshal binary transaction: %w", err)
		}

		signedTrx = &eos.SignedTransaction{
			Transaction: trx,
		}
	}

	ctx.recordDTrxOp(&pbcodec.DTrxOp{
		Operation:     op,
		ActionIndex:   uint32(actionIndex),
		Sender:        chunks[3],
		SenderId:      chunks[4],
		Payer:         chunks[5],
		PublishedAt:   chunks[6],
		DelayUntil:    chunks[7],
		ExpirationAt:  chunks[8],
		TransactionId: chunks[9],
		Transaction:   SignedTransactionToDEOS(signedTrx),
	})

	return nil
}

// Line format:
//   DTRX_OP FAILED ${action_id}
func (ctx *parseCtx) readFailedDTrxOp(line string) error {
	chunks := strings.SplitN(line, " ", 3)
	if len(chunks) != 3 {
		return fmt.Errorf("expected 3 fields, got %d", len(chunks))
	}

	actionIndex, err := strconv.Atoi(chunks[2])
	if err != nil {
		return fmt.Errorf("action_index is not a valid number, got: %q", chunks[2])
	}

	ctx.recordDTrxOp(&pbcodec.DTrxOp{
		Operation:   pbcodec.DTrxOp_OPERATION_FAILED,
		ActionIndex: uint32(actionIndex),
	})

	return nil
}

// Line formats:
//   FEATURE_OP ACTIVATE ${feature_digest} ${feature}
func (ctx *parseCtx) readFeatureOpActivate(line string) error {
	chunks := strings.SplitN(line, " ", 4)
	if len(chunks) != 4 {
		return fmt.Errorf("expected 4 fields, got %d", len(chunks))
	}

	feature := &pbcodec.Feature{}
	err := json.Unmarshal(json.RawMessage(chunks[3]), &feature)
	if err != nil {
		return fmt.Errorf("unmashall new feature data: %s", err)
	}

	ctx.recordFeatureOp(&pbcodec.FeatureOp{
		Kind:          chunks[1],
		FeatureDigest: chunks[2],
		Feature:       feature,
	})

	return nil
}

// Line formats:
//   FEATURE_OP PRE_ACTIVATE ${action_id} ${feature_digest} ${feature}
func (ctx *parseCtx) readFeatureOpPreActivate(line string) error {
	chunks := strings.SplitN(line, " ", 5)
	if len(chunks) != 5 {
		return fmt.Errorf("expected 5 fields, got %d", len(chunks))
	}

	actionIndex, err := strconv.Atoi(chunks[2])
	if err != nil {
		return fmt.Errorf("action_index is not a valid number, got: %q", chunks[2])
	}

	feature := &pbcodec.Feature{}
	err = json.Unmarshal(json.RawMessage(chunks[4]), &feature)
	if err != nil {
		return fmt.Errorf("unmashall new feature data: %s", err)
	}

	ctx.recordFeatureOp(&pbcodec.FeatureOp{
		Kind:          chunks[1],
		ActionIndex:   uint32(actionIndex),
		FeatureDigest: chunks[3],
		Feature:       feature,
	})
	return nil
}

// Line formats: (the `[...]` represents optional fields)
//   PERM_OP INS ${action_id} [${permission_id}] ${data}
//   PERM_OP UPD ${action_id} [${permission_id}] ${data}
//   PERM_OP REM ${action_id} [${permission_id}] ${data} <-- {"old": <old>, "new": <new>}
func (ctx *parseCtx) readPermOp(line string) error {
	chunks, err := splitNToM(line, 4, 5)
	if err != nil {
		return err
	}

	actionIndex, err := strconv.Atoi(chunks[2])
	if err != nil {
		return fmt.Errorf("action_index is not a valid number, got: %q", chunks[2])
	}

	opString := chunks[1]
	dataChunk := chunks[3]
	var permissionID uint64

	// A `PERM_OP` with 5 fields have ["permission_id"] field in index #3 set and data chunk is actually index #4
	if len(chunks) == 5 {
		permissionID, err = strconv.ParseUint(chunks[3], 10, 64)
		if err != nil {
			return fmt.Errorf("permission_id is not a valid number, got: %q", chunks[3])
		}
		dataChunk = chunks[4]
	}

	op := pbcodec.PermOp_OPERATION_UNKNOWN
	var oldData, newData []byte

	switch opString {
	case "INS":
		op = pbcodec.PermOp_OPERATION_INSERT
		newData = []byte(dataChunk)

	case "UPD":
		op = pbcodec.PermOp_OPERATION_UPDATE

		oldJSONResult := gjson.Get(dataChunk, "old")
		if !oldJSONResult.Exists() {
			return fmt.Errorf("a PERM_OP UPD should JSON data should have an 'old' field, found none in: %q", dataChunk)
		}

		newJSONResult := gjson.Get(dataChunk, "new")
		if !newJSONResult.Exists() {
			return fmt.Errorf("a PERM_OP UPD should JSON data should have an 'new' field, found none in: %q", dataChunk)
		}

		oldData = []byte(oldJSONResult.Raw)
		newData = []byte(newJSONResult.Raw)

	case "REM":
		op = pbcodec.PermOp_OPERATION_REMOVE

		oldData = []byte(dataChunk)

	default:
		return fmt.Errorf("unknown PERM_OP op: %q", opString)
	}

	permOp := &pbcodec.PermOp{
		Operation:   op,
		ActionIndex: uint32(actionIndex),
	}

	if len(newData) > 0 {
		newPerm := &permissionObject{}
		err = json.Unmarshal(newData, &newPerm)
		if err != nil {
			return fmt.Errorf("unmashal new perm data: %s", err)
		}

		permOp.NewPerm = newPerm.ToProto()
		permOp.NewPerm.Id = permissionID
	}

	if len(oldData) > 0 {
		oldPerm := &permissionObject{}
		err = json.Unmarshal(oldData, &oldPerm)
		if err != nil {
			return fmt.Errorf("unmashal old perm data: %s", err)
		}

		permOp.OldPerm = oldPerm.ToProto()
		permOp.OldPerm.Id = permissionID

	}

	ctx.recordPermOp(permOp)

	return nil
}

// Line format:
//   RAM_OP ${action_index} ${unique_key} ${namespace} ${action} ${legacy_tag} ${payer} ${new_usage} ${delta}
func (ctx *parseCtx) readRAMOp(line string) error {
	chunks := strings.SplitN(line, " ", 9)
	if len(chunks) != 9 {
		return fmt.Errorf("expected 9 fields, got %d", len(chunks))
	}

	actionIndex, err := strconv.Atoi(chunks[1])
	if err != nil {
		return fmt.Errorf("action_index is not a valid number, got: %q", chunks[1])
	}

	namespaceString := chunks[3]
	namespace, ok := pbcodec.RAMOp_Namespace_value["NAMESPACE_"+strings.ToUpper(namespaceString)]
	if !ok {
		return fmt.Errorf("namespace %q unknown", namespaceString)
	}

	actionString := chunks[4]
	action, ok := pbcodec.RAMOp_Action_value["ACTION_"+strings.ToUpper(actionString)]
	if !ok {
		return fmt.Errorf("action %q unknown", actionString)
	}

	operationString := chunks[5]
	operation, ok := pbcodec.RAMOp_Operation_value["OPERATION_"+strings.ToUpper(operationString)]
	if !ok {
		return fmt.Errorf("operation %q unknown", operationString)
	}

	usage, err := strconv.ParseInt(chunks[7], 10, 64)
	if err != nil {
		return fmt.Errorf("usage is not a valid number, got: %q", chunks[4])
	}

	delta, err := strconv.ParseInt(chunks[8], 10, 64)
	if err != nil {
		return fmt.Errorf("delta is not a valid number, got: %q", chunks[5])
	}

	ctx.recordRAMOp(&pbcodec.RAMOp{
		ActionIndex: uint32(actionIndex),
		UniqueKey:   chunks[2],
		Namespace:   pbcodec.RAMOp_Namespace(namespace),
		Action:      pbcodec.RAMOp_Action(action),
		Operation:   pbcodec.RAMOp_Operation(operation),
		Payer:       chunks[6],
		Usage:       uint64(usage),
		Delta:       int64(delta),
	})
	return nil
}

// Line format:
//  Version 12
//    DEEP_MIND_VERSION ${major_version}
//
//  Version 13
//    DEEP_MIND_VERSION ${major_version} ${minor_version}
func (ctx *parseCtx) readDeepmindVersion(line string) error {
	chunks, err := splitNToM(line, 2, 3)
	if err != nil {
		return err
	}

	majorVersion := chunks[1]
	if !inSupportedVersion(majorVersion) {
		return fmt.Errorf("deep mind reported version %s, but this reader supports only %s", majorVersion, strings.Join(supportedVersions, ", "))
	}

	zlog.Info("read deep mind version", zap.String("major_version", majorVersion))

	return nil
}

func inSupportedVersion(majorVersion string) bool {
	for _, supportedVersion := range supportedVersions {
		if majorVersion == supportedVersion {
			return true
		}
	}

	return false
}

// Line format:
//  Version 12
//    ABIDUMP START
//
//  Version 13
//    ABIDUMP START ${block_num} ${global_sequence_num}
func (ctx *parseCtx) readABIStart(line string) error {
	chunks := strings.SplitN(line, " ", -1)

	var logFields []zap.Field
	switch len(chunks) {
	case 2: // Version 12
		break
	case 4: // Version 13
		blockNum, err := strconv.Atoi(chunks[2])
		if err != nil {
			return fmt.Errorf("block_num is not a valid number, got: %q", chunks[2])
		}

		globalSequence, err := strconv.Atoi(chunks[3])
		if err != nil {
			return fmt.Errorf("global_sequence_num is not a valid number, got: %q", chunks[3])
		}

		logFields = append(logFields, zap.Int("block_num", blockNum), zap.Int("global_sequence", globalSequence))
	default:
		return fmt.Errorf("expected to have either %d or %d fields, got %d", 2, 4, len(chunks))
	}

	zlog.Info("read ABI start marker", logFields...)
	ctx.abiDecoder.resetCache()
	return nil
}

// Line format:
//  Version 12
//    ABIDUMP ABI ${block_num} ${contract} ${base64_abi}
//
//  Version 13
//    ABIDUMP ABI ${contract} ${base64_abi}
func (ctx *parseCtx) readABIDump(line string) error {
	chunks, err := splitNToM(line, 4, 5)
	if err != nil {
		return err
	}

	var contract, rawABI string
	switch len(chunks) {
	case 5: // Version 12
		contract = chunks[3]
		rawABI = chunks[4]

	case 4: // Version 13
		contract = chunks[2]
		rawABI = chunks[3]
	}

	if traceEnabled {
		zlog.Debug("read initial ABI for contract", zap.String("contract", contract))
	}

	return ctx.abiDecoder.addInitialABI(contract, rawABI)
}

// Line format:
//   RAM_CORRECTION_OP ${action_id} ${correction_id} ${unique_key} ${payer} ${delta}
func (ctx *parseCtx) readRAMCorrectionOp(line string) error {
	chunks := strings.SplitN(line, " ", 6)
	if len(chunks) != 6 {
		return fmt.Errorf("expected 6 fields, got %d", len(chunks))
	}

	// We assume ${action_id} will always be 0, since called from onblock, so that's why we do not process it

	delta, err := strconv.ParseInt(chunks[5], 10, 64)
	if err != nil {
		return fmt.Errorf("delta not a valid number, got: %q", chunks[5])
	}

	ctx.recordRAMCorrectionOp(&pbcodec.RAMCorrectionOp{
		CorrectionId: chunks[2],
		UniqueKey:    chunks[3],
		Payer:        chunks[4],
		Delta:        int64(delta),
	})
	return nil
}

// Line formats:
//   RLIMIT_OP CONFIG         INS ${data}
//   RLIMIT_OP CONFIG         UPD ${data}
//   RLIMIT_OP STATE          INS ${data}
//   RLIMIT_OP STATE          UPD ${data}
//   RLIMIT_OP ACCOUNT_LIMITS INS ${data}
//   RLIMIT_OP ACCOUNT_LIMITS UPD ${data}
//   RLIMIT_OP ACCOUNT_USAGE  INS ${data}
//   RLIMIT_OP ACCOUNT_USAGE  UPD ${data}
func (ctx *parseCtx) readRlimitOp(line string) error {
	chunks := strings.SplitN(line, " ", 4)
	if len(chunks) != 4 {
		return fmt.Errorf("expected 4 fields, got %d", len(chunks))
	}

	kindString := chunks[1]
	operationString := chunks[2]

	operation := pbcodec.RlimitOp_OPERATION_UNKNOWN
	switch operationString {
	case "INS":
		operation = pbcodec.RlimitOp_OPERATION_INSERT
	case "UPD":
		operation = pbcodec.RlimitOp_OPERATION_UPDATE
	default:
		return fmt.Errorf("operation %q is unknown", operationString)
	}

	op := &pbcodec.RlimitOp{Operation: operation}
	data := json.RawMessage(chunks[3])

	switch kindString {
	case "CONFIG":
		obj := &rlimitConfig{}
		err := json.Unmarshal(data, &obj)
		if err != nil {
			return fmt.Errorf("marshaling config: %s", err)
		}

		op.Kind = obj.ToProto()

	case "STATE":
		obj := &rlimitState{}
		err := json.Unmarshal(data, &obj)
		if err != nil {
			return fmt.Errorf("marshaling state: %s", err)
		}

		op.Kind = obj.ToProto()

	case "ACCOUNT_LIMITS":
		obj := &rlimitAccountLimits{}
		err := json.Unmarshal(data, &obj)
		if err != nil {
			return fmt.Errorf("marshaling account limits: %s", err)
		}

		op.Kind = obj.ToProto()

	case "ACCOUNT_USAGE":
		obj := &rlimitAccountUsage{}
		err := json.Unmarshal(data, &obj)
		if err != nil {
			return fmt.Errorf("marshaling account usage: %s", err)
		}

		op.Kind = obj.ToProto()

	default:
		return fmt.Errorf("unknown kind: %q", kindString)
	}

	ctx.recordRlimitOp(op)

	return nil
}

// Line formats:
//   TBL_OP INS ${action_id} ${code} ${scope} ${table} ${payer}
//   TBL_OP REM ${action_id} ${code} ${scope} ${table} ${payer}
func (ctx *parseCtx) readTableOp(line string) error {
	chunks := strings.SplitN(line, " ", 7)
	if len(chunks) != 7 {
		return fmt.Errorf("expected 7 fields, got %d", len(chunks))
	}

	actionIndex, err := strconv.Atoi(chunks[2])
	if err != nil {
		return fmt.Errorf("action_index is not a valid number, got: %q", chunks[2])
	}

	opString := chunks[1]
	op := pbcodec.TableOp_OPERATION_UNKNOWN
	switch opString {
	case "INS":
		op = pbcodec.TableOp_OPERATION_INSERT
	case "REM":
		op = pbcodec.TableOp_OPERATION_REMOVE
	default:
		return fmt.Errorf("unknown kind: %q", opString)
	}

	ctx.recordTableOp(&pbcodec.TableOp{
		Operation:   op,
		ActionIndex: uint32(actionIndex),
		Payer:       chunks[6],
		Code:        chunks[3],
		Scope:       chunks[4],
		TableName:   chunks[5],
	})

	return nil
}

// Line formats:
//   TRX_OP CREATE onblock|onerror ${id} ${trx}
func (ctx *parseCtx) readTrxOp(line string) error {
	chunks := strings.SplitN(line, " ", 5)
	if len(chunks) != 5 {
		return fmt.Errorf("expected 5 fields, got %d", len(chunks))
	}

	opString := chunks[1]
	op := pbcodec.TrxOp_OPERATION_UNKNOWN
	switch opString {
	case "CREATE":
		op = pbcodec.TrxOp_OPERATION_CREATE
	default:
		return fmt.Errorf("unknown kind: %q", opString)
	}

	name := chunks[2]
	trxID := chunks[3]

	trxHex, err := hex.DecodeString(chunks[4])
	if err != nil {
		return fmt.Errorf("unable to decode signed transaction %s hex: %w", trxID, err)
	}

	trx := &eos.SignedTransaction{}
	err = unmarshalBinary(trxHex, trx)
	if err != nil {
		return fmt.Errorf("unmarshal binary signed transaction %s: %w", trxID, err)
	}

	ctx.recordTrxOp(&pbcodec.TrxOp{
		Operation:     op,
		Name:          name,  // "onblock" or "onerror"
		TransactionId: trxID, // the hash of the transaction
		Transaction:   SignedTransactionToDEOS(trx),
	})

	return nil
}

func unmarshalBinary(data []byte, v interface{}) error {
	decoder := eos.NewDecoder(data)
	decoder.DecodeActions(false)
	decoder.DecodeP2PMessage(false)

	return decoder.Decode(v)
}

func splitNToM(line string, min, max int) ([]string, error) {
	chunks := strings.SplitN(line, " ", -1)
	if len(chunks) < min || len(chunks) > max {
		return nil, fmt.Errorf("expected between %d to %d fields (inclusively), got %d", min, max, len(chunks))
	}

	return chunks, nil
}
