package migrator

import (
	"fmt"

	"github.com/dfuse-io/eosio-boot/config"
	bootops "github.com/dfuse-io/eosio-boot/ops"
	"github.com/eoscanada/eos-go/ecc"
	"go.uber.org/zap"
)

func init() {
	bootops.Register("migration.inject", &OpMigration{})
}

type OpMigration struct {
	DataDir string `json:"data_dir"`
}

func (op *OpMigration) Actions(opPubkey ecc.PublicKey, c *config.OpConfig, in chan interface{}) error {
	migrator := newMigrator(opPubkey, op.DataDir, in)

	err := migrator.init()
	if err != nil {
		return fmt.Errorf("faile to initialize migrator: %w", err)
	}

	migrator.startMigration()
	if err != nil {
		return fmt.Errorf("unable to read contract list: %w", err)
	}

	return nil
}

func (m *Migrator) init() error {
	zlog.Info("setting injector account", zap.String("account", string(m.contract)))
	err := m.newAccountActions(m.opPublicKey, m.actionChan)
	if err != nil {
		return fmt.Errorf("unable to get migrator contract actions: %w", err)
	}
	m.actionChan <- bootops.EndTransaction(m.opPublicKey) // end transaction
	return nil
}

func (m *Migrator) startMigration() {
	contracts, err := ReadContractList(m.dataDir)
	if err != nil {
		zlog.Error("unable to read contract list", zap.Error(err))
		return
	}

	zlog.Info("retrieved contract list", zap.Int("contract_count", len(contracts)))

	for _, contract := range contracts {

		account, err := NewAccountData(m.dataDir, contract)
		if err != nil {
			zlog.Error("unable to initiate account migration", zap.String("contract", contract))
			continue
		}

		err = m.migrateAccount(account)
		if err != nil {
			zlog.Error("unable to process account", zap.String("contract", contract))
			continue
		}
	}
}
