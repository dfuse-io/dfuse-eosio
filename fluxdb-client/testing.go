package fluxdb

import (
	"context"
	"encoding/json"

	"github.com/eoscanada/eos-go"
)

type getABIResponse struct {
	response *GetABIResponse
	err      error
}

type getTableResponse struct {
	response *GetTableResponse
	err      error
}

type TestClient struct {
	getABIResponse   *getABIResponse
	getTableResponse *getTableResponse
}

func (c *TestClient) GetTableScopes(ctx context.Context, startBlock uint32, request *GetTableScopesRequest) (*GetTableScopesResponse, error) {
	panic("implement me")
}

func (c *TestClient) GetTablesMultiScopes(ctx context.Context, startBlock uint32, request *GetTablesMultiScopesRequest) (*GetTablesMultiScopesResponse, error) {
	panic("implement me")
}

func (c *TestClient) GetAccountByPubKey(ctx context.Context, startBlock uint32, pubKey string) (*GetAccountByPubKeyResponses, error) {
	panic("implement me")
}

func NewTestFluxClient() *TestClient {
	return &TestClient{}
}

func (c *TestClient) SetGetABIResponse(response string, err error) *TestClient {
	var unmarshalledResponse *GetABIResponse

	return c.setResponse(response, &unmarshalledResponse, func() {
		c.getABIResponse = &getABIResponse{
			response: unmarshalledResponse,
			err:      err,
		}
	})
}

func (c *TestClient) SetGetTableResponse(response string, err error) *TestClient {
	var unmarshalledResponse *GetTableResponse

	return c.setResponse(response, &unmarshalledResponse, func() {
		c.getTableResponse = &getTableResponse{
			response: unmarshalledResponse,
			err:      err,
		}
	})
}

func (c *TestClient) setResponse(response string, receiver interface{}, setter func()) *TestClient {
	if response != "" {
		merr := json.Unmarshal([]byte(response), receiver)
		if merr != nil {
			panic(merr)
		}
	}

	setter()

	return c
}

func (c *TestClient) GetABI(ctx context.Context, startBlock uint32, account eos.AccountName) (*GetABIResponse, error) {
	return c.getABIResponse.response, c.getABIResponse.err
}

func (c *TestClient) GetTable(ctx context.Context, startBlock uint32, request *GetTableRequest) (*GetTableResponse, error) {
	return c.getTableResponse.response, c.getTableResponse.err
}
