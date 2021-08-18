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
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/streamingfast/derr"
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/olivere/elastic.v3/backoff"
)

func FowardErrorResponse(w http.ResponseWriter, r *http.Request, response *http.Response) {
	content, err := ioutil.ReadAll(response.Body)
	if err != nil {
		WriteError(w, r, derr.Wrap(err, "unable to read response body while forwarding response"))
	}

	w.WriteHeader(response.StatusCode)
	_, err = w.Write(content)
	if err != nil {
		logWriteResponseError(r.Context(), "failed forwarding error response", err)
	}
}

func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	derr.WriteError(r.Context(), w, "unable to fullfil request", err)
}

func WriteJSON(w http.ResponseWriter, r *http.Request, v interface{}) {
	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(v); err != nil {
		logWriteResponseError(r.Context(), "failed encoding JSON response", err)
	}
}

func logWriteResponseError(ctx context.Context, message string, err error) {
	level := zapcore.ErrorLevel
	if derr.IsClientSideNetworkError(err) {
		level = zapcore.DebugLevel
	}

	logging.Logger(ctx, zlog).Check(level, message).Write(zap.Error(err))
}

func Retry(ctx context.Context, attempts int, sleep time.Duration, callback func() error) (err error) {
	zlogger := logging.Logger(ctx, zlog)

	b := backoff.NewExponentialBackoff(sleep, 5*time.Second)
	for i := 0; ; i++ {
		err = callback()
		if err == nil {
			return
		}

		if ctx.Err() == context.Canceled {
			return ctx.Err()
		}

		if i >= (attempts - 1) {
			break
		}

		time.Sleep(b.Next())
		zlogger.Debug("retrying after error", zap.Error(err))
	}

	return derr.Wrapf(err, "failed after %d attempts", attempts)
}

func int64Input(in string) int64 {
	if in == "" {
		return 0
	}
	val, _ := strconv.ParseInt(in, 10, 64)
	return val
}

func boolInput(in string) bool {
	return in == "true" || in == "1"
}

func mapString(input string) map[string]bool {
	out := make(map[string]bool)
	for _, el := range strings.Split(input, "|") {
		word := strings.TrimSpace(el)
		if word != "" {
			out[word] = true
		}
	}
	return out
}
