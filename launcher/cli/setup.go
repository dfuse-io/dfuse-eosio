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

package cli

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"syscall"

	"github.com/dfuse-io/dgrpc"
	"github.com/dfuse-io/dmetrics"
	"go.uber.org/zap"
)

func init() {
	dgrpc.Verbosity = 2
}

func setup() {
	setupLogger()
	setupTracing()

	userLog.Printf("Confidential property of dfuse")
	go dmetrics.Serve(":9102")

	err := setMaxOpenFilesLimit()
	if err != nil {
		userLog.Warn("unable to adjust ulimit max open files value, it might causes problem along the road", zap.Error(err))
	}

	go func() {
		listenAddr := "localhost:6060"
		err := http.ListenAndServe(listenAddr, nil)
		if err != nil {
			userLog.Debug("unable to start profiling server", zap.Error(err), zap.String("listen_addr", listenAddr))
		}
	}()
}

const osxDefaultMaximalOpenFilesLimit uint64 = 24576

func setMaxOpenFilesLimit() error {
	maxOpenFilesLimit, err := getMaxOpenFilesLimit()
	if err != nil {
		return err
	}

	userLog.Debug("ulimit max open files before adjustment", zap.Uint64("current_value", maxOpenFilesLimit))

	// For now, we use OS X maximal value because changing the maximal value on OS X
	// is rather hard (see https://superuser.com/a/514049/459230). As such, we will try
	// to ensure that `dfusebox` can work under such maximal values of open files.
	err = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &syscall.Rlimit{
		Cur: osxDefaultMaximalOpenFilesLimit,
		Max: osxDefaultMaximalOpenFilesLimit,
	})

	if err != nil {
		return fmt.Errorf("cannot set ulimit max open files: %w", err)
	}

	maxOpenFilesLimit, err = getMaxOpenFilesLimit()
	if err != nil {
		return err
	}

	userLog.Debug("ulimit max open files after adjustment", zap.Uint64("current_value", maxOpenFilesLimit))
	return nil
}

func getMaxOpenFilesLimit() (uint64, error) {
	var rLimit syscall.Rlimit
	err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		return 0, fmt.Errorf("cannot get ulimit max open files value: %w", err)
	}

	return rLimit.Cur, nil
}
