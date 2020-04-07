// Copyright 2020 dfuse Platform Inc.
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

package eosws

import (
	"context"
	"fmt"
	"io"
	"net/http"

	pbsearch "github.com/dfuse-io/pbgo/dfuse/search/v1"
	"github.com/dfuse-io/dtracing"
	"github.com/dfuse-io/logging"
	"go.uber.org/atomic"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type MultiRouterClient struct {
	v1Client pbsearch.RouterClient
	v2Client pbsearch.RouterClient
	Toggle   *atomic.Bool
}

func NewMultiRouterClient(v1Client pbsearch.RouterClient, v2Client pbsearch.RouterClient) *MultiRouterClient {
	routerClient := &MultiRouterClient{
		v1Client: v1Client,
		v2Client: v2Client,
		Toggle:   atomic.NewBool(false),
	}

	go func() {
		zlog.Info("starting atomic level switcher, port :1066")
		if err := http.ListenAndServe(":1066", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			routerClient.Toggle.Toggle()
			w.Write([]byte(fmt.Sprintf("switch toggles: %t", routerClient.Toggle.Load())))
		})); err != nil {
			zlog.Info("failed listening on :1066 to switch multi search router:", zap.Error(err))
		}
	}()
	return routerClient
}

func (m *MultiRouterClient) StreamMatches(ctx context.Context, in *pbsearch.RouterRequest, opts ...grpc.CallOption) (pbsearch.Router_StreamMatchesClient, error) {
	if m.Toggle.Load() {
		go func() {
			zlogger := logging.Logger(ctx, zlog)
			zlogger.Info("Sending request to secondary server.",
				zap.String("dhammer_drill_trace_id", dtracing.GetTraceID(ctx).String()))

			v2Stream, err := m.v2Client.StreamMatches(ctx, in, opts...)
			if err != nil {
				zlog.Warn("V2 failed to stream matches:", zap.Error(ctx.Err()))
				return
			}
			count := 0
			for {
				_, err := v2Stream.Recv()
				if ctx.Err() != nil {
					zlogger.Warn("V2 ctx error:",
						zap.String("dhammer_drill_trace_id", dtracing.GetTraceID(ctx).String()),
						zap.Error(ctx.Err()))
					break
				}
				if err != nil {
					if err == io.EOF {
						zlogger.Info("V2 Recv done:",
							zap.String("dhammer_drill_trace_id", dtracing.GetTraceID(ctx).String()),
							zap.Int("count", count))
						break
					}
					zlogger.Warn("V2 Recv failed:",
						zap.String("dhammer_drill_trace_id", dtracing.GetTraceID(ctx).String()),
						zap.Error(err))
					break
				}
				count++
			}
		}()
	}

	v1Stream, err := m.v1Client.StreamMatches(ctx, in, opts...)
	if err != nil {
		return nil, err
	}
	return v1Stream, nil
}
