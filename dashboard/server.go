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

package dashboard

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sort"
	"time"

	rice "github.com/GeertJohan/go.rice"
	dashboard "github.com/dfuse-io/dfuse-eosio/dashboard/pb"
	pbdashboard "github.com/dfuse-io/dfuse-eosio/dashboard/pb"
	core "github.com/dfuse-io/dfuse-eosio/launcher"
	"github.com/dfuse-io/dgrpc"
	"github.com/dfuse-io/shutter"
	"github.com/golang/protobuf/ptypes"
	tspb "github.com/golang/protobuf/ptypes/timestamp"
	"github.com/gorilla/mux"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type server struct {
	*shutter.Shutter
	config            *Config
	modules           *Modules
	httpServer        *http.Server
	grpcServer        *grpc.Server
	grpcToHTTPServer  *grpcweb.WrappedGrpcServer
	managerController *core.Controller
	box               *rice.HTTPBox
}

func newServer(config *Config, modules *Modules) *server {
	return &server{
		Shutter:           shutter.New(),
		config:            config,
		modules:           modules,
		managerController: core.NewController(config.EosNodeManagerAPIAddr),
		box:               rice.MustFindBox("client/build").HTTPBox(),
	}
}

func (s *server) cleanUp(err error) {
	if s.grpcServer != nil {
		s.grpcServer.Stop()
	}

	if s.httpServer != nil {
		s.httpServer.Close()
	}
}

func (s *server) Launch() error {
	zlog.Info("starting dashboard server")
	s.OnTerminating(s.cleanUp)

	zlog.Info("configuring dashboard grpc server")
	s.grpcServer = dgrpc.NewServer(dgrpc.WithLogger(zlog))
	pbdashboard.RegisterDashboardServer(s.grpcServer, s)

	grpcConn, err := net.Listen("tcp", s.config.GRPCListenAddr)
	if err != nil {
		return err
	}

	grpcWebErr := make(chan error)
	go func() {
		grpcWebErr <- s.grpcServer.Serve(grpcConn)
	}()

	zlog.Info("configuring dashboard http server and grpc-web over http wrapper")
	s.grpcToHTTPServer = grpcweb.WrapServer(s.grpcServer,
		grpcweb.WithOriginFunc(func(origin string) bool { return true }),
		grpcweb.WithAllowedRequestHeaders([]string{"x-grpc-web", "content-type"}),
	)

	router := mux.NewRouter()
	router.PathPrefix("/api").HandlerFunc(s.grcpToHTTPApiHandler)
	router.PathPrefix("/").HandlerFunc(s.dashboardStaticHandler)
	s.httpServer = &http.Server{
		Addr:    s.config.HTTPListenAddr,
		Handler: router,
	}

	zlog.Info("starting http server that wraps grpc server")
	httpErr := make(chan error)
	go func() {
		httpErr <- s.httpServer.ListenAndServe()
	}()

	select {
	case err = <-httpErr:
	case err = <-grpcWebErr:
	}

	return err
}

func (s *server) grcpToHTTPApiHandler(resp http.ResponseWriter, req *http.Request) {
	http.StripPrefix("/api", s.grpcToHTTPServer).ServeHTTP(resp, req)
}

func (s *server) Dmesh(ctx context.Context, req *pbdashboard.DmeshRequest) (*pbdashboard.DmeshResponse, error) {
	zlog.Debug("dmesh")

	out := &pbdashboard.DmeshResponse{
		Clients: []*pbdashboard.DmeshClient{},
	}
	searchPeers := s.modules.DmeshClient.Peers()
	sort.Slice(searchPeers, func(i, j int) bool {
		return searchPeers[i].TierLevel < searchPeers[j].TierLevel
	})

	for _, peer := range searchPeers {
		out.Clients = append(out.Clients, &pbdashboard.DmeshClient{
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
	}
	return out, nil

}

func (s *server) AppsList(ctx context.Context, req *pbdashboard.AppsListRequest) (*pbdashboard.AppsListResponse, error) {
	appIDs := s.modules.Launcher.GetAppIDs()
	resp := &pbdashboard.AppsListResponse{}
	for _, appID := range appIDs {
		if appDef, found := core.AppRegistry[appID]; found {
			resp.Apps = append(resp.Apps, &pbdashboard.AppInfo{
				Id:          appDef.ID,
				Title:       appDef.Title,
				Description: appDef.Description,
			})
		}
		// TODO: should we handle this case? error?
	}
	return resp, nil
}

func (s *server) AppsMetrics(req *pbdashboard.AppsMetricsRequest, stream pbdashboard.Dashboard_AppsMetricsServer) error {
	sub := s.modules.MetricManager.Subscribe(req.FilterAppId)
	defer s.modules.MetricManager.Unsubscribe(req.FilterAppId, sub)

	for {
		select {
		case <-stream.Context().Done():
			return nil
		case metric, opened := <-sub.IncommingAppMetrics:
			if !opened {
				// we've been shutdown somehow, simply close the current connection.
				// we'll have logged at the source
				return nil
			}

			zlog.Debug("sending stream metric",
				zap.String("app_id", metric.Id),
				zap.Int("metric_count", len(metric.Metrics)),
			)
			err := stream.Send(metric)
			if err != nil {
				zlog.Info("failed writing to socket, shutting down subscription", zap.Error(err))
				return err
			}
		}
	}
}

func (s *server) AppsInfo(req *pbdashboard.AppsInfoRequest, stream pbdashboard.Dashboard_AppsInfoServer) error {
	zlog.Info("app info by name", zap.String("app_id", req.FilterAppId))
	l := s.modules.Launcher

	// when first called, stream latest status of one or all apps depending on FilterAppId
	if req.FilterAppId == "" {
		appIDs := l.GetAppIDs()
		resp := &pbdashboard.AppsInfoResponse{}
		for _, appID := range appIDs {
			if appDef, found := core.AppRegistry[appID]; found {
				resp.Apps = append(resp.Apps, &pbdashboard.AppInfo{
					Id:     appDef.ID,
					Status: l.GetAppStatus(appDef.ID),
				})
			}
			// TODO: should we handle this case? error?
		}
		stream.Send(resp)
	} else {
		resp := &pbdashboard.AppsInfoResponse{}
		if appDef, found := core.AppRegistry[req.FilterAppId]; found {
			resp.Apps = append(resp.Apps, &pbdashboard.AppInfo{
				Id:     appDef.ID,
				Status: l.GetAppStatus(appDef.ID),
			})
		}
		// TODO: should we handle this case? error?
		stream.Send(resp)
	}

	sub := s.modules.Launcher.SubscribeAppStatus()
	defer s.modules.Launcher.UnsubscribeAppStatus(sub)

	for {
		select {
		case <-stream.Context().Done():
			return nil
		case info, opened := <-sub.IncomingAppInfo:
			if !opened {
				// we've been shutdown somehow, simply close the current connection.
				// we'll have logged at the source
				return nil
			}

			if req.FilterAppId == "" || req.FilterAppId == info.Id {
				zlog.Debug("sending stream info",
					zap.String("app_id", info.Id),
					zap.Int32("app_status", int32(info.Status)),
				)

				err := stream.Send(&dashboard.AppsInfoResponse{
					Apps: []*dashboard.AppInfo{info},
				})
				if err != nil {
					zlog.Info("failed writing to socket, shutting down subscription", zap.Error(err))
					return err
				}
			}
		}
	}

}

var successfulStartAppResponse = &pbdashboard.StartAppResponse{}
var successfulStopAppResponse = &pbdashboard.StopAppResponse{}

func (s *server) StartApp(context.Context, *pbdashboard.StartAppRequest) (*pbdashboard.StartAppResponse, error) {
	response, err := s.managerController.StartNode()
	if err != nil {
		// TODO: Fix to return appropriate grpc error formatting
		return nil, fmt.Errorf("unable to start manager: %w", err)
	}

	zlog.Debug("started manager", zap.String("response", response))
	return successfulStartAppResponse, nil
}

func (s *server) StopApp(context.Context, *pbdashboard.StopAppRequest) (*pbdashboard.StopAppResponse, error) {
	response, err := s.managerController.StopNode()
	if err != nil {
		// TODO: Fix to return appropriate grpc error formatting
		return nil, fmt.Errorf("unable to stop manager: %w", err)
	}

	zlog.Debug("stopped manager", zap.String("response", response))
	return successfulStopAppResponse, nil
}

func timeToProtoTimestamp(t *time.Time) *tspb.Timestamp {
	out, _ := ptypes.TimestampProto(*t)
	return out
}
