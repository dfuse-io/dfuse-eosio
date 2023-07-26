package injector

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/dfuse-io/dfuse-eosio/accounthist/metrics"

	"github.com/streamingfast/dmetrics"

	"github.com/dfuse-io/dfuse-eosio/accounthist"

	pbaccounthist "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/accounthist/v1"
	"github.com/golang/protobuf/proto"
	"github.com/streamingfast/bstream"
	"github.com/streamingfast/dstore"
	"github.com/streamingfast/kvdb/store"
	"github.com/streamingfast/shutter"
	"go.uber.org/zap"
)

type Injector struct {
	*shutter.Shutter

	ShardNum            byte // 0 is live
	flushBlocksInterval uint64
	BlockFilter         func(blk *bstream.Block) error
	MaxEntries          uint64
	cacheSeqData        map[string]accounthist.SequenceData

	blocksStore dstore.Store
	KvStore     store.KVStore
	source      bstream.Source
	rwCache     *RWCache

	startBlockNum uint64
	stopBlockNum  uint64

	tracker               *bstream.Tracker
	startedFromCheckpoint bool
	lastCheckpoint        *pbaccounthist.ShardCheckpoint

	facetFactory accounthist.FacetFactory

	lastWrittenBlock    *lastWrittenBlock
	currentBatchMetrics blockBatchMetrics
	headBlockTimeDrift  *dmetrics.HeadTimeDrift
	headBlockNumber     *dmetrics.HeadBlockNum
}

func NewInjector(
	kvdb store.KVStore,
	blocksStore dstore.Store,
	blockFilter func(blk *bstream.Block) error,
	shardNum byte,
	maxEntries uint64,
	flushBlocksInterval uint64,
	startBlockNum uint64,
	stopBlockNum uint64,
	tracker *bstream.Tracker,
) *Injector {
	return &Injector{
		Shutter:             shutter.New(),
		KvStore:             kvdb,
		blocksStore:         blocksStore,
		BlockFilter:         blockFilter,
		ShardNum:            shardNum,
		MaxEntries:          maxEntries,
		flushBlocksInterval: flushBlocksInterval,
		startBlockNum:       startBlockNum,
		stopBlockNum:        stopBlockNum,
		tracker:             tracker,
		cacheSeqData:        make(map[string]accounthist.SequenceData),
		currentBatchMetrics: blockBatchMetrics{
			batchStartTime: time.Now(),
		},
	}
}

func (i *Injector) SetFacetFactory(facetFactory accounthist.FacetFactory) {
	i.facetFactory = facetFactory

}
func (i *Injector) SetupMetrics(serviceName string) {
	i.headBlockTimeDrift = metrics.NewHeadBlockTimeDrift(serviceName)
	i.headBlockNumber = metrics.NewHeadBlockNumber(serviceName)
}

func (i *Injector) Launch() {
	i.source.OnTerminating(func(err error) {
		zlog.Info("block source is shutting down, notifying service about its termination")
		i.Shutdown(err)
	})

	i.OnTerminating(func(_ error) {
		zlog.Info("accounthist service is shutting down down, shutting down block source")
		i.source.Shutdown(nil)
	})

	i.source.Run()
}

func (i *Injector) Shutdown(err error) {
	zlog.Info("accounthist service has been shutdown, about to terminate child services")
	i.Shutter.Shutdown(err)
}

func (i *Injector) deleteStaleRows(ctx context.Context, key accounthist.Facet, acctSeqData accounthist.SequenceData) (lastDeletedSeq uint64, err error) {
	// If the last current ordinal is bigger than the max allowed entries for this account,
	// adjust our sliding window by deleting anything below least recent ordinal
	if acctSeqData.CurrentOrdinal > acctSeqData.MaxEntries {
		// Don't forget, we are in a sliding window setup, so if last written was 12, assuming max entry of 5,
		// we have normally a window composed of ordinals [8, 9, 10, 11, 12], so anything from 7 and downwards should
		// be deleted.
		leastRecentOrdinal := acctSeqData.CurrentOrdinal - acctSeqData.MaxEntries

		// Assuming for this account that the last deleted ordinal is already higher or equal to our least
		// recent ordinal, it means there is nothing to do since everything below this last deleted ordinal
		// should already be gone.
		if acctSeqData.LastDeletedOrdinal >= leastRecentOrdinal {
			return acctSeqData.LastDeletedOrdinal, nil
		}

		// Let's assume our last deleted ordinal was 5, let's delete everything from 5 up and including 7
		zlog.Debug("deleting all actions between last deleted and now least recent ordinal", zap.Uint64("last_deleted_ordinal", acctSeqData.LastDeletedOrdinal), zap.Uint64("least_recent_ordinal", leastRecentOrdinal))
		for j := acctSeqData.LastDeletedOrdinal + 1; j <= leastRecentOrdinal; j++ {
			err := i.deleteAction(ctx, key, j)
			if err != nil {
				return 0, fmt.Errorf("error while deleting action: %w", err)
			}
		}
		return leastRecentOrdinal, nil
	}

	return acctSeqData.LastDeletedOrdinal, nil
}

func (i *Injector) deleteAction(ctx context.Context, key accounthist.Facet, sequenceNumber uint64) error {

	rowKey := key.Row(i.ShardNum, sequenceNumber)

	if traceEnabled {
		zlog.Debug("deleting action",
			zap.Uint64("sequence", sequenceNumber),
			zap.String("key", hex.EncodeToString(rowKey)),
		)
	}

	ctx, cancel := context.WithTimeout(ctx, accounthist.DatabaseTimeout)
	defer cancel()

	return i.KvStore.BatchDelete(ctx, [][]byte{rowKey})
}

func (i *Injector) WriteAction(ctx context.Context, key accounthist.Facet, acctSeqData accounthist.SequenceData, rawTrace []byte) error {
	rowKey := key.Row(i.ShardNum, acctSeqData.CurrentOrdinal)

	zlog.Debug("writing action", zap.Stringer("key", rowKey))

	ctx, cancel := context.WithTimeout(ctx, accounthist.DatabaseTimeout)
	defer cancel()

	if acctSeqData.LastDeletedOrdinal != 0 {
		// this is will append the protobuf pbaccounthist.ActionRowAppend to the protobuf pbaccounthist.ActionRow, since
		// both struct have the field last_deleted_seq with the same index (3), when unmarshalling the row
		// into an pbaccounthist.ActionRow the value of `last_deleted_seq` in the appended pbaccounthist.ActionRowAppend will
		// override the value defined in the pbaccounthist.ActionRow struct
		appendSeq := &pbaccounthist.ActionRowAppend{LastDeletedSeq: acctSeqData.LastDeletedOrdinal}
		encodedAppendSeq, _ := proto.Marshal(appendSeq)
		rawTrace = append(rawTrace, encodedAppendSeq...)
	}

	return i.KvStore.Put(ctx, rowKey, rawTrace)
}
