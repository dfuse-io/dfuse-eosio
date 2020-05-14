package main

import (
	"net/http"
	_ "net/http/pprof"

	"github.com/dfuse-io/sqlsync"
	"go.uber.org/zap"
)

func setup() {
	setupLogger()
	setupMetrics()

	go func() {
		listenAddr := "localhost:6060"
		err := http.ListenAndServe(listenAddr, nil)
		if err != nil {
			zlog.Error("unable to start profiling server", zap.Error(err), zap.String("listen_addr", listenAddr))
		}
	}()

	go sqlsync.ServeMetrics()
}
