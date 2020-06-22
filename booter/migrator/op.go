package migrator

import (
	"fmt"

	"github.com/eoscanada/eos-go"

	"github.com/dfuse-io/eosio-boot/config"
	bootops "github.com/dfuse-io/eosio-boot/ops"
	"github.com/eoscanada/eos-go/ecc"
)

func init() {
	bootops.Register("migration.inject", &OpMigration{})
}

type OpMigration struct {
	Account string `json:"account"`
	DataDir string `json:"data_dir"`
}

func (op *OpMigration) Actions(opPubkey ecc.PublicKey, c *config.OpConfig, in chan interface{}) error {
	migrator := newMigrator(eos.AN(op.Account), opPubkey, op.DataDir, in)

	err := migrator.init()
	if err != nil {
		return fmt.Errorf("faile to initialize migrator: %w", err)
	}

	migrator.migrate()
	if err != nil {
		return fmt.Errorf("unable to read contract list: %w", err)
	}

	return nil
}
