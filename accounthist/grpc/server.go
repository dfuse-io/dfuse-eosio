package grpc

import (
	"fmt"
	"net"
	"time"

	"github.com/dfuse-io/dfuse-eosio/accounthist"
	pbaccounthist "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/accounthist/v1"
	"github.com/dfuse-io/dgrpc"
	"github.com/dfuse-io/shutter"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type Server struct {
	*shutter.Shutter

	grpcAddr string
	server   *grpc.Server
	service  *accounthist.Service
}

func New(grpcAddr string, service *accounthist.Service) *Server {
	return &Server{
		Shutter: shutter.New(),

		grpcAddr: grpcAddr,
		service:  service,
	}
}

func (s *Server) Serve() {
	s.server = dgrpc.NewServer(dgrpc.WithLogger(zlog))
	pbaccounthist.RegisterAccountHistoryServer(s.server, s)

	zlog.Info("listening for accounthist", zap.String("addr", s.grpcAddr))
	lis, err := net.Listen("tcp", s.grpcAddr)
	if err != nil {
		s.Shutdown(fmt.Errorf("failed listening grpc %q: %w", s.grpcAddr, err))
		return
	}

	if err := s.server.Serve(lis); err != nil {
		s.Shutdown(fmt.Errorf("error on grpcServer.Serve: %w", err))
		return
	}
}

func (s *Server) Terminate(err error) {
	if s.server == nil {
		return
	}

	stopped := make(chan bool)

	// Stop the server gracefully
	go func() {
		s.server.GracefulStop()
		close(stopped)
	}()

	// And don't wait more than 60 seconds for graceful stop to happen
	select {
	case <-time.After(30 * time.Second):
		zlog.Info("gRPC server did not terminate gracefully within allowed time, forcing shutdown")
		s.server.Stop()
	case <-stopped:
		zlog.Info("gRPC server teminated gracefully")
	}
}
