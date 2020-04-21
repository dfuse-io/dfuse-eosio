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
	"os"
	"path/filepath"
	"strings"
	"sync"

	_ "github.com/dfuse-io/dfuse-eosio/kvdb-loader"
	"github.com/dfuse-io/dfuse-eosio/launcher"
	zapbox "github.com/dfuse-io/dfuse-eosio/zap-box"
	"github.com/dfuse-io/logging"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var userLog = zapbox.NewCLILogger(zap.NewNop())

type zl = zapcore.Level

// Core & Libraries
var commongLoggingDef = &launcher.LoggingDef{
	Levels: []zl{zap.WarnLevel, zap.InfoLevel, zap.InfoLevel, zap.DebugLevel},
}

var dfuseLoggingDef = &launcher.LoggingDef{
	Levels: []zl{zap.InfoLevel, zap.InfoLevel, zap.DebugLevel},
	Regex:  "github.com/dfuse-io/dfuse-eosio(/metrics|/cmd/dfuseeos)?$",
}

var bstreamLoggingDef = &launcher.LoggingDef{
	Levels: []zl{zap.WarnLevel, zap.InfoLevel, zap.InfoLevel, zap.DebugLevel},
	Regex:  "github.com/dfuse-io/bstream.*",
}

func init() {
	logging.Register("github.com/dfuse-io/dfuse-eosio/cmd/dfuseeos", userLog.LoggerReference())
}

func setupLogger() {
	dataDir := viper.GetString("global-data-dir")
	verbosity := viper.GetInt("global-verbose")

	// TODO: The logger expect that the dataDir already exists...
	// The second argument is a `closer` method, it should be linked to exit of application, for now, we don't care, OS will cleanup
	logFileWriter := createLogFileWriter(dataDir)
	logStdoutWriter := zapcore.Lock(os.Stdout)

	commonLogger := createLogger("common", commongLoggingDef, verbosity, logFileWriter, logStdoutWriter)
	logging.Set(commonLogger)

	for _, appDef := range launcher.AppRegistry {
		logging.Set(createLogger(appDef.ID, appDef.Logger, verbosity, logFileWriter, logStdoutWriter), appDef.Logger.Regex)
	}
	logging.Set(createLogger("dfuse", dfuseLoggingDef, verbosity, logFileWriter, logStdoutWriter), dfuseLoggingDef.Regex)
	logging.Set(createLogger("bstream", bstreamLoggingDef, verbosity, logFileWriter, logStdoutWriter), bstreamLoggingDef.Regex)

	// Fine-grain customization
	//
	// Note that `zapbox.WithLevel` used below does not work in all circumstances! See
	// https://github.com/uber-go/zap/issues/581#issuecomment-600641485 for details.

	if value := os.Getenv("WARN"); value != "" {
		changeLoggersLevel(value, zap.WarnLevel)
	}

	if value := os.Getenv("INFO"); value != "" {
		changeLoggersLevel(value, zap.InfoLevel)
	}

	if value := os.Getenv("DEBUG"); value != "" {
		changeLoggersLevel(value, zap.DebugLevel)
	}

	// The userLog are wrapped, they need to be re-configured with newly set base instance to work correctly
	userLog.ReconfigureReference()
	launcher.UserLog().ReconfigureReference()

	// Hijack standard Golang `log` and redirect it to our common logger
	zap.RedirectStdLogAt(commonLogger, zap.DebugLevel)
}

var appToAtomicLevel = map[string]zap.AtomicLevel{}
var appToAtomicLevelLock sync.Mutex

func createLogger(appID string, loggingDef *launcher.LoggingDef, verbosity int, fileSyncer zapcore.WriteSyncer, consoleSyncer zapcore.WriteSyncer) *zap.Logger {
	fileCore := zapcore.NewNopCore()
	if fileSyncer != nil {
		encoderConfig := zap.NewProductionEncoderConfig()
		fileCore = zapcore.NewCore(zapcore.NewJSONEncoder(encoderConfig), fileSyncer, zap.InfoLevel)
	}

	// It's ok for concurrent use here, we assume all logger are created in a single goroutine
	appToAtomicLevel[appID] = zap.NewAtomicLevelAt(appLoggerLevel(loggingDef.Levels, verbosity))
	consoleCore := zapcore.NewCore(zapbox.NewEncoder(verbosity), consoleSyncer, appToAtomicLevel[appID])
	teeCore := zapcore.NewTee(consoleCore, fileCore)

	return zap.New(teeCore, zap.AddCaller()).Named(appID)
}

func changeLoggersLevel(inputs string, level zapcore.Level) {
	for _, input := range strings.Split(inputs, ",") {
		normalizeInput := strings.Trim(input, " ")
		if normalizeInput == "bstream" || normalizeInput == "dfuse" || launcher.AppRegistry[normalizeInput] != nil {
			changeAppLogLevel(normalizeInput, level)
		} else {
			// Assumes it's a regex, we use the unnormalized input, just in case it had some spaces
			logging.Extend(overrideLoggerLevel(level), input)
		}
	}
}

// At some point, we will want to control the level from the server directly. It will
// be possible to use this method to achieve that. However, it might be required to be
// moved to `dfuse` package directly, so it's available to be used by the `gRPC` server
// in dashboard. To be determined once the issue is tackled.
func changeAppLogLevel(appID string, level zapcore.Level) {
	appToAtomicLevelLock.Lock()
	defer appToAtomicLevelLock.Unlock()

	atomicLevel, found := appToAtomicLevel[appID]
	if found {
		atomicLevel.SetLevel(level)
	}
}

func overrideLoggerLevel(level zapcore.Level) logging.LoggerExtender {
	return func(current *zap.Logger) *zap.Logger {
		return current.WithOptions(zapbox.WithLevel(level))
	}
}

func appLoggerLevel(levels []zl, verbosity int) zapcore.Level {
	severityIndex := verbosity
	if severityIndex > len(levels)-1 {
		severityIndex = len(levels) - 1
	}

	return levels[severityIndex]
}

func createLogFileWriter(dataDir string) zapcore.WriteSyncer {
	_ = os.Mkdir(dataDir, 0755)

	logFile := filepath.Join(dataDir, "dfuse.log.json")
	writer, _, err := zap.Open(logFile)
	if err != nil {
		tempLogFile := filepath.Join(os.TempDir(), "dfuse.log.json")
		fmt.Printf("Unable to use %q for logging purposes, trying with %q instead (error: %s)\n", logFile, tempLogFile, err)
		writer, _, err := zap.Open(logFile)
		if err != nil {
			fmt.Printf("Unable to use %q for logging purposes, logs won't be saved in a log file and will be printed to console only (error: %s)\n", tempLogFile, err)
			return writer
		}
	}

	// Might return `nil`, which is handled by logging
	return writer
}
