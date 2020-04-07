export default {
  search: {
    placeholder: "Search for accounts, blocks, transactions, timestamps, ...",
    result: {
      noResultFoundFor: "No result found for",
      blockFound: "Block found but not handled yet, come back soon!",
      error: "An error occurred on the server while performing the search",
      nothingFound: "Nothing found",
      searchQuery: "Search query",
      unregisteredLabel: "Unregistered account",
      unregisteredValue: "You might be able to claim in the future",
      errors: {
        label: "Error:",
        request_validation_error: "your query was malformed",
        generic_error: "The search failed"
      }
    },
    suggestions: {
      summary: {
        account_history: "Account history for {{accountName}}",
        signed_by: "Transactions Signed by {{accountName}}",
        eos_token_transfer: "Token transfers from {{accountName}}",
        fuzzy_token_search: "Fuzzy token search for {{accountName}}"
      }
    },
    syntax: "SYNTAX:",
    irreversibleOnly: "Only includes irreversible blocks",
    searchResultsFor: "Search Results",
    sqeDocumentation: "SQE LANGUAGE",
    loading: "Loading...",
    errorFetch: "No suggestions for this search"
  },
  transactionSearch: {
    buttons: {
      signedBy: "SIGNED BY {{accountName}}",
      notifications: "NOTIFICATIONS"
    },
    search: "SEARCH",
    title: "Search Transactions",
    results: {
      title: "Results for",
      subTitle: "Only includes irreversible blocks"
    },
    buttonLabels: {
      account: "ACCOUNT",
      tokens: "TOKENS",
      system: "SYSTEM ACTIONS"
    },
    dropdowns: {
      tokens: {
        allTokens: "ALL TOKENS",
        eos: "EOS",
        popularTokens: "POPULAR TOKENS"
      },
      system: {
        claimRewards: "CLAIM REWARDS",
        delegateBandwidth: "DELEGATE BANDWIDTH",
        undelegateBandwidth: "UNDELEGATE BANDWIDTH",
        regProducer: "REG PRODUCER",
        setCode: "SET CODE"
      }
    }
  }
}
