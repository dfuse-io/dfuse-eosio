package dgraphql

import (
	"github.com/GeertJohan/go.rice/embedded"
	"time"
)

func init() {

	// define files
	file2 := &embedded.EmbeddedFile{
		Filename:    "get_account_balances.graphql",
		FileModTime: time.Unix(1601300274, 0),

		Content: string("query($account: String!, $limit: Uint32, $opts: [ACCOUNT_BALANCE_OPTION!]) {\n  accountBalances(account: $account,limit: $limit, options: $opts) {\n    blockRef {\n      id\n      number\n    }\n    pageInfo {\n      startCursor\n      endCursor\n    }\n    edges {\n      node {\n        account\n        contract\n        symbol\n        precision\n        balance\n      }\n    }\n  }\n}"),
	}
	file3 := &embedded.EmbeddedFile{
		Filename:    "get_account_history_by_account.graphql",
		FileModTime: time.Unix(1601300274, 0),

		Content: string("query($account: String!, $cursor: String, $limit: Int64) {\n  getAccountHistoryActions(account: $account,cursor: $cursor, limit: $limit ) {\n    edges {\n      cursor\n      node {\n        blockID\n        blockNum\n        blockTime\n        receiver\n        account\n        name\n        authorization {\n          actor\n        }\n      }\n    }\n    pageInfo {\n      startCursor\n      hasNextPage\n      hasPreviousPage\n    }\n  }\n}"),
	}
	file4 := &embedded.EmbeddedFile{
		Filename:    "get_account_history_by_account_contract.graphql",
		FileModTime: time.Unix(1601300274, 0),

		Content: string("query($account: String!,$contract: String, $cursor: String, $limit: Int64) {\n  getAccountHistoryActions(account: $account,contract: $contract,cursor: $cursor, limit: $limit ) {\n    edges {\n      cursor\n      node {\n        blockID\n        blockNum\n        blockTime\n        receiver\n        account\n        name\n        authorization {\n          actor\n        }\n      }\n    }\n    pageInfo {\n      startCursor\n      hasNextPage\n      hasPreviousPage\n    }\n  }\n}"),
	}
	file5 := &embedded.EmbeddedFile{
		Filename:    "get_block_by_id.graphql",
		FileModTime: time.Unix(1601300274, 0),

		Content: string("query ($blockId: String!) {\n  block(id: $blockId) {\n    id\n    num\n    dposLIBNum\n    executedTransactionCount\n    irreversible\n    header {\n      id\n      num\n      timestamp\n      producer\n      previous\n    }\n    transactionTraces(first: 5) {\n      pageInfo {\n        startCursor\n        endCursor\n      }\n      edges {\n        cursor\n        node {\n          id\n          status\n          topLevelActions {\n            account\n            name\n            receiver\n            json\n          }\n        }\n      }\n    }\n  }\n}\n"),
	}
	file6 := &embedded.EmbeddedFile{
		Filename:    "get_block_by_num.graphql",
		FileModTime: time.Unix(1601300274, 0),

		Content: string("query ($blockNum: Uint32) {\n  block(num: $blockNum) {\n    id\n    num\n    dposLIBNum\n    executedTransactionCount\n    irreversible\n    header {\n      id\n      num\n      timestamp\n      producer\n      previous\n    }\n    transactionTraces(first: 5) {\n      pageInfo {\n        startCursor\n        endCursor\n      }\n      edges {\n        cursor\n        node {\n          id\n          status\n          topLevelActions {\n            account\n            name\n            receiver\n            json\n          }\n        }\n      }\n    }\n  }\n}\n"),
	}
	file7 := &embedded.EmbeddedFile{
		Filename:    "get_token_balances.graphql",
		FileModTime: time.Unix(1601300274, 0),

		Content: string("query($contract: String!, $symbol:String!, $limit: Uint32, $opts: [ACCOUNT_BALANCE_OPTION!]) {\n  tokenBalances(contract: $contract, symbol: $symbol,limit: $limit, options: $opts) {\n    blockRef {\n      id\n      number\n    }\n    pageInfo {\n      startCursor\n      endCursor\n    }\n    edges {\n      node {\n        account\n        contract\n        symbol\n        precision\n        balance\n      }\n    }\n  }\n}"),
	}
	file8 := &embedded.EmbeddedFile{
		Filename:    "get_tokens.graphql",
		FileModTime: time.Unix(1601300274, 0),

		Content: string("query {\n  tokens {\n    blockRef {\n      id\n      number\n    }\n    pageInfo {\n      startCursor\n      endCursor\n    }\n    edges {\n      cursor\n      node {\n        symbol\n        contract\n        holders\n        totalSupply\n\n      }\n    }\n  }\n}"),
	}
	file9 := &embedded.EmbeddedFile{
		Filename:    "search_query_backward.graphql",
		FileModTime: time.Unix(1601300274, 0),

		Content: string("query ($query: String!, $cursor: String, $limit: Int64, $low: Int64, $high: Int64) {\n  searchTransactionsBackward(query: $query, lowBlockNum: $low, highBlockNum: $high, limit: $limit, cursor: $cursor) {\n    results {\n      cursor\n      trace {\n        block {\n          num\n          id\n          confirmed\n          timestamp\n          previous\n        }\n        id\n        matchingActions {\n          account\n          name\n          json\n          seq\n          receiver\n        }\n      }\n    }\n  }\n}\n"),
	}
	filea := &embedded.EmbeddedFile{
		Filename:    "search_query_forward.graphql",
		FileModTime: time.Unix(1601300274, 0),

		Content: string("query ($query: String!, $cursor: String, $limit: Int64, $low: Int64, $high: Int64) {\n  searchTransactionsForward(query: $query, lowBlockNum: $low, highBlockNum: $high, limit: $limit, cursor: $cursor) {\n    results {\n      undo\n      cursor\n      trace {\n        block {\n          num\n          id\n          confirmed\n          timestamp\n          previous\n        }\n        id\n        matchingActions {\n          account\n          name\n          json\n          seq\n          receiver\n        }\n      }\n    }\n  }\n}\n"),
	}
	fileb := &embedded.EmbeddedFile{
		Filename:    "search_stream_backward.graphql",
		FileModTime: time.Unix(1601300274, 0),

		Content: string("subscription ($query: String!, $cursor: String, $limit: Int64, $low: Int64) {\n  searchTransactionsBackward(query: $query, lowBlockNum: $low, limit: $limit, cursor: $cursor) {\n    cursor\n    trace {\n      block {\n        num\n        id\n        confirmed\n        timestamp\n        previous\n      }\n      id\n      matchingActions {\n        account\n        name\n        json\n        seq\n        receiver\n      }\n    }\n  }\n}\n"),
	}
	filec := &embedded.EmbeddedFile{
		Filename:    "search_stream_forward.graphql",
		FileModTime: time.Unix(1601300274, 0),

		Content: string("subscription ($query: String!, $cursor: String, $limit: Int64) {\n  searchTransactionsForward(query: $query, limit: $limit, cursor: $cursor) {\n    undo\n    cursor\n    trace {\n      block {\n        num\n        id\n        confirmed\n        timestamp\n        previous\n      }\n      id\n      matchingActions {\n        account\n        name\n        json\n        seq\n        receiver\n      }\n    }\n  }\n}\n"),
	}
	filed := &embedded.EmbeddedFile{
		Filename:    "time_ranges.graphql",
		FileModTime: time.Unix(1601300274, 0),

		Content: string("query ($start: Time!, $end: Time!) {\n  low: blockIDByTime(time: $start) {\n    num\n    id\n  }\n  high: blockIDByTime(time: $end) {\n    num\n    id\n  }\n}\n"),
	}

	// define dirs
	dir1 := &embedded.EmbeddedDir{
		Filename:   "",
		DirModTime: time.Unix(1601300274, 0),
		ChildFiles: []*embedded.EmbeddedFile{
			file2, // get_account_balances.graphql
			file3, // get_account_history_by_account.graphql
			file4, // get_account_history_by_account_contract.graphql
			file5, // get_block_by_id.graphql
			file6, // get_block_by_num.graphql
			file7, // get_token_balances.graphql
			file8, // get_tokens.graphql
			file9, // search_query_backward.graphql
			filea, // search_query_forward.graphql
			fileb, // search_stream_backward.graphql
			filec, // search_stream_forward.graphql
			filed, // time_ranges.graphql

		},
	}

	// link ChildDirs
	dir1.ChildDirs = []*embedded.EmbeddedDir{}

	// register embeddedBox
	embedded.RegisterEmbeddedBox(`graphql-examples`, &embedded.EmbeddedBox{
		Name: `graphql-examples`,
		Time: time.Unix(1601300274, 0),
		Dirs: map[string]*embedded.EmbeddedDir{
			"": dir1,
		},
		Files: map[string]*embedded.EmbeddedFile{
			"get_account_balances.graphql":                    file2,
			"get_account_history_by_account.graphql":          file3,
			"get_account_history_by_account_contract.graphql": file4,
			"get_block_by_id.graphql":                         file5,
			"get_block_by_num.graphql":                        file6,
			"get_token_balances.graphql":                      file7,
			"get_tokens.graphql":                              file8,
			"search_query_backward.graphql":                   file9,
			"search_query_forward.graphql":                    filea,
			"search_stream_backward.graphql":                  fileb,
			"search_stream_forward.graphql":                   filec,
			"time_ranges.graphql":                             filed,
		},
	})
}
