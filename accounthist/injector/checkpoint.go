package injector

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/dfuse-io/dfuse-eosio/accounthist"

	pbaccounthist "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/accounthist/v1"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/dfuse-io/kvdb/store"
	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"
)

func (i *Injector) GetShardCheckpoint(ctx context.Context) (*pbaccounthist.ShardCheckpoint, error) {

	key := CheckpointKeyGenerator(i.ShardNum)

	ctx, cancel := context.WithTimeout(ctx, accounthist.DatabaseTimeout)
	defer cancel()

	val, err := i.KvStore.Get(ctx, key)
	if err == store.ErrNotFound {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("error while last processed block: %w", err)
	}

	// Decode val as `pbaccounthist.ShardCheckpoint`
	out := &pbaccounthist.ShardCheckpoint{}
	if err := proto.Unmarshal(val, out); err != nil {
		return nil, err
	}

	return out, nil
}

func (i *Injector) DeleteCheckpoint(ctx context.Context, shard byte) error {
	key := CheckpointKeyGenerator(shard)

	if traceEnabled {
		zlog.Debug("deleting checkpoint",
			zap.Int("shard_num", int(shard)),
			zap.String("checkpoint_key", hex.EncodeToString(key)),
		)
	}

	ctx, cancel := context.WithTimeout(ctx, accounthist.DatabaseTimeout)
	defer cancel()

	i.KvStore.BatchDelete(ctx, [][]byte{key})
	return i.KvStore.FlushPuts(ctx)
}

func (i *Injector) writeCheckpoint(ctx context.Context, blk *pbcodec.Block) error {
	key := CheckpointKeyGenerator(i.ShardNum)

	i.lastCheckpoint.LastWrittenBlockNum = blk.Num()
	i.lastCheckpoint.LastWrittenBlockId = blk.ID()

	value, err := proto.Marshal(i.lastCheckpoint)
	if err != nil {
		return err
	}

	return i.KvStore.Put(ctx, key, value)
}
