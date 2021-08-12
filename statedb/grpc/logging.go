package grpc

import (
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
)

var zlog = zap.NewNop()

func init() {
	logging.Register("github.com/dfuse-io/dfuse-eosio/statedb/grpc", &zlog)
}
