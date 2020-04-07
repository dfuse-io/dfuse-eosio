package fluxdb

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/dfuse-io/derr"
	"github.com/eoscanada/eos-go"
)

type Client interface {
	GetABI(ctx context.Context, startBlock uint32, account eos.AccountName) (*GetABIResponse, error)
	GetTable(ctx context.Context, startBlock uint32, request *GetTableRequest) (*GetTableResponse, error)
	GetTableScopes(ctx context.Context, startBlock uint32, request *GetTableScopesRequest) (*GetTableScopesResponse, error)
	GetTablesMultiScopes(ctx context.Context, startBlock uint32, request *GetTablesMultiScopesRequest) (*GetTablesMultiScopesResponse, error)
	GetAccountByPubKey(ctx context.Context, startBlock uint32, pubKey string) (*GetAccountByPubKeyResponses, error)
}

type DefaultClient struct {
	addr       string
	httpClient *http.Client
}

func NewClient(addr string, transport http.RoundTripper) *DefaultClient {
	return &DefaultClient{
		addr: addr,
		httpClient: &http.Client{
			Transport: transport,
		},
	}
}

type GetAccountByPubKeyResponses struct {
	BlockNum     uint32            `json:"block_num"`
	AccountNames []eos.AccountName `json:"account_names"`
}

func (c *DefaultClient) GetAccountByPubKey(ctx context.Context, startBlock uint32, pubKey string) (*GetAccountByPubKeyResponses, error) {
	val := url.Values{}
	val.Set("block_num", fmt.Sprintf("%d", startBlock))
	val.Set("public_key", pubKey)

	body, err := c.performFormRequest(ctx, "/v0/state/key_accounts", val)
	if err != nil {
		return nil, derr.Wrap(err, "unable to get account for key")
	}

	var response *GetAccountByPubKeyResponses
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, derr.Wrap(err, "unable to decode response")
	}

	return response, nil
}

type GetABIResponse struct {
	BlockNum uint32          `json:"block_num"`
	Account  eos.AccountName `json:"account"`
	ABI      *eos.ABI        `json:"abi"`
}

func (c *DefaultClient) GetABI(ctx context.Context, startBlock uint32, account eos.AccountName) (*GetABIResponse, error) {
	val := url.Values{}
	val.Set("block_num", fmt.Sprintf("%d", startBlock))
	val.Set("account", string(account))
	val.Set("json", "true")

	body, err := c.performFormRequest(ctx, "/v0/state/abi", val)
	if err != nil {
		return nil, derr.Wrap(err, "unable to get abi")
	}

	var response *GetABIResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, derr.Wrap(err, "unable to decode response")
	}

	return response, nil
}

type GetTableRequest struct {
	Account eos.AccountName
	Scope   eos.Name
	Table   eos.TableName
	KeyType string
	JSON    bool
}

type GetTablesMultiScopesRequest struct {
	Account eos.AccountName
	Scopes  []eos.Name
	Table   eos.TableName
	KeyType string
	JSON    bool
}

type GetTableScopesRequest struct {
	Account eos.AccountName
	Table   eos.TableName
}

type GetTableScopesResponse struct {
	Block_num uint32     `json:"block_num"`
	Scopes    []eos.Name `json:"scopes"`
}

func NewGetTableRequest(account eos.AccountName, scope eos.Name, table eos.TableName, keyType string) *GetTableRequest {
	return &GetTableRequest{
		Account: account,
		Scope:   scope,
		Table:   table,
		KeyType: keyType,
		JSON:    true,
	}
}

type GetTableResponse struct {
	LastIrreversibleBlockID  string `json:"last_irreversible_block_id"`
	LastIrreversibleBlockNum uint32 `json:"last_irreversible_block_num"`
	UpToBlockID              string `json:"up_to_block_id"`
	UpToBlockNum             uint32 `json:"up_to_block_num"`

	ABI  *eos.ABI
	Rows json.RawMessage `json:"rows"`
}

type GetTablesMultiScopesResponse struct {
	LastIrreversibleBlockID  string `json:"last_irreversible_block_id"`
	LastIrreversibleBlockNum uint32 `json:"last_irreversible_block_num"`
	UpToBlockID              string `json:"up_to_block_id"`
	UpToBlockNum             uint32 `json:"up_to_block_num"`

	Tables []struct {
		Scope string          `json:"scope"`
		Rows  json.RawMessage `json:"rows"`
	} `json:"tables"`
}

func (c *DefaultClient) GetTablesMultiScopes(ctx context.Context, startBlock uint32, request *GetTablesMultiScopesRequest) (*GetTablesMultiScopesResponse, error) {
	val := url.Values{}
	val.Set("block_num", fmt.Sprintf("%d", startBlock))
	val.Set("account", string(request.Account))
	scopes := ""
	for _, s := range request.Scopes {
		if scopes == "" {
			scopes = string(s)
		} else {
			scopes = fmt.Sprintf("%s|%s", scopes, string(s))
		}
	}
	val.Set("scopes", scopes)
	val.Set("table", string(request.Table))
	val.Set("key_type", request.KeyType)
	val.Set("json", fmt.Sprintf("%v", request.JSON))

	body, err := c.performFormRequest(ctx, "/v0/state/tables/scopes", val)
	if err != nil {
		return nil, derr.Wrap(err, "unable to get table")
	}

	var response *GetTablesMultiScopesResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, derr.Wrap(err, "unable to decode response")
	}

	return response, nil
}

func (c *DefaultClient) GetTable(ctx context.Context, startBlock uint32, request *GetTableRequest) (*GetTableResponse, error) {
	val := url.Values{}
	val.Set("block_num", fmt.Sprintf("%d", startBlock))
	val.Set("account", string(request.Account))
	val.Set("scope", string(request.Scope))
	val.Set("table", string(request.Table))
	val.Set("key_type", request.KeyType)
	val.Set("json", fmt.Sprintf("%v", request.JSON))

	body, err := c.performFormRequest(ctx, "/v0/state/table", val)
	if err != nil {
		return nil, derr.Wrap(err, "unable to get table")
	}

	var response *GetTableResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, derr.Wrap(err, "unable to decode response")
	}

	return response, nil
}

func (c *DefaultClient) GetTableScopes(ctx context.Context, startBlock uint32, request *GetTableScopesRequest) (*GetTableScopesResponse, error) {
	val := url.Values{}
	val.Set("block_num", fmt.Sprintf("%d", startBlock))
	val.Set("account", string(request.Account))
	val.Set("table", string(request.Table))

	body, err := c.performFormRequest(ctx, "/v0/state/table_scopes", val)
	if err != nil {
		return nil, derr.Wrap(err, "unable to get table scopes")
	}

	var response *GetTableScopesResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, derr.Wrap(err, "unable to decode response")
	}

	return response, nil
}

func (c *DefaultClient) performFormRequest(ctx context.Context, path string, form url.Values) (body []byte, err error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s%s?%s", c.addr, path, form.Encode()), nil)
	if err != nil {
		return nil, fmt.Errorf("unable to create new request: %s", err)
	}

	req = req.WithContext(ctx)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("unable to perform HTTP request: %s", err)
	}

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("unable to read HTTP response body: %s", err)
	}

	if resp.StatusCode != 200 {
		return nil, bodyToError(resp.StatusCode, body)
	}

	return
}

func bodyToError(statusCode int, body []byte) error {
	var errorResponse *derr.ErrorResponse
	if err := json.Unmarshal(body, &errorResponse); err == nil {
		// Status code is not serialized, let's re-hydrate it from HTTP response code, which should be the same
		errorResponse.Status = statusCode
		return errorResponse
	}

	return fmt.Errorf("request failed, status code was %d, body: %q", statusCode, string(body))
}
