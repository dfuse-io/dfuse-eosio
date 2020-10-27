package cache

import (
	pbtokenmeta "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/tokenmeta/v1"
	"github.com/eoscanada/eos-go"
)

func ProtoEOSAccountBalanceToOwnedAsset(bal *pbtokenmeta.AccountBalance) *OwnedAsset {
	return &OwnedAsset{
		Owner: eos.AccountName(bal.Account),
		Asset: &eos.ExtendedAsset{
			Asset: eos.Asset{
				Amount: eos.Int64(bal.Amount),
				Symbol: eos.Symbol{
					Precision: uint8(bal.Precision),
					Symbol:    bal.Symbol,
				},
			},
			Contract: eos.AccountName(bal.TokenContract),
		},
	}
}

func AssetToProtoAccountBalance(asset *OwnedAsset) *pbtokenmeta.AccountBalance {
	return &pbtokenmeta.AccountBalance{
		TokenContract: string(asset.Asset.Contract),
		Account:       string(asset.Owner),
		Amount:        uint64(asset.Asset.Asset.Amount),
		Precision:     uint32(asset.Asset.Asset.Precision),
		Symbol:        asset.Asset.Asset.Symbol.Symbol,
	}
}

func lessValueToBool(value int, order SortingOrder) bool {
	if order == ASC {
		if value > 0 {
			return false
		}

		if value < 0 {
			return true
		}
	}
	if value < 0 {
		return false
	}

	if value > 0 {
		return true
	}

	return false
}
