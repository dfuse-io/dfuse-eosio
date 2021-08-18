package codec

import (
	"os"

	"github.com/streamingfast/logging"
	"go.uber.org/zap"
)

func init() {
	if os.Getenv("DEBUG") != "" || os.Getenv("TRACE") == "true" {
		logger, _ := zap.NewDevelopment()
		logging.Override(logger)
	}
}
