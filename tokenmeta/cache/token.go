package cache

import (
	"sort"
	"strings"

	pbtokenmeta "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/tokenmeta/v1"
	"go.uber.org/zap"
)

func SortTokensBySymbolAlpha(tokens []*pbtokenmeta.Token, order SortingOrder) []*pbtokenmeta.Token {
	sort.SliceStable(tokens, func(i, j int) bool {
		return compareTokenBySymbol(tokens[i], tokens[j], order)
	})
	return tokens

}

func SortTokensByHolderCount(tokens []*pbtokenmeta.Token, order SortingOrder) []*pbtokenmeta.Token {
	sort.SliceStable(tokens, func(i, j int) bool {
		return compareTokenByHolders(tokens[i], tokens[j], order)
	})
	return tokens

}

// TODO: implement me
func SortTokensByMarketCap(tokens []*pbtokenmeta.Token, order SortingOrder) []*pbtokenmeta.Token {
	return tokens
}

func compareTokenBySymbol(a *pbtokenmeta.Token, b *pbtokenmeta.Token, order SortingOrder) bool {
	value := strings.Compare(a.Symbol, b.Symbol)
	if value == 0 {
		return compareTokenByContract(a, b, ASC)
	}
	return lessValueToBool(value, order)
}

func compareTokenByContract(a *pbtokenmeta.Token, b *pbtokenmeta.Token, order SortingOrder) bool {
	if (a.Contract == b.Contract) && (a.Symbol == b.Symbol) {
		zlog.Error("cannot have two identical token contracts this is a data inconsistency",
			zap.Reflect("token_a", a),
			zap.Reflect("token_b", b),
		)
		return false
	}
	value := strings.Compare(a.Contract, b.Contract)

	if value == 0 {
		return compareTokenBySymbol(a, b, ASC)
	}
	return lessValueToBool(value, order)
}

func compareTokenByHolders(a *pbtokenmeta.Token, b *pbtokenmeta.Token, order SortingOrder) bool {
	if a.Holders == b.Holders {
		return compareTokenByContract(a, b, ASC)
	}

	if order == ASC {
		return a.Holders < b.Holders
	}
	return a.Holders > b.Holders
}

func sortTokenByMarketCap(a *pbtokenmeta.Token, b *pbtokenmeta.Token, order SortingOrder) bool {
	return true
}
