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

package abicodec

import (
	"context"
	"fmt"
	"net"

	pbabicodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/abicodec/v1"
	"github.com/dfuse-io/dgrpc"
	pbhealth "github.com/dfuse-io/pbgo/grpc/health/v1"
	"github.com/dfuse-io/shutter"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type Server struct {
	*shutter.Shutter

	cache          Cache
	grpcListenAddr string
	gs             *grpc.Server
	ready          bool
}

func NewServer(cache Cache, grpcListenAddr string) *Server {
	gs := dgrpc.NewServer(dgrpc.WithLogger(zlog))
	pbabicodec.RegisterDecoderServer(gs, NewDecoder(cache))
	srv := &Server{
		Shutter:        shutter.New(),
		cache:          cache,
		grpcListenAddr: grpcListenAddr,
		gs:             gs,
	}

	pbhealth.RegisterHealthServer(gs, srv)

	return srv
}

func (s *Server) Check(ctx context.Context, in *pbhealth.HealthCheckRequest) (*pbhealth.HealthCheckResponse, error) {
	status := pbhealth.HealthCheckResponse_NOT_SERVING
	if s.ready {
		status = pbhealth.HealthCheckResponse_SERVING
	}

	return &pbhealth.HealthCheckResponse{
		Status: status,
	}, nil
}

func (s *Server) SetReady() {
	s.ready = true
}

func (s *Server) Serve() {
	zlog.Info("starting grpc server", zap.String("address", s.grpcListenAddr))
	listener, err := net.Listen("tcp", s.grpcListenAddr)
	if err != nil {
		s.Shutdown(fmt.Errorf("unable to listen on %q: %w", s.grpcListenAddr, err))
		return
	}

	err = s.gs.Serve(listener)
	if err == nil || err == grpc.ErrServerStopped {
		zlog.Info("server shut down cleanly, nothing to do")
		return
	}

	if err != nil {
		s.Shutdown(err)
	}
}

func (s *Server) Stop(err error) {
	s.gs.GracefulStop()
}
