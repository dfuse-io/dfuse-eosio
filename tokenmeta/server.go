package tokenmeta

import (
	"context"
	"fmt"
	"net"
	"time"

	pbtokenmeta "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/tokenmeta/v1"
	"github.com/dfuse-io/dfuse-eosio/tokenmeta/cache"
	"github.com/streamingfast/dgrpc"
	pbhealth "github.com/streamingfast/pbgo/grpc/health/v1"
	"github.com/streamingfast/shutter"
	"github.com/eoscanada/eos-go"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type Server struct {
	*shutter.Shutter

	grpcServer          *grpc.Server
	cache               cache.Cache
	readinessMaxLatency time.Duration
}

func NewServer(cache cache.Cache, readinessMaxLatency time.Duration) *Server {
	s := &Server{
		readinessMaxLatency: readinessMaxLatency,
		Shutter:             shutter.New(),
		cache:               cache,
		grpcServer:          dgrpc.NewServer(dgrpc.WithLogger(zlog)),
	}

	pbtokenmeta.RegisterTokenMetaServer(s.grpcServer, s)
	pbhealth.RegisterHealthServer(s.grpcServer, s)

	return s
}

func (s *Server) Check(ctx context.Context, in *pbhealth.HealthCheckRequest) (*pbhealth.HealthCheckResponse, error) {
	status := pbhealth.HealthCheckResponse_SERVING

	if s.IsTerminating() {
		status = pbhealth.HealthCheckResponse_NOT_SERVING
	}
	if s.readinessMaxLatency > 0 {
		headBlkTime := s.cache.GetHeadBlockTime()
		if headBlkTime.IsZero() || time.Since(headBlkTime) > s.readinessMaxLatency {
			status = pbhealth.HealthCheckResponse_NOT_SERVING
		}
	}

	return &pbhealth.HealthCheckResponse{
		Status: status,
	}, nil
}

func (s *Server) Serve(listenAddr string) {
	zlog.Info("starting grpc server", zap.String("address", listenAddr))
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		s.Shutdown(fmt.Errorf("unable to listen on %q: %w", listenAddr, err))
		return
	}

	err = s.grpcServer.Serve(listener)
	if err == nil || err == grpc.ErrServerStopped {
		zlog.Info("server shut down cleanly, nothing to do")
		return
	}

	if err != nil {
		s.Shutdown(err)
	}
}

func (s *Server) Close() {
	s.grpcServer.GracefulStop()
}

// func (s *Server) Ready() bool {
// 	return true
// }

func (s *Server) GetTokens(ctx context.Context, in *pbtokenmeta.GetTokensRequest) (*pbtokenmeta.TokensResponse, error) {
	zlog.Debug("get tokens",
		zap.Strings("filter_token_contracts", in.FilterTokenContracts),
		zap.Strings("filter_token_symbols", in.FilterTokenSymbols),
		zap.Uint32("limit", in.Limit),
		zap.String("order", in.SortOrder.String()),
		zap.String("filed", in.SortField.String()),
	)

	tokens := []*pbtokenmeta.Token{}
	for _, t := range s.cache.Tokens() {
		if matchFilters(eos.AccountName(t.Contract), t.Symbol, in.FilterTokenContracts, in.FilterTokenSymbols) {
			tokens = append(tokens, t)
		}
	}

	tokens = sortGetTokens(tokens, in.SortField, in.SortOrder)
	tokens = limitTokenResults(tokens, in.Limit)
	blockRef := s.cache.AtBlockRef()

	out := &pbtokenmeta.TokensResponse{
		Tokens:     []*pbtokenmeta.Token{},
		AtBlockNum: blockRef.Num(),
		AtBlockId:  blockRef.ID(),
	}
	for _, t := range tokens {
		out.Tokens = append(out.Tokens, t)
	}

	return out, nil
}

func (s *Server) GetAccountBalances(ctx context.Context, in *pbtokenmeta.GetAccountBalancesRequest) (*pbtokenmeta.AccountBalancesResponse, error) {
	zlog.Debug("get account balances",
		zap.Strings("filter_token_contracts", in.FilterTokenContracts),
		zap.Strings("filter_token_symbols", in.FilterTokenSymbols),
		zap.Uint32("limit", in.Limit),
		zap.Any("options", in.Options),
		zap.String("order", in.SortOrder.String()),
		zap.String("account_holder", in.Account),
	)

	options := []cache.AccountBalanceOption{}
	if hasAccountOption(in.Options, pbtokenmeta.GetAccountBalancesRequest_EOS_INCLUDE_STAKED) {
		options = append(options, cache.EOSIncludeStakedAccOpt)
	}

	assets := []*cache.OwnedAsset{}
	for _, a := range s.cache.AccountBalances(eos.AccountName(in.Account), options...) {
		if matchFilters(a.Asset.Contract, a.Asset.Asset.Symbol.Symbol, in.FilterTokenContracts, in.FilterTokenSymbols) {
			assets = append(assets, a)
		}
	}
	assets = sortAccountBalances(assets, in.SortField, in.SortOrder)
	assets = limitAssetsResults(assets, in.Limit)
	blockRef := s.cache.AtBlockRef()

	out := &pbtokenmeta.AccountBalancesResponse{
		Balances:   []*pbtokenmeta.AccountBalance{},
		AtBlockNum: blockRef.Num(),
		AtBlockId:  blockRef.ID(),
	}

	for _, a := range assets {
		out.Balances = append(out.Balances, cache.AssetToProtoAccountBalance(a))
	}

	return out, nil
}

func hasAccountOption(opts []pbtokenmeta.GetAccountBalancesRequest_Option, opt pbtokenmeta.GetAccountBalancesRequest_Option) bool {
	for _, o := range opts {
		if o == opt {
			return true
		}
	}
	return false
}

func hasTokenOption(opts []pbtokenmeta.GetTokenBalancesRequest_Option, opt pbtokenmeta.GetTokenBalancesRequest_Option) bool {
	for _, o := range opts {
		if o == opt {
			return true
		}
	}
	return false
}

func (s *Server) GetTokenBalances(ctx context.Context, in *pbtokenmeta.GetTokenBalancesRequest) (*pbtokenmeta.TokenBalancesResponse, error) {
	zlog.Debug("get token balances",
		zap.Strings("filter_holder_accounts", in.FilterHolderAccounts),
		zap.Strings("filter_token_symbols", in.FilterTokenSymbols),
		zap.Uint32("limit", in.Limit),
		zap.String("order", in.SortOrder.String()),
		zap.String("token_contract", in.TokenContract),
	)

	options := []cache.TokenBalanceOption{}
	if hasTokenOption(in.Options, pbtokenmeta.GetTokenBalancesRequest_EOS_INCLUDE_STAKED) {
		options = append(options, cache.EOSIncludeStakedTokOpt)
	}
	assets := []*cache.OwnedAsset{}
	for _, a := range s.cache.TokenBalances(eos.AccountName(in.TokenContract), options...) {
		if matchFilters(a.Asset.Contract, a.Asset.Asset.Symbol.Symbol, []string{}, in.FilterTokenSymbols) {
			if stringInFilter(string(a.Owner), in.FilterHolderAccounts) {
				assets = append(assets, a)
			}
		}
	}
	assets = sortTokenBalances(assets, in.SortField, in.SortOrder)
	// Limit by token? the full list
	assets = limitAssetsResults(assets, in.Limit)
	blockRef := s.cache.AtBlockRef()

	out := &pbtokenmeta.TokenBalancesResponse{
		Tokens:     []*pbtokenmeta.TokenContractBalancesResponse{},
		AtBlockNum: blockRef.Num(),
		AtBlockId:  blockRef.ID(),
	}

	symbolIndex := map[string]int{}
	for _, a := range assets {
		if index, ok := symbolIndex[a.Asset.Asset.Symbol.Symbol]; ok {
			out.Tokens[index].Balances = append(out.Tokens[index].Balances, cache.AssetToProtoAccountBalance(a))
		} else {
			out.Tokens = append(out.Tokens, &pbtokenmeta.TokenContractBalancesResponse{
				Token: &pbtokenmeta.Token{
					Contract:  string(a.Asset.Contract),
					Symbol:    a.Asset.Asset.Symbol.Symbol,
					Precision: uint32(a.Asset.Asset.Symbol.Precision),
				},
				Balances: []*pbtokenmeta.AccountBalance{cache.AssetToProtoAccountBalance(a)},
			})
			symbolIndex[a.Asset.Asset.Symbol.Symbol] = len(out.Tokens) - 1
		}
	}
	return out, nil
}

func matchFilters(contract eos.AccountName, symbol string, contractFilter []string, symbolFilter []string) bool {
	if !stringInFilter(symbol, symbolFilter) {
		return false
	}

	if !stringInFilter(string(contract), contractFilter) {
		return false
	}

	return true
}

func sortGetTokens(tokens []*pbtokenmeta.Token, sortField pbtokenmeta.GetTokensRequest_SortField, sortOrder pbtokenmeta.SortOrder) []*pbtokenmeta.Token {
	switch sortField {
	case pbtokenmeta.GetTokensRequest_NONE:
		return tokens
	case pbtokenmeta.GetTokensRequest_ALPHA:
		return cache.SortTokensBySymbolAlpha(tokens, sortOrderMapper(sortOrder))
	case pbtokenmeta.GetTokensRequest_HOLDERS:
		return cache.SortTokensByHolderCount(tokens, sortOrderMapper(sortOrder))
	case pbtokenmeta.GetTokensRequest_MARKET_CAP:
		// TODO: implement me
		return tokens
	}
	return tokens
}

func sortAccountBalances(assets []*cache.OwnedAsset, sortField pbtokenmeta.GetAccountBalancesRequest_SortField, sortOrder pbtokenmeta.SortOrder) []*cache.OwnedAsset {
	switch sortField {
	case pbtokenmeta.GetAccountBalancesRequest_NONE:
		return assets
	case pbtokenmeta.GetAccountBalancesRequest_ALPHA:
		return cache.SortOwnedAssetBySymbolAlpha(assets, sortOrderMapper(sortOrder))
	case pbtokenmeta.GetAccountBalancesRequest_AMOUNT:
		return cache.SortOwnedAssetByTokenAmount(assets, sortOrderMapper(sortOrder))
	case pbtokenmeta.GetAccountBalancesRequest_MARKET_VALUE:
		// TODO: implement me
		return assets
	}
	return assets
}

func sortTokenBalances(assets []*cache.OwnedAsset, sortField pbtokenmeta.GetTokenBalancesRequest_SortField, sortOrder pbtokenmeta.SortOrder) []*cache.OwnedAsset {
	switch sortField {
	case pbtokenmeta.GetTokenBalancesRequest_NONE:
		return assets
	case pbtokenmeta.GetTokenBalancesRequest_ALPHA:
		return cache.SortOwnedAssetByAccountAlpha(assets, sortOrderMapper(sortOrder))
	case pbtokenmeta.GetTokenBalancesRequest_AMOUNT:
		return cache.SortOwnedAssetByTokenAmount(assets, sortOrderMapper(sortOrder))
	case pbtokenmeta.GetTokenBalancesRequest_MARKET_VALUE:
		// TODO: implement me
		return assets
	}
	return assets
}

func limitTokenResults(results []*pbtokenmeta.Token, limit uint32) (out []*pbtokenmeta.Token) {
	if limit == 0 {
		return results
	}

	if limit > uint32(len(results)) {
		return results
	}

	return results[:limit]
}

func limitAssetsResults(results []*cache.OwnedAsset, limit uint32) (out []*cache.OwnedAsset) {
	if limit == 0 {
		return results
	}

	if limit > uint32(len(results)) {
		return results
	}

	return results[:limit]
}

func sortOrderMapper(order pbtokenmeta.SortOrder) cache.SortingOrder {
	switch order {
	case pbtokenmeta.SortOrder_ASC:
		return cache.ASC
	case pbtokenmeta.SortOrder_DESC:
		return cache.DESC
	}
	return cache.ASC
}
