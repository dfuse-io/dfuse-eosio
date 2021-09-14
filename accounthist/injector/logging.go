package injector

import (
	"os"

	"github.com/streamingfast/logging"
	"go.uber.org/zap"
)

var traceEnabled = os.Getenv("TRACE") == "true"
var zlog *zap.Logger

func init() {
	logging.Register("github.com/dfuse-io/dfuse-eosio/accounthist/injector", &zlog)
}
