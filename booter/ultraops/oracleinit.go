package ultraops

import (
	"github.com/dfuse-io/eosio-boot/config"
	bootops "github.com/dfuse-io/eosio-boot/ops"
	eos "github.com/eoscanada/eos-go"
	"github.com/eoscanada/eos-go/ecc"
	oracle "github.com/eoscanada/eos-go/oracle"
)

func init() {
	bootops.Register("oracle.init", &OpOracleInit{})
}

type OpOracleInit struct {
	Interval                     uint8       `json:"interval"`
	CacheWindow                  uint32      `json:"cache_window"`
	FinalPriceTableSize          []uint32    `json:"final_price_table_size,omitempty"`
	FinalMovingAverageSettings   []eos.Asset `json:"final_moving_average_settings,omitempty"`
	UltraComprehensiveRateWeight uint32      `json:"ultra_comprehensive_rate_weight"`
}

func (op *OpOracleInit) Actions(opPubkey ecc.PublicKey, c *config.OpConfig, in chan interface{}) error {
	in <- (*bootops.TransactionAction)(oracle.NewInit(op.Interval, op.CacheWindow, op.FinalPriceTableSize, op.FinalMovingAverageSettings, op.UltraComprehensiveRateWeight))
	in <- bootops.EndTransaction(opPubkey) // end transaction
	return nil
}

func (op *OpOracleInit) RequireValidation() bool {
	return true
}
