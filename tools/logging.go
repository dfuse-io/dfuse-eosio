package tools

import (
	"os"

	"go.uber.org/zap"
)

var zlog = zap.NewNop()

func init() {
	if os.Getenv("DEBUG") != "" {
		logger, _ := zap.NewDevelopment()
		zlog = logger
	}
}
