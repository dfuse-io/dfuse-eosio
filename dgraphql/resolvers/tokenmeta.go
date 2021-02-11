package resolvers

import (
	"context"
	"fmt"
	"strings"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/dfuse-eosio/dgraphql/types"
	pbtokenmeta "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/tokenmeta/v1"
	"github.com/dfuse-io/dgraphql"
	commonTypes "github.com/dfuse-io/dgraphql/types"
	"github.com/dfuse-io/dmetering"
	"github.com/eoscanada/eos-go"
	"github.com/golang/protobuf/proto"
)

var accountBalanceCursorDecoder = dgraphql.NewOpaqueProtoCursorDecoder(func() proto.Message { return &pbtokenmeta.AccountBalanceCursor{} })
var tokenCursorDecoder = dgraphql.NewOpaqueProtoCursorDecoder(func() proto.Message { return &pbtokenmeta.TokenCursor{} })

type SortOrder string

const (
	SortOrderAsc  SortOrder = "ASC"
	SortOrderDesc SortOrder = "DESC"
)

type TokensRequestSortField string

const (
	TokensRequestSortFieldNone    TokensRequestSortField = "NONE"
	TokensRequestSortFieldAlpha   TokensRequestSortField = "ALPHA"
	TokensRequestSortFieldHolders TokensRequestSortField = "HOLDERS"
)

type TokensRequest struct {
	TokenSymbols   *[]string
	TokenContracts *[]string
	Cursor         *string
	Limit          *commonTypes.Uint32
	SortField      TokensRequestSortField
	SortOrder      SortOrder
}

func (r *Root) QueryTokens(ctx context.Context, args *TokensRequest) (*TokenConnection, error) {
	if err := r.RateLimit(ctx, "token"); err != nil {
		return nil, err
	}
	request := &pbtokenmeta.GetTokensRequest{
		SortOrder: pbtokenmeta.SortOrder(pbtokenmeta.SortOrder_value[string(args.SortOrder)]),
		SortField: pbtokenmeta.GetTokensRequest_SortField(pbtokenmeta.GetTokensRequest_SortField_value[string(args.SortField)]),
	}

	if args.TokenContracts != nil {
		request.FilterTokenContracts = *args.TokenContracts
	}

	if args.TokenSymbols != nil {
		request.FilterTokenSymbols = *args.TokenSymbols
	}

	resp, err := r.tokenmetaClient.GetTokens(ctx, request)
	if err != nil {
		return nil, err
	}

	if len(resp.Tokens) == 0 {
		//////////////////////////////////////////////////////////////////////
		// Billable event on GraphQL Query - One Request, Many Outbound Documents ???
		// WARNING: Ingress / Egress bytess is taken care by the middleware
		//////////////////////////////////////////////////////////////////////
		dmetering.EmitWithContext(dmetering.Event{
			Source:         "dgraphql",
			Kind:           "GraphQL Query",
			Method:         "Tokens",
			RequestsCount:  1,
			ResponsesCount: 1,
		}, ctx)
		//////////////////////////////////////////////////////////////////////
		return newEmptyTokenConnection(), nil
	}

	paginator, err := dgraphql.NewPaginator(args.Limit, nil, nil, args.Cursor, 100, tokenCursorDecoder)
	if err != nil {
		return nil, dgraphql.Errorf(ctx, "%s", err)
	}

	eosTokens := PagineableTokens(resp.Tokens)
	paginatedEosTokens := paginator.Paginate(&eosTokens)
	if paginatedEosTokens == nil {
		//////////////////////////////////////////////////////////////////////
		// Billable event on GraphQL Query - One Request, Many Outbound Documents ???
		// WARNING: Ingress / Egress bytess is taken care by the middleware
		//////////////////////////////////////////////////////////////////////
		dmetering.EmitWithContext(dmetering.Event{
			Source:         "dgraphql",
			Kind:           "GraphQL Query",
			Method:         "Tokens",
			RequestsCount:  1,
			ResponsesCount: 1,
		}, ctx)
		//////////////////////////////////////////////////////////////////////
		return newEmptyTokenConnection(), nil

	}
	eosTokens = paginatedEosTokens.(PagineableTokens)

	edges := []*TokenEdge{}
	for _, item := range eosTokens {
		edges = append(edges, newTokenEdge(newToken(item), dgraphql.MustProtoToOpaqueCursor(&pbtokenmeta.TokenCursor{
			Ver:      1,
			Contract: item.Contract,
			Symbol:   item.Symbol,
		}, "token")))
	}

	//////////////////////////////////////////////////////////////////////////////////
	// Billable event on GraphQL Query - One Request, One  Outbound Document per edge
	// WARNING: Ingress / Egress bytess is taken care by the middleware
	//////////////////////////////////////////////////////////////////////////////////
	dmetering.EmitWithContext(dmetering.Event{
		Source:         "dgraphql",
		Kind:           "GraphQL Query",
		Method:         "Tokens",
		RequestsCount:  1,
		ResponsesCount: countMinOne(len(edges)),
	}, ctx)
	//////////////////////////////////////////////////////////////////////

	pageInfo := &PageInfo{HasNextPage: paginator.HasNextPage, HasPreviousPage: paginator.HasPreviousPage}
	if len(edges) != 0 {
		pageInfo.StartCursor = edges[0].cursor
		pageInfo.EndCursor = edges[len(edges)-1].cursor
	}

	return newTokenConnection(edges, pageInfo, newBlockRef(resp.AtBlockId, resp.AtBlockNum)), nil
}

type AccountBalancesRequestSortField string

const (
	AccountBalancesRequestSortFieldNone   AccountBalancesRequestSortField = "NONE"
	AccountBalancesRequestSortFieldAlpha  AccountBalancesRequestSortField = "ALPHA"
	AccountBalancesRequestSortFieldAmount AccountBalancesRequestSortField = "AMOUNT"
)

type AccountBalanceOption string

const (
	AccountBalanceOptionEosIncludeStaked AccountBalanceOption = "EOS_INCLUDE_STAKED"
)

type AccountBalancesRequest struct {
	Account        string
	TokenSymbols   *[]string
	TokenContracts *[]string
	Cursor         *string
	Limit          *commonTypes.Uint32
	Options        *[]AccountBalanceOption
	SortField      AccountBalancesRequestSortField
	SortOrder      SortOrder
}

func (r *Root) QueryAccountBalances(ctx context.Context, args *AccountBalancesRequest) (*AccountBalanceConnection, error) {
	if err := r.RateLimit(ctx, "token"); err != nil {
		return nil, err
	}
	request := &pbtokenmeta.GetAccountBalancesRequest{
		Account:   args.Account,
		Options:   []pbtokenmeta.GetAccountBalancesRequest_Option{},
		SortOrder: pbtokenmeta.SortOrder(pbtokenmeta.SortOrder_value[string(args.SortOrder)]),
		SortField: pbtokenmeta.GetAccountBalancesRequest_SortField(pbtokenmeta.GetAccountBalancesRequest_SortField_value[string(args.SortField)]),
	}

	if args.TokenContracts != nil {
		request.FilterTokenContracts = *args.TokenContracts
	}

	if args.TokenSymbols != nil {
		request.FilterTokenSymbols = *args.TokenSymbols
	}

	if args.Options != nil {
		for _, option := range *args.Options {
			if o, ok := pbtokenmeta.GetAccountBalancesRequest_Option_value[string(option)]; ok {
				request.Options = append(request.Options, pbtokenmeta.GetAccountBalancesRequest_Option(o))
			}
		}
	}

	resp, err := r.tokenmetaClient.GetAccountBalances(ctx, request)
	if err != nil {
		return nil, err
	}

	if len(resp.Balances) == 0 {
		//////////////////////////////////////////////////////////////////////
		// Billable event on GraphQL Query - One Request, Many Outbound Documents ???
		// WARNING: Ingress / Egress bytess is taken care by the middleware
		//////////////////////////////////////////////////////////////////////
		dmetering.EmitWithContext(dmetering.Event{
			Source:         "dgraphql",
			Kind:           "GraphQL Query",
			Method:         "AccountBalances",
			RequestsCount:  1,
			ResponsesCount: 1,
		}, ctx)
		//////////////////////////////////////////////////////////////////////
		return newEmptyAccountBalanceConnection(), nil
	}

	paginator, err := dgraphql.NewPaginator(args.Limit, nil, nil, args.Cursor, 100, accountBalanceCursorDecoder)
	if err != nil {
		return nil, dgraphql.Errorf(ctx, "%s", err)
	}

	accountBalances := PagineableAcccountBalances(resp.Balances)
	paginatedAccountBalances := paginator.Paginate(&accountBalances)
	if paginatedAccountBalances == nil {
		//////////////////////////////////////////////////////////////////////
		// Billable event on GraphQL Query - One Request, Many Outbound Documents ???
		// WARNING: Ingress / Egress bytess is taken care by the middleware
		//////////////////////////////////////////////////////////////////////
		dmetering.EmitWithContext(dmetering.Event{
			Source:         "dgraphql",
			Kind:           "GraphQL Query",
			Method:         "AccountBalances",
			RequestsCount:  1,
			ResponsesCount: 1,
		}, ctx)
		//////////////////////////////////////////////////////////////////////

		return newEmptyAccountBalanceConnection(), nil
	}
	accountBalances = paginatedAccountBalances.(PagineableAcccountBalances)

	edges := []*AccountBalanceEdge{}
	for _, item := range accountBalances {
		edges = append(edges, newAccountBalanceEdge(newAccountBalance(item), dgraphql.MustProtoToOpaqueCursor(&pbtokenmeta.AccountBalanceCursor{
			Ver:      1,
			Contract: item.TokenContract,
			Symbol:   item.Symbol,
			Account:  item.Account,
		}, "account_balance")))
	}

	//////////////////////////////////////////////////////////////////////
	// TODO: is this multiple tokens is X documents?
	// Billable event on GraphQL Query - One Request, Many Outbound Documents ???
	// WARNING: Ingress / Egress bytess is taken care by the middleware
	//////////////////////////////////////////////////////////////////////
	dmetering.EmitWithContext(dmetering.Event{
		Source:         "dgraphql",
		Kind:           "GraphQL Query",
		Method:         "AccountBalances",
		RequestsCount:  1,
		ResponsesCount: countMinOne(len(edges)),
	}, ctx)
	//////////////////////////////////////////////////////////////////////

	pageInfo := &PageInfo{HasNextPage: paginator.HasNextPage, HasPreviousPage: paginator.HasPreviousPage}
	if len(edges) != 0 {
		pageInfo.StartCursor = edges[0].cursor
		pageInfo.EndCursor = edges[len(edges)-1].cursor
	}

	return newAccountBalanceConnection(edges, pageInfo, newBlockRef(resp.AtBlockId, resp.AtBlockNum)), nil
}

type TokenBalancesRequestSortField string

const (
	TokenBalancesRequestSortFieldNone   TokenBalancesRequestSortField = "NONE"
	TokenBalancesRequestSortFieldAlpha  TokenBalancesRequestSortField = "ALPHA"
	TokenBalancesRequestSortFieldAmount TokenBalancesRequestSortField = "AMOUNT"
)

type TokenBalancesRequest struct {
	Contract     string
	Symbol       string
	TokenHolders *[]string
	Cursor       *string
	Limit        *commonTypes.Uint32
	Options      *[]AccountBalanceOption
	SortField    TokenBalancesRequestSortField
	SortOrder    SortOrder
}

func (r *Root) QueryTokenBalances(ctx context.Context, args *TokenBalancesRequest) (*AccountBalanceConnection, error) {
	if err := r.RateLimit(ctx, "token"); err != nil {
		return nil, err
	}
	request := &pbtokenmeta.GetTokenBalancesRequest{
		TokenContract:      args.Contract,
		Options:            []pbtokenmeta.GetTokenBalancesRequest_Option{},
		SortOrder:          pbtokenmeta.SortOrder(pbtokenmeta.SortOrder_value[string(args.SortOrder)]),
		SortField:          pbtokenmeta.GetTokenBalancesRequest_SortField(pbtokenmeta.GetTokenBalancesRequest_SortField_value[string(args.SortField)]),
		FilterTokenSymbols: []string{args.Symbol},
	}

	if args.TokenHolders != nil {
		request.FilterHolderAccounts = *args.TokenHolders
	}

	if args.Options != nil {
		for _, option := range *args.Options {
			if o, ok := pbtokenmeta.GetTokenBalancesRequest_Option_value[string(option)]; ok {
				request.Options = append(request.Options, pbtokenmeta.GetTokenBalancesRequest_Option(o))
			}
		}
	}

	resp, err := r.tokenmetaClient.GetTokenBalances(ctx, request)
	if err != nil {
		return nil, err
	}

	if len(resp.Tokens) <= 0 {
		return nil, dgraphql.Errorf(ctx, "unable to get token with symbol %q on contract %q", args.Symbol, args.Contract)
	}

	contractSymbolBalances := resp.Tokens[0].Balances
	if len(contractSymbolBalances) == 0 {
		dmetering.EmitWithContext(dmetering.Event{
			Source:         "dgraphql",
			Kind:           "GraphQL Query",
			Method:         "TokenBalances",
			RequestsCount:  1,
			ResponsesCount: 1,
		}, ctx)
		return newEmptyAccountBalanceConnection(), nil
	}

	paginator, err := dgraphql.NewPaginator(args.Limit, nil, nil, args.Cursor, 100, accountBalanceCursorDecoder)
	if err != nil {
		return nil, dgraphql.Errorf(ctx, "%s", err)
	}

	accountBalances := PagineableAcccountBalances(contractSymbolBalances)
	paginatedAccountBalances := paginator.Paginate(&accountBalances)
	if paginatedAccountBalances == nil {
		//////////////////////////////////////////////////////////////////////
		// Billable event on GraphQL Query - One Request, Many Outbound Documents ???
		// WARNING: Ingress / Egress bytess is taken care by the middleware
		//////////////////////////////////////////////////////////////////////
		dmetering.EmitWithContext(dmetering.Event{
			Source:         "dgraphql",
			Kind:           "GraphQL Query",
			Method:         "TokenBalances",
			RequestsCount:  1,
			ResponsesCount: 1,
		}, ctx)
		//////////////////////////////////////////////////////////////////////

		return newEmptyAccountBalanceConnection(), nil
	}
	accountBalances = paginatedAccountBalances.(PagineableAcccountBalances)

	edges := []*AccountBalanceEdge{}
	for _, item := range accountBalances {
		edges = append(edges, newAccountBalanceEdge(newAccountBalance(item), dgraphql.MustProtoToOpaqueCursor(&pbtokenmeta.AccountBalanceCursor{
			Ver:      1,
			Contract: item.TokenContract,
			Symbol:   item.Symbol,
			Account:  item.Account,
		}, "account_balance")))
	}

	//////////////////////////////////////////////////////////////////////
	// TODO: is this multiple tokens is X documents?
	// Billable event on GraphQL Query - One Request, Many Outbound Documents ???
	// WARNING: Ingress / Egress bytess is taken care by the middleware
	//////////////////////////////////////////////////////////////////////
	dmetering.EmitWithContext(dmetering.Event{
		Source:         "dgraphql",
		Kind:           "GraphQL Query",
		Method:         "TokenBalances",
		RequestsCount:  1,
		ResponsesCount: countMinOne(len(edges)),
	}, ctx)
	//////////////////////////////////////////////////////////////////////

	pageInfo := &PageInfo{HasNextPage: paginator.HasNextPage, HasPreviousPage: paginator.HasPreviousPage}
	if len(edges) != 0 {
		pageInfo.StartCursor = edges[0].cursor
		pageInfo.EndCursor = edges[len(edges)-1].cursor
	}

	return newAccountBalanceConnection(edges, pageInfo, newBlockRef(resp.AtBlockId, resp.AtBlockNum)), nil
}

//---------------------------
// Token Connection
//----------------------------
type TokenConnection struct {
	Edges    []*TokenEdge
	PageInfo *PageInfo
	BlockRef *BlockRef
}

func newEmptyTokenConnection() *TokenConnection {
	return &TokenConnection{
		Edges:    []*TokenEdge{},
		PageInfo: nil,
		BlockRef: nil,
	}
}

func newTokenConnection(edges []*TokenEdge, pageInfo *PageInfo, blockRef *BlockRef) *TokenConnection {
	return &TokenConnection{
		Edges:    edges,
		PageInfo: pageInfo,
		BlockRef: blockRef,
	}
}

//---------------------------
// Token Edge
//----------------------------
type TokenEdge struct {
	cursor string
	node   *Token
}

func newTokenEdge(node *Token, cursor string) *TokenEdge {
	return &TokenEdge{
		cursor: cursor,
		node:   node,
	}
}

func (e *TokenEdge) Cursor() string { return e.cursor }
func (e *TokenEdge) Node() *Token   { return e.node }

//----------------------------
// BlockRef
//----------------------------
type BlockRef struct {
	blk bstream.BlockRef
}

func newBlockRef(blockID string, blockNum uint64) *BlockRef {
	return &BlockRef{blk: bstream.NewBlockRef(blockID, blockNum)}
}

func (b *BlockRef) Number() types.Uint64 {
	if b.blk != nil {
		return types.Uint64(b.blk.Num())
	}
	return 0
}

func (b *BlockRef) Id() string {
	if b.blk != nil {
		return b.blk.ID()
	}
	return ""
}

//----------------------------
// Token
//----------------------------
type Token struct {
	t *pbtokenmeta.Token
}

func newToken(t *pbtokenmeta.Token) *Token {
	return &Token{
		t: t,
	}
}

func (t *Token) Contract() string              { return t.t.Contract }
func (t *Token) Symbol() string                { return t.t.Symbol }
func (t *Token) Precision() commonTypes.Uint32 { return commonTypes.Uint32(t.t.Precision) }
func (t *Token) Issuer() string                { return t.t.Issuer }
func (t *Token) Holders() types.Uint64         { return types.Uint64(t.t.Holders) }
func (t *Token) MaximumSupply(args *AssetArgs) string {
	return assetToString(t.t.MaximumSupply, t.t.Precision, t.t.Symbol, args)
}
func (t *Token) TotalSupply(args *AssetArgs) string {
	return assetToString(t.t.MaximumSupply, t.t.Precision, t.t.Symbol, args)
}

//---------------------------
// Account Balance Connection
//----------------------------
type AccountBalanceConnection struct {
	Edges    []*AccountBalanceEdge
	PageInfo *PageInfo
	BlockRef *BlockRef
}

func newEmptyAccountBalanceConnection() *AccountBalanceConnection {
	return &AccountBalanceConnection{
		Edges:    []*AccountBalanceEdge{},
		PageInfo: nil,
		BlockRef: &BlockRef{},
	}
}

func newAccountBalanceConnection(edges []*AccountBalanceEdge, pageInfo *PageInfo, blockRef *BlockRef) *AccountBalanceConnection {
	return &AccountBalanceConnection{
		Edges:    edges,
		PageInfo: pageInfo,
		BlockRef: blockRef,
	}
}

//---------------------------
// Account Balance Edge
//----------------------------
type AccountBalanceEdge struct {
	cursor string
	node   *AccountBalance
}

func newAccountBalanceEdge(node *AccountBalance, cursor string) *AccountBalanceEdge {
	return &AccountBalanceEdge{
		cursor: cursor,
		node:   node,
	}
}

func (e *AccountBalanceEdge) Cursor() string        { return e.cursor }
func (e *AccountBalanceEdge) Node() *AccountBalance { return e.node }

//----------------------------
// Account Balance
//----------------------------
type AccountBalance struct {
	a *pbtokenmeta.AccountBalance
}

func newAccountBalance(a *pbtokenmeta.AccountBalance) *AccountBalance {
	return &AccountBalance{
		a: a,
	}
}

func (a *AccountBalance) Contract() string              { return a.a.TokenContract }
func (a *AccountBalance) Account() string               { return a.a.Account }
func (a *AccountBalance) Amount() types.Uint64          { return types.Uint64(a.a.Amount) }
func (a *AccountBalance) Symbol() string                { return a.a.Symbol }
func (a *AccountBalance) Precision() commonTypes.Uint32 { return commonTypes.Uint32(a.a.Precision) }
func (a *AccountBalance) Balance(args *AssetArgs) string {
	return assetToString(a.a.Amount, a.a.Precision, a.a.Symbol, args)
}

//----------------------------
// EOS Token Collection
//----------------------------
type PagineableTokens []*pbtokenmeta.Token

func (e PagineableTokens) IsEqual(index int, key string) bool {
	return e[index].Key() == key
}

func (e PagineableTokens) Append(slice dgraphql.Pagineable, index int) dgraphql.Pagineable {
	if slice == nil {
		var arr PagineableTokens = []*pbtokenmeta.Token{e[index]}
		return dgraphql.Pagineable(arr)
	} else {
		return dgraphql.Pagineable(append(slice.(PagineableTokens), e[index]))
	}
}

func (e PagineableTokens) Length() int {
	return len(e)
}

type PagineableAcccountBalances []*pbtokenmeta.AccountBalance

func (p PagineableAcccountBalances) IsEqual(index int, key string) bool {
	return p[index].Key() == key
}

func (p PagineableAcccountBalances) Append(slice dgraphql.Pagineable, index int) dgraphql.Pagineable {
	if slice == nil {
		var arr PagineableAcccountBalances = []*pbtokenmeta.AccountBalance{p[index]}
		return dgraphql.Pagineable(arr)
	} else {
		return dgraphql.Pagineable(append(slice.(PagineableAcccountBalances), p[index]))
	}
}

func (p PagineableAcccountBalances) Length() int {
	return len(p)
}

type AssetArgs struct {
	Format AssetFormat
}

type AssetFormat string

const (
	AssetFormatAsset   AssetFormat = "ASSET"
	AssetFormatInteger AssetFormat = "INTEGER"
	AssetFormatDecimal AssetFormat = "DECIMAL"
)

func assetToString(amount uint64, precision uint32, symbol string, args *AssetArgs) string {
	if args == nil || args.Format == AssetFormatAsset {
		return eos.Asset{
			Amount: eos.Int64(amount),
			Symbol: eos.Symbol{
				Precision: uint8(precision),
				Symbol:    symbol,
			},
		}.String()
	}

	if args.Format == AssetFormatInteger {
		return fmt.Sprintf("%d", amount)
	}

	amt := amount
	if amt < 0 {
		amt = -amt
	}
	strInt := fmt.Sprintf("%d", amt)
	if len(strInt) < int(precision+1) {
		// prepend `0` for the difference:
		strInt = strings.Repeat("0", int(uint8(precision)+uint8(1))-len(strInt)) + strInt
	}

	var result string
	if precision == 0 {
		result = strInt
	} else {
		result = strInt[:len(strInt)-int(precision)] + "." + strInt[len(strInt)-int(precision):]
	}
	if amount < 0 {
		result = "-" + result
	}
	return fmt.Sprintf("%s", result)
}
