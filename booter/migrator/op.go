package migrator

import (
	"fmt"

	"github.com/eoscanada/eos-go"

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
	migrator := newMigrator()

	zlog.Debug("getting migrator account actions")
	err := migrator.newAccountActions(opPubkey, in)
	if err != nil {
		zlog.Error("unable to get migrator contract actions", zap.Error(err))
		return err
	}
	in <- bootops.EndTransaction(opPubkey) // end transaction

	zlog.Debug("getting migrator contract actions")
	err = migrator.setContractActions(migrator.contract, in)
	if err != nil {
		zlog.Error("unable to get migrator contract actions", zap.Error(err))
		return err
	}
	in <- bootops.EndTransaction(opPubkey) // end transaction

	table := newContractTable(3)
	table["eoscanadacom"] = []*TableRow{
		{
			Key:   "........ehbo5",
			Payer: "eoscanadacom",
			Data:  `{"balance":"3 EOS"}`,
		},
	}
	table["eoscanadacoa"] = []*TableRow{
		{
			Key:   "........ehbo5",
			Payer: "eoscanadacoa",
			Data:  `{"balance":"3 EOS"}`,
		},
	}
	table["eoscanadacob"] = []*TableRow{
		{
			Key:   "EOS",
			Payer: "eoscanadacob",
			Data:  `{"balance":"3 EOS"}`,
		},
	}

	contracts := []eos.AccountName{
		AN("eosio.token"),
	}

	for _, contract := range contracts {
		err = migrator.setContractActions(contract, in)
		if err != nil {
			zlog.Error("unable set mirgator for contract account", zap.String("contract", string(contract)), zap.Error(err))
			return err
		}
		in <- bootops.EndTransaction(opPubkey) // end transaction

		err = migrator.processContractTable(contract, TN("accounts"), table, in)
		if err != nil {
			zlog.Error("unable get accounts for eosio.token accounts", zap.Error(err))
			return err
		}
		in <- bootops.EndTransaction(opPubkey) // end transaction
	}

	return nil
}

func decodeOpPublicKey(c *config.OpConfig, opPubKey string) (ecc.PublicKey, error) {
	privateKey, err := c.GetPrivateKey(opPubKey)
	if err == nil {
		return privateKey.PublicKey(), nil
	}

	pubKey, err := ecc.NewPublicKey(opPubKey)
	if err != nil {
		return ecc.PublicKey{}, fmt.Errorf("reading pubkey: %s", err)
	}
	return pubKey, nil
}
