package main

import (
	"context"
	"encoding/json"
	"flag"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	"github.com/dfuse-io/derr"
	"github.com/dfuse-io/logging"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var gracefulShutdownDelay = flag.Duration("graceful-shutdown-delay", time.Second*1, "delay before shutting down, after the health endpoint returns unhealthy")
var deploymentEnv = flag.String("env", "local", "The target deployment environment")
var htmlRoot = flag.String("html-root", "./build", "Root directory where to serve assets")
var listenAddr = flag.String("listen-addr", "0.0.0.0:8001", "Interface to listen on, with main application")
var frontendConfigPath = flag.String("frontend-config-path", "config.json", "Config file for eosq frontend")

func main() {
	setupLogger()

	go func() {
		if err := http.ListenAndServe("localhost:6060", nil); err != nil {
			zlog.Info("listening localhost:6060", zap.Error(err))
		}
	}()

	zlog.Info("parsing command line flags")
	flag.Parse()

	fileInfo, err := os.Stat(*frontendConfigPath)
	if fileInfo == nil || os.IsNotExist(err) {
		zlog.Fatal("the frontend config file must exists", zap.String("config_file", *frontendConfigPath))
	}

	router := mux.NewRouter()
	eosQueryUI := NewDevServerProxy()

	router.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if derr.IsShuttingDown() {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Write([]byte("ok"))
	})

	router.PathPrefix("/").Handler(eosQueryUI)

	zlog.Info("serving http", zap.String("listen_addr", *listenAddr))
	go func() {
		err := http.ListenAndServe(*listenAddr, router)
		if err != nil {
			zlog.Fatal("listening server", zap.Error(err), zap.String("listen_addr", *listenAddr))
		}
	}()
	<-derr.SetupSignalHandler(*gracefulShutdownDelay)
}

func writeJSON(ctx context.Context, w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(v); err != nil {
		level := zapcore.ErrorLevel
		if derr.IsClientSideNetworkError(err) {
			level = zapcore.DebugLevel
		}

		logging.Logger(ctx, zlog).Check(level, "an error occurred while writing response").Write(zap.Error(err))
	}
}
