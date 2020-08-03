package grpc

import (
	"fmt"
	"net"
	"strconv"

	"github.com/dfuse-io/bstream"
	pbstatedb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/statedb/v1"
	"github.com/dfuse-io/dgrpc"
	"github.com/dfuse-io/fluxdb"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"
)

type Server struct {
	db       *fluxdb.FluxDB
	grpcAddr string
}

func New(grpcAddr string, db *fluxdb.FluxDB) *Server {
	return &Server{
		db:       db,
		grpcAddr: grpcAddr,
	}
}

func (s *Server) Serve() {
	zlog.Info("listening & serving GRPC content", zap.String("grpc_listen_addr", s.grpcAddr))
	grpcServer := dgrpc.NewServer(dgrpc.WithLogger(zlog))
	pbstatedb.RegisterStateServer(grpcServer, s)

	lis, err := net.Listen("tcp", s.grpcAddr)
	if err != nil {
		s.db.Shutdown(fmt.Errorf("failed listening grpc %q: %w", s.grpcAddr, err))
		return
	}

	if err := grpcServer.Serve(lis); err != nil {
		s.db.Shutdown(fmt.Errorf("error on gs.Serve: %w", err))
		return
	}
}

func newMetadata(upToBlock, lastWrittenBlock bstream.BlockRef) metadata.MD {
	md := metadata.New(map[string]string{})
	if upToBlock != nil {
		md.Set("flux-up-to-block-id", upToBlock.ID())
		md.Set("flux-up-to-block-num", strconv.FormatUint(upToBlock.Num(), 10))
	}

	md.Set("flux-last-irreversible-block-id", lastWrittenBlock.ID())
	md.Set("flux-last-irreversible-block-num", strconv.FormatUint(lastWrittenBlock.Num(), 10))
	return md
}
