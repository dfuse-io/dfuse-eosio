package migrator

import (
	"testing"

	rice "github.com/GeertJohan/go.rice"

	"github.com/eoscanada/eos-go/ecc"

	"github.com/eoscanada/eos-go"

	"github.com/dfuse-io/logging"
	"go.uber.org/zap"
)

func init() {
	//if os.Getenv("DEBUG") != "" {
	logger, _ := zap.NewDevelopment()
	logging.Override(logger)
	//}
}

func Test_Migrator(t *testing.T) {
	actions := make(chan interface{})

	migrator := &Migrator{
		box:         rice.MustFindBox("./code/build"),
		contract:    "dfuse.mgrt",
		opPublicKey: ecc.PublicKey{},
		actionChan:  actions,
		dataDir:     "/Users/julien/codebase/dfuse-io/dfuseeos-data/boot/migration-data",
	}

	go func() {
		defer close(actions)
		migrator.startMigration()
	}()

	for {
		act, ok := <-actions
		if !ok {
			zlog.Info("Test Done")
			return
		}
		switch act.(type) {
		case *eos.Action:
			action := act.(*eos.Action)
			if action != nil {
				zlog.Info("received action")
			}
		}
	}
}
