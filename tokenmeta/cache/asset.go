package cache

import (
	"sort"
	"strings"

	"github.com/eoscanada/eos-go"
)

type OwnedAsset struct {
	Owner eos.AccountName    // ex: eoscanadadad
	Asset *eos.ExtendedAsset // ex: 1.23 EOS (eosio.token)
}

type OwnedAssetSorter func([]*OwnedAsset, SortingOrder) []*OwnedAsset

func SortOwnedAssetBySymbolAlpha(assets []*OwnedAsset, order SortingOrder) []*OwnedAsset {
	sort.SliceStable(assets, func(i, j int) bool {
		return compareOwnedAssetBySymbol(assets[i], assets[j], order)
	})
	return assets
}

func SortOwnedAssetByAccountAlpha(assets []*OwnedAsset, order SortingOrder) []*OwnedAsset {
	sort.SliceStable(assets, func(i, j int) bool {
		return compareOwnedAssetByAccount(assets[i], assets[j], order)
	})
	return assets
}

func SortOwnedAssetByTokenAmount(assets []*OwnedAsset, order SortingOrder) []*OwnedAsset {
	sort.SliceStable(assets, func(i, j int) bool {
		return compareOwnedAssetByTokenAmount(assets[i], assets[j], order)
	})
	return assets
}

func SortOwnedAssetByTokenMarketValue(assets []*OwnedAsset, order SortingOrder) []*OwnedAsset {
	//TODO: implement me
	return assets
}

func compareOwnedAssetBySymbol(a, b *OwnedAsset, order SortingOrder) bool {
	value := strings.Compare(a.Asset.Asset.Symbol.Symbol, b.Asset.Asset.Symbol.Symbol)
	if value != 0 {
		return lessValueToBool(value, order)
	}

	value = strings.Compare(string(a.Asset.Contract), string(b.Asset.Contract))
	if value != 0 {
		temp := lessValueToBool(value, ASC)
		return temp
	}

	return compareOwnedAssetByAccount(a, b, ASC)
}

func compareOwnedAssetByAccount(a, b *OwnedAsset, order SortingOrder) bool {
	value := strings.Compare(string(a.Owner), string(b.Owner))

	if value == 0 {
		return compareOwnedAssetBySymbol(a, b, ASC)
	}

	return lessValueToBool(value, order)
}

func compareOwnedAssetByTokenAmount(a, b *OwnedAsset, order SortingOrder) bool {
	if a.Asset.Asset.Amount == b.Asset.Asset.Amount {
		return compareOwnedAssetByAccount(a, b, ASC)
	}
	if order == ASC {
		return a.Asset.Asset.Amount < b.Asset.Asset.Amount
	}

	return a.Asset.Asset.Amount > b.Asset.Asset.Amount
}

// TODO: implement me
func compareOwnedAssetByTokenMarketValue(a, b *OwnedAsset, order SortingOrder) bool {
	return false
}
