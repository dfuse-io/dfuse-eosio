package searchclient_test

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/dfuse-io/dfuse-eosio/eosdb"
	searchclient "github.com/dfuse-io/dfuse-eosio/search-client"
	"github.com/dfuse-io/dgrpc"
	"github.com/dfuse-io/logging"
	pbsearch "github.com/dfuse-io/pbgo/dfuse/search/v1"
	"go.uber.org/zap"
)

func init() {
	if os.Getenv("DEBUG") != "" {
		logger, _ := zap.NewDevelopment()
		logging.Override(logger)
	}
}

// Remove the leading # just before 'Output' at very end of ExampleEOSClient see actual execution results!
func ExampleEOSClient() {
	kvdbDSN, searchAddr, err := getEOSConfig()
	if err != nil {
		fmt.Println(err)
		return
	}

	dbReader, err := eosdb.New(kvdbDSN)
	if err != nil {
		fmt.Println(fmt.Errorf("unable to create EOS database instance: %w", err))
		return
	}

	searchConn, err := dgrpc.NewInternalClient(searchAddr)
	if err != nil {
		fmt.Println(fmt.Errorf("unable to create search connection: %w", err))
		return
	}

	client := searchclient.NewEOSClient(searchConn, dbReader)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream, err := client.StreamMatches(ctx, &pbsearch.RouterRequest{
		Query:          "account:eosio.token action:transfer",
		LowBlockNum:    110521022,
		HighBlockNum:   110521023,
		Limit:          5,
		WithReversible: true,
		Mode:           pbsearch.RouterRequest_STREAMING,
	})

	if err != nil {
		fmt.Println("Unable to stream matches", err)
		return
	}

	for {
		match, err := stream.Recv()
		if err != nil {
			if err != io.EOF {
				fmt.Printf("Client error: %s\n", err)
			}

			return
		}

		if match.TransactionTrace == nil {
			fmt.Printf("Live marker at block %s, continuing\n", match.BlockID)
			continue
		}

		fmt.Printf("Match %s (%s), %d actions\n",
			match.TransactionTrace.Id,
			match.BlockID,
			len(match.MatchingActions),
		)

		for _, action := range match.MatchingActions {
			fmt.Printf(" - Action #%d - %s\n", action.ActionOrdinal, action.SimpleName())
		}
	}

	// #Output: any
}

func getEOSConfig() (kvdbDSN, searchAddr string, err error) {
	kvdbDSN = os.Getenv("KVDB_DSN")
	if kvdbDSN == "" {
		return "", "", fmt.Errorf("the environment variable KVDB_DSN must be set, for example 'bigtable://dfuse-io.dfuse-saas/mainnet-v4'")
	}

	searchAddr = os.Getenv("SEARCH_ADDR")
	if searchAddr == "" {
		return "", "", fmt.Errorf("the environment variable SEARCH_ADDR must be set, for example 'localhost:9001'")
	}

	return
}
