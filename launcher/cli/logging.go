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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/blendle/zapdriver"
	"github.com/dfuse-io/dfuse-eosio/launcher"
	_ "github.com/dfuse-io/dfuse-eosio/trxdb-loader"
	zapbox "github.com/dfuse-io/dfuse-eosio/zap-box"
	"github.com/dfuse-io/logging"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var userLog = zapbox.NewCLILogger(zap.NewNop())

type zl = zapcore.Level

// Core & Libraries
var commonLoggingDef = &launcher.LoggingDef{
	Levels: []zl{zap.WarnLevel, zap.WarnLevel, zap.InfoLevel, zap.DebugLevel},
}

var dfuseLoggingDef = &launcher.LoggingDef{
	Levels: []zl{zap.InfoLevel, zap.InfoLevel, zap.DebugLevel},
	Regex:  "github.com/dfuse-io/dfuse-eosio(/metrics|/cmd/dfuseeos)?$",
}

var bstreamLoggingDef = &launcher.LoggingDef{
	Levels: []zl{zap.WarnLevel, zap.WarnLevel, zap.InfoLevel, zap.DebugLevel},
	Regex:  "github.com/dfuse-io/bstream.*",
}

func init() {
	logging.Register("github.com/dfuse-io/dfuse-eosio/cmd/dfuseeos", userLog.LoggerReference())
}

func setupLogger() {
	dataDir := viper.GetString("global-data-dir")
	verbosity := viper.GetInt("global-verbose")
	logformat := viper.GetString("global-log-format")
	logToFile := viper.GetBool("global-log-to-file")
	listenAddr := viper.GetString("global-log-level-switcher-listen-addr")

	// TODO: The logger expect that the dataDir already exists...

	var logFileWriter zapcore.WriteSyncer
	if logToFile {
		logFileWriter = createLogFileWriter(dataDir)
	}
	logStdoutWriter := zapcore.Lock(os.Stdout)

	commonLogger := createLogger("common", commonLoggingDef, verbosity, logFileWriter, logStdoutWriter, logformat)
	logging.Set(commonLogger)

	for _, appDef := range launcher.AppRegistry {
		logging.Set(createLogger(appDef.ID, appDef.Logger, verbosity, logFileWriter, logStdoutWriter, logformat), appDef.Logger.Regex)
	}
	logging.Set(createLogger("dfuse", dfuseLoggingDef, verbosity, logFileWriter, logStdoutWriter, logformat), dfuseLoggingDef.Regex)
	logging.Set(createLogger("bstream", bstreamLoggingDef, verbosity, logFileWriter, logStdoutWriter, logformat), bstreamLoggingDef.Regex)

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

	if listenAddr != "" {
		go func() {
			userLog.Debug("starting atomic level switcher", zap.String("listen_addr", listenAddr))
			if err := http.ListenAndServe(listenAddr, http.HandlerFunc(handleHTTPLogChange)); err != nil {
				userLog.Warn("failed starting atomic level switcher", zap.Error(err), zap.String("listen_addr", listenAddr))
			}
		}()
	}

}

type logChangeReq struct {
	Inputs string `json:"inputs"`
	Level  string `json:"level"`
}

func handleHTTPLogChange(w http.ResponseWriter, r *http.Request) {

	b, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		http.Error(w, fmt.Sprintf("cannot read body: %s", err), 400)
		return
	}

	// Unmarshal
	var in logChangeReq
	err = json.Unmarshal(b, &in)
	if err != nil {
		http.Error(w, fmt.Sprintf("cannot unmarshal JSON body: %s", err), 400)
		return
	}

	if in.Inputs == "" {
		http.Error(w, fmt.Sprintf("inputs not defined, should be comma-separated list of words or a regular expressions: %s", err), 400)
		return
	}

	switch strings.ToLower(in.Level) {
	case "warn", "warning":
		changeLoggersLevel(in.Inputs, zap.WarnLevel)
	case "info":
		changeLoggersLevel(in.Inputs, zap.InfoLevel)
	case "debug":
		changeLoggersLevel(in.Inputs, zap.DebugLevel)
	default:
		http.Error(w, fmt.Sprintf("invalid value for 'level': %s", in.Level), 400)
		return
	}

	w.Write([]byte("ok"))
}

var appToAtomicLevel = map[string]zap.AtomicLevel{}
var appToAtomicLevelLock sync.Mutex

func createLogger(appID string, loggingDef *launcher.LoggingDef, verbosity int, fileSyncer zapcore.WriteSyncer, consoleSyncer zapcore.WriteSyncer, format string) *zap.Logger {

	// It's ok for concurrent use here, we assume all logger are created in a single goroutine
	appToAtomicLevel[appID] = zap.NewAtomicLevelAt(appLoggerLevel(loggingDef.Levels, verbosity))
	opts := []zap.Option{zap.AddCaller()}

	var consoleCore zapcore.Core
	switch format {
	case "stackdriver":
		opts = append(opts, zapdriver.WrapCore(zapdriver.ReportAllErrors(true), zapdriver.ServiceName(appID)))
		encoderConfig := zapdriver.NewProductionEncoderConfig()
		consoleCore = zapcore.NewCore(zapcore.NewJSONEncoder(encoderConfig), consoleSyncer, appToAtomicLevel[appID])
	default:
		consoleCore = zapcore.NewCore(zapbox.NewEncoder(verbosity), consoleSyncer, appToAtomicLevel[appID])
	}

	if fileSyncer == nil {
		return zap.New(consoleCore, opts...).Named(appID)
	}

	encoderConfig := zap.NewProductionEncoderConfig()
	fileCore := zapcore.NewCore(zapcore.NewJSONEncoder(encoderConfig), fileSyncer, zap.InfoLevel)
	teeCore := zapcore.NewTee(consoleCore, fileCore)

	return zap.New(teeCore, opts...).Named(appID)

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
