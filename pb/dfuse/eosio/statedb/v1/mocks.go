package pbstatedb

import (
	context "context"

	grpc "google.golang.org/grpc"
)

type MockStateClient struct {
	tableRowsStream State_StreamTableRowsClient
}

func NewMockStateClient() *MockStateClient {
	return &MockStateClient{}
}

func (m *MockStateClient) GetABI(ctx context.Context, in *GetABIRequest, opts ...grpc.CallOption) (*GetABIResponse, error) {
	return nil, nil
}

func (m *MockStateClient) GetKeyAccounts(ctx context.Context, in *GetKeyAccountsRequest, opts ...grpc.CallOption) (*GetKeyAccountsResponse, error) {
	return nil, nil
}

func (m *MockStateClient) GetPermissionLinks(ctx context.Context, in *GetPermissionLinksRequest, opts ...grpc.CallOption) (*GetPermissionLinksResponse, error) {
	return nil, nil
}

func (m *MockStateClient) GetTableRow(ctx context.Context, in *GetTableRowRequest, opts ...grpc.CallOption) (*GetTableRowResponse, error) {
	return nil, nil
}

func (m *MockStateClient) StreamTableRows(ctx context.Context, in *StreamTableRowsRequest, opts ...grpc.CallOption) (State_StreamTableRowsClient, error) {
	return m.tableRowsStream, nil
}

func (m *MockStateClient) StreamTableScopes(ctx context.Context, in *StreamTableScopesRequest, opts ...grpc.CallOption) (State_StreamTableScopesClient, error) {
	return nil, nil
}

func (m *MockStateClient) StreamMultiScopesTableRows(ctx context.Context, in *StreamMultiScopesTableRowsRequest, opts ...grpc.CallOption) (State_StreamMultiScopesTableRowsClient, error) {
	return nil, nil
}

func (m *MockStateClient) StreamMultiContractsTableRows(ctx context.Context, in *StreamMultiContractsTableRowsRequest, opts ...grpc.CallOption) (State_StreamMultiContractsTableRowsClient, error) {
	return nil, nil
}

func (m *MockStateClient) SetStreamTableRowsClient(stream State_StreamTableRowsClient) {
	m.tableRowsStream = stream
}
