package filtering

import (
	"os"

	"github.com/dfuse-io/logging"
	"go.uber.org/zap"
)

var traceEnabled = false
var zlog *zap.Logger

func init() {
	logging.Register("github.com/dfuse-io/dfuse-eosio/filtering", &zlog)

	if os.Getenv("TRACE") == "true" {
		traceEnabled = true
	}
}
