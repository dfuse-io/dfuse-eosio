package dgraphql

import (
	"encoding/json"
	"fmt"
	"time"

	rice "github.com/GeertJohan/go.rice"
	"github.com/dfuse-io/dgraphql/static"
)

//go:generate rice embed-go

// GraphqlExamples returns the ordered list of predefined GraphQL examples that should be
// displayed inside GraphiQL interface.
func GraphqlExamples() []*static.GraphqlExample {
	box := rice.MustFindBox("graphql-examples")

	return []*static.GraphqlExample{
		{
			Label:    "Search Stream (Forward)",
			Document: graphqlDocument(box, "search_stream_forward.graphql"),
			Variables: static.GraphqlVariablesByNetwork{
				"generic": r(`{"query": "receiver:eosio action:newaccount", "cursor": "", "limit": 10}`),
				"mainnet": r(`{"query": "receiver:eosio.token action:transfer -data.quantity:'0.0001 EOS'", "cursor": "", "limit": 10}`),
				"jungle":  r("mainnet"),
				"kylin":   r("mainnet"),
			},
		},
		{
			Label:    "Search Query (Forward)",
			Document: graphqlDocument(box, "search_query_forward.graphql"),
			Variables: static.GraphqlVariablesByNetwork{
				"generic": r(`{"query": "receiver:eosio action:newaccount", "cursor": "", "limit": 10}`),
				"mainnet": r(`{"query": "receiver:eosio.token action:transfer -data.quantity:'0.0001 EOS'", "low": -500, "high": -1, "cursor": "", "limit": 10}`),
				"jungle":  r("mainnet"),
				"kylin":   r("mainnet"),
			},
		},
		{
			Label:    "Search Stream (Backward)",
			Document: graphqlDocument(box, "search_stream_backward.graphql"),
			Variables: static.GraphqlVariablesByNetwork{
				"generic": r(`{"query": "receiver:eosio action:newaccount", "cursor": "", "low": 1, "limit": 10}`),
				"mainnet": r(`{"query": "receiver:eosio.token action:transfer -data.quantity:'0.0001 EOS'", "cursor": "", "low": 1, "limit": 10}`),
				"jungle":  r("mainnet"),
				"kylin":   r("mainnet"),
			},
		},
		{
			Label:    "Search Query (Backward)",
			Document: graphqlDocument(box, "search_query_backward.graphql"),
			Variables: static.GraphqlVariablesByNetwork{
				"generic": r(`{"query": "receiver:eosio action:newaccount", "low": -500, "high": -1, "cursor": "", "limit": 10}`),
				"mainnet": r(`{"query": "receiver:eosio.token action:transfer -data.quantity:'0.0001 EOS'", "low": -500, "high": -1, "cursor": "", "limit": 10}`),
				"jungle":  r("mainnet"),
				"kylin":   r("mainnet"),
			},
		},
		{
			Label:    "Time Ranges",
			Document: graphqlDocument(box, "time_ranges.graphql"),
			Variables: static.GraphqlVariablesByNetwork{
				"generic": r(fmt.Sprintf(`{"start": "%s", "end": "%s"}`, dateOffsetByBlock(0), dateOffsetByBlock(5))),
				"mainnet": r(fmt.Sprintf(`{"start": "%s", "end": "%s"}`, oneWeekAgo(), dateOffsetByBlock(-1))),
				"jungle":  r("mainnet"),
				"kylin":   r("mainnet"),
			},
		},
		{
			Label:    "Get Block By Id (Alpha)",
			Document: graphqlDocument(box, "get_block_by_id.graphql"),
			Variables: static.GraphqlVariablesByNetwork{
				"generic": r(`{"blockId": "<Block ID Here>"}`),
				"mainnet": r(`{"blockId": "063a7e525142f64d7465bbebc690afbb228bff7d7e0ffda31d9a06106fbc1982"}`),
				"jungle":  r(`{"blockId": "047c7822f396e64b9cbb28cc2b199b8e5a4c33c894b8742eab646e670486bb0d"}`),
				"kylin":   r(`{"blockId": "05609e94b57cdea5ce4ff8afa89070d37a85923855f0d41efdbd956dbaddb5f7"}`),
			},
		},
		{
			Label:    "Get Block By Num (Alpha) ",
			Document: graphqlDocument(box, "get_block_by_num.graphql"),
			Variables: static.GraphqlVariablesByNetwork{
				"generic": r(`{"blockNum": 10}`),
				"mainnet": r(`{"blockNum": 104782163}`),
				"jungle":  r(`{"blockNum": 22430834}`),
				"kylin":   r(`{"blockNum": 90340699}`),
			},
		},
		{
			Label:     "Get Tokens (Alpha)",
			Document:  graphqlDocument(box, "get_tokens.graphql"),
			Variables: nil,
		},
		{
			Label:    "Get Token Balances (Alpha)",
			Document: graphqlDocument(box, "get_token_balances.graphql"),
			Variables: static.GraphqlVariablesByNetwork{
				"generic": r(`{"contract": "eosio.token", "symbol": "EOS", "opts": ["EOS_INCLUDE_STAKED"], "limit": 10}`),
			},
		},
		{
			Label:    "Get Account Balances (Alpha)",
			Document: graphqlDocument(box, "get_account_balances.graphql"),
			Variables: static.GraphqlVariablesByNetwork{
				"generic": r(`{"account": "eosio.token", "opts": ["EOS_INCLUDE_STAKED"], "limit": 10}`),
			},
		},
	}
}

func graphqlDocument(box *rice.Box, name string) static.GraphqlDocument {
	asset, err := box.String(name)
	if err != nil {
		panic(fmt.Errorf("unable to get content for graphql examples file %q: %w", name, err))
	}

	return static.GraphqlDocument(asset)
}

func oneWeekAgo() string {
	return time.Now().Add(-7 * 24 * time.Hour).UTC().Format("2006-01-02T15:04:05Z")
}

func dateOffsetByBlock(blockCount int) string {
	return time.Now().Add(time.Duration(blockCount) * 500 * time.Millisecond).UTC().Format("2006-01-02T15:04:05Z")
}

func r(rawJSON string) json.RawMessage {
	return json.RawMessage(rawJSON)
}
