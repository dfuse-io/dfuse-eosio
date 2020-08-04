package pbstatedb

import (
	context "context"
	"io"
	"strconv"

	"github.com/dfuse-io/logging"
	"go.uber.org/zap"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var zlog = zap.NewNop()

func init() {
	logging.Register("github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/statedb/v1/statedb", &zlog)
}

type MockStateClient struct {
	tableRowsStream State_StreamTableRowsClient

	streamTableRows *MockStreamTableRows
}

type MockStreamTableRows struct {
	*mockStream

	LastIrrBlockID  string              `json:"last_irreversible_block_id"`
	LastIrrBlockNum uint64              `json:"last_irreversible_block_num"`
	UpToBlockID     string              `json:"up_to_block_id"`
	UpToBlockNum    uint64              `json:"up_to_block_num"`
	Rows            []*TableRowResponse `json:"rows"`

	at int
}

func (s *MockStreamTableRows) Recv() (*TableRowResponse, error) {
	if s.at == 0 {
		zlog.Debug("first streaming of mock table rows", zap.Int("row_count", len(s.Rows)))
	}

	if s.at >= len(s.Rows) {
		return nil, io.EOF
	}

	out := s.Rows[s.at]
	zlog.Debug("streaming row at", zap.Int("index", s.at), zap.Reflect("row", out))
	s.at++

	return out, nil
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
	if m.streamTableRows == nil {
		return m.tableRowsStream, nil
	}

	return m.streamTableRows, nil
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

func (m *MockStateClient) SetStreamTableRows(response *MockStreamTableRows) {
	response.mockStream = &mockStream{
		headers: metadata.MD{
			MetdataLastIrrBlockID:  []string{response.LastIrrBlockID},
			MetdataLastIrrBlockNum: []string{strconv.FormatUint(response.LastIrrBlockNum, 10)},
			MetdataUpToBlockID:     []string{response.UpToBlockID},
			MetdataUpToBlockNum:    []string{strconv.FormatUint(response.UpToBlockNum, 10)},
		},
	}

	m.streamTableRows = response
}

type mockStream struct {
	headers  metadata.MD
	trailers metadata.MD
}

func (s *mockStream) Header() (metadata.MD, error) {
	return s.headers, nil
}

func (s *mockStream) Trailer() metadata.MD {
	return s.trailers
}

func (s *mockStream) CloseSend() error {
	return nil
}

func (s *mockStream) Context() context.Context {
	return context.Background()
}

func (s *mockStream) SendMsg(m interface{}) error {
	return nil
}

func (s *mockStream) RecvMsg(m interface{}) error {
	return nil
}
