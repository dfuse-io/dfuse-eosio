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
	"github.com/dfuse-io/dfuse-box/launcher"
	zapbox "github.com/dfuse-io/dfuse-box/zap-box"
	_ "github.com/dfuse-io/dfuse-eosio/trxdb-loader"
	"github.com/dfuse-io/logging"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var userLog = zapbox.NewCLILogger(zap.NewNop())
var zlog *zap.Logger

func init() {
	logging.Register("github.com/dfuse-io/dfuse-eosio/cmd/dfuseeos/userlog", userLog.LoggerReference())
	logging.Register("github.com/dfuse-io/dfuse-eosio/cmd/dfuseeos", &zlog)
	// Core & Libraries
	launcher.CommongLoggingDef = &launcher.LoggingDef{
		Levels: []zapcore.Level{zap.WarnLevel, zap.InfoLevel, zap.InfoLevel, zap.DebugLevel},
	}

	launcher.DfuseLoggingDef = &launcher.LoggingDef{
		Levels: []zapcore.Level{zap.InfoLevel, zap.InfoLevel, zap.DebugLevel},
		Regex:  "github.com/dfuse-io/dfuse-eosio(/metrics|/cmd/dfuseeos)?$",
	}

	launcher.BstreamLoggingDef = &launcher.LoggingDef{
		Levels: []zapcore.Level{zap.WarnLevel, zap.InfoLevel, zap.InfoLevel, zap.DebugLevel},
		Regex:  "github.com/dfuse-io/bstream.*",
	}

}
