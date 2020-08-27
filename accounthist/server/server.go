package server

import (
	"fmt"
	"net"

	"github.com/dfuse-io/dfuse-eosio/accounthist"
	pbaccounthist "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/accounthist/v1"
	"github.com/dfuse-io/dgrpc"
	"go.uber.org/zap"
)

type Server struct {
	grpcAddr string

	service *accounthist.Service
}

func New(grpcAddr string, service *accounthist.Service) *Server {
	s := &Server{
		grpcAddr: grpcAddr,
		service:  service,
	}

	return s
}

func (s *Server) Serve() {
	grpcServer := dgrpc.NewServer(dgrpc.WithLogger(zlog))
	pbaccounthist.RegisterAccountHistoryServer(grpcServer, s.service)

	zlog.Info("listening for accounthist", zap.String("addr", s.grpcAddr))
	lis, err := net.Listen("tcp", s.grpcAddr)
	if err != nil {
		s.service.Shutdown(fmt.Errorf("failed listening grpc %q: %w", s.grpcAddr, err))
		return
	}

	if err := grpcServer.Serve(lis); err != nil {
		s.service.Shutdown(fmt.Errorf("error on grpcServer.Serve: %w", err))
		return
	}
}
