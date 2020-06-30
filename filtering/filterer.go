package filtering

import (
	"context"
	"fmt"
	"io"
	"net"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/dfuse-eosio/codec"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/dfuse-io/dgrpc"
	pbbstream "github.com/dfuse-io/pbgo/dfuse/bstream/v1"
	pbhealth "github.com/dfuse-io/pbgo/grpc/health/v1"
	"github.com/dfuse-io/shutter"
	"go.uber.org/zap"
)

type Filterer struct {
	*shutter.Shutter

	relayerAddr    string
	grpcListenAddr string
	blockFilter    *BlockFilter

	ready bool
}

func NewFilterer(relayerAddr string, grpcListenAddr string, blockFilter *BlockFilter) *Filterer {
	return &Filterer{
		Shutter:        shutter.New(),
		relayerAddr:    relayerAddr,
		grpcListenAddr: grpcListenAddr,
		blockFilter:    blockFilter,
	}
}

func (r *Filterer) Launch() {
	lis, err := net.Listen("tcp", r.grpcListenAddr)
	if err != nil {
		r.Shutdown(fmt.Errorf("failed listening grpc %q: %w", r.grpcListenAddr, err))
		return
	}

	zlog.Info("tcp listener created")
	zlog.Info("listening & serving grpc content", zap.String("grpc_listen_addr", r.grpcListenAddr))

	gs := dgrpc.NewServer()
	pbhealth.RegisterHealthServer(gs, r)
	pbbstream.RegisterBlockStreamServer(gs, r)

	r.ready = true
	if err := gs.Serve(lis); err != nil {
		r.Shutdown(fmt.Errorf("error on grpc serve: %w", err))
		return
	}
}

func (f *Filterer) Blocks(req *pbbstream.BlockRequest, srv pbbstream.BlockStream_BlocksServer) error {
	relayerConn, err := dgrpc.NewInternalClient(f.relayerAddr)
	if err != nil {
		return fmt.Errorf("unable to create relayer grpc client: %w", err)
	}

	relayer := pbbstream.NewBlockStreamClient(relayerConn)

	relayerCtx, cancelStreamBlocks := context.WithCancel(srv.Context())
	defer cancelStreamBlocks()

	streamBlocks, err := relayer.Blocks(relayerCtx, req)
	if err != nil {
		return fmt.Errorf("relayer stream blocks failed: %w", err)
	}

	for {
		pbblock, err := streamBlocks.Recv()
		if err == io.EOF {
			return io.EOF
		}

		if err != nil {
			return fmt.Errorf("unable to receive relayer block: %w", err)
		}

		blk, err := bstream.BlockFromProto(pbblock)
		if err != nil {
			return fmt.Errorf("unable to decode proto block %s: %w", bstream.BlockRefFromID(pbblock.Id), err)
		}

		filteredBlock, err := f.filterBlock(blk.ToNative().(*pbcodec.Block))
		if err != nil {
			return fmt.Errorf("unable to filter block %s: %w", blk, err)
		}

		filteredPbblock, err := f.packBlock(filteredBlock)
		if err != nil {
			return fmt.Errorf("unable to pack filtered block %s: %w", blk, err)
		}

		err = srv.Send(filteredPbblock)
		if err != nil {
			return fmt.Errorf("unable to send filtered block: %w", err)
		}
	}
}

func (f *Filterer) filterBlock(block *pbcodec.Block) (*pbcodec.Block, error) {
	f.blockFilter.TransformInPlace(block)
	return block, nil
}

func (f *Filterer) packBlock(block *pbcodec.Block) (*pbbstream.Block, error) {
	blk, err := codec.BlockFromProto(block)
	if err != nil {
		return nil, fmt.Errorf("unable to transform codec block to bstream block %s: %w", block.AsRef(), err)
	}

	return blk.ToProto()
}

func (r *Filterer) Check(ctx context.Context, in *pbhealth.HealthCheckRequest) (*pbhealth.HealthCheckResponse, error) {
	status := pbhealth.HealthCheckResponse_NOT_SERVING
	if r.ready {
		status = pbhealth.HealthCheckResponse_SERVING
	}
	return &pbhealth.HealthCheckResponse{
		Status: status,
	}, nil
}
