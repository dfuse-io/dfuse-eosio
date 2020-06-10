package grpc

import (
	"fmt"
	"net"

	"github.com/dfuse-io/dfuse-eosio/fluxdb"
	pbfluxdb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/fluxdb/v1"
	"github.com/dfuse-io/dgrpc"
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
	zlog.Info("listening & serving GRPC content", zap.String("http_listen_addr", s.grpcAddr))

	grpcServer := dgrpc.NewServer(dgrpc.WithLogger(zlog))
	pbfluxdb.RegisterStateServer(grpcServer, s)

	lis, err := net.Listen("tcp", s.grpcAddr)
	if err != nil {
		s.db.Shutdown(fmt.Errorf("failed listening grpc %q: %w", s.grpcAddr, err))
		return
	}

	zlog.Info("listening & serving gRPC content", zap.String("grpc_listen_addr", s.grpcAddr))
	if err := grpcServer.Serve(lis); err != nil {
		s.db.Shutdown(fmt.Errorf("error on gs.Serve: %w", err))
		return
	}
}

func newMetadata(upToBlockID, lastWrittenBlockID string) metadata.MD {
	md := metadata.New(map[string]string{})
	md.Set("flux-up-to-block-id", upToBlockID)
	md.Set("flux-up-to-block-num", fmt.Sprintf("%d", fluxdb.BlockNum(upToBlockID)))
	md.Set("flux-last-irreversible-block-id", lastWrittenBlockID)
	md.Set("flux-last-irreversible-block-num", fmt.Sprintf("%d", fluxdb.BlockNum(lastWrittenBlockID)))
	return md
}
