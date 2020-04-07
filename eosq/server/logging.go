package main

import (
	"github.com/dfuse-io/logging"
	"go.uber.org/zap"
)

var zlog = zap.NewNop()

func setupLogger() {
	zlog = logging.MustCreateLogger()
}
