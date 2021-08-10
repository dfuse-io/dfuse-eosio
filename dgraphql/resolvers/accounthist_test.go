package resolvers

import (
	"testing"

	pbaccounthist "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/accounthist/v1"
	"github.com/streamingfast/dgrpc"
	"github.com/stretchr/testify/require"
)

func TestRoot_CheckAccounthistServiceAvailability(t *testing.T) {
	conn, err := dgrpc.NewInternalClient(":9000")
	require.NoError(t, err)
	tests := []struct {
		name        string
		resolver    *Root
		request     GetAccountHistoryActionsArgs
		expectError bool
	}{
		{
			name: "no service available & request account",
			resolver: &Root{
				accounthistClients: &AccounthistClient{},
			},
			request: GetAccountHistoryActionsArgs{
				Account: "eoscanacom",
			},
			expectError: true,
		},
		{
			name: "no service available & request account-contract",
			resolver: &Root{
				accounthistClients: &AccounthistClient{},
			},
			request: GetAccountHistoryActionsArgs{
				Account:  "eoscanacom",
				Contract: s("eosio.token"),
			},
			expectError: true,
		},
		{
			name: "account service available & request account",
			resolver: &Root{
				accounthistClients: &AccounthistClient{
					Account: pbaccounthist.NewAccountHistoryClient(conn),
				},
			},
			request: GetAccountHistoryActionsArgs{
				Account: "eoscanacom",
			},
			expectError: false,
		},
		{
			name: "account service available & request account-contract",
			resolver: &Root{
				accounthistClients: &AccounthistClient{
					Account: pbaccounthist.NewAccountHistoryClient(conn),
				},
			},
			request: GetAccountHistoryActionsArgs{
				Account:  "eoscanacom",
				Contract: s("eosio.token"),
			},
			expectError: true,
		},
		{
			name: "account-contract service available & request account",
			resolver: &Root{
				accounthistClients: &AccounthistClient{
					AccountContract: pbaccounthist.NewAccountContractHistoryClient(conn),
				},
			},
			request: GetAccountHistoryActionsArgs{
				Account: "eoscanacom",
			},
			expectError: true,
		},
		{
			name: "account-contract service available & request account-contract",
			resolver: &Root{
				accounthistClients: &AccounthistClient{
					AccountContract: pbaccounthist.NewAccountContractHistoryClient(conn),
				},
			},
			request: GetAccountHistoryActionsArgs{
				Account:  "eoscanacom",
				Contract: s("eosio.token"),
			},
			expectError: false,
		},
		{
			name: "both service available & request account",
			resolver: &Root{
				accounthistClients: &AccounthistClient{
					Account:         pbaccounthist.NewAccountHistoryClient(conn),
					AccountContract: pbaccounthist.NewAccountContractHistoryClient(conn),
				},
			},
			request: GetAccountHistoryActionsArgs{
				Account: "eoscanacom",
			},
			expectError: false,
		},
		{
			name: "both service available & request account-contract",
			resolver: &Root{
				accounthistClients: &AccounthistClient{
					Account:         pbaccounthist.NewAccountHistoryClient(conn),
					AccountContract: pbaccounthist.NewAccountContractHistoryClient(conn),
				},
			},
			request: GetAccountHistoryActionsArgs{
				Account:  "eoscanacom",
				Contract: s("eosio.token"),
			},
			expectError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.resolver.checkAccounthistServiceAvailability(zlog, test.request)
			if test.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func s(str string) *string {
	return &str
}
