import { searchStore } from "../stores"
import { SearchTransactionsResponse } from "@dfuse/client"
// eslint-disable-next-line import/no-extraneous-dependencies
import { getDfuseClient } from "@dfuse/explore"

export function performStructuredSearch(cursor: string) {
  if (!searchStore.loadingTransactions) {
    if (searchStore.didRangeFilterChange()) {
      searchStore.cursorCache.resetAll()
      cursor = ""
    }

    searchStore.saveBlockRange()
    searchStore.loadingTransactions = true

    const { q, ...rest } = searchStore.toParams(cursor)

    return getDfuseClient()
      .searchTransactions(q, rest)
      .then((response: SearchTransactionsResponse) => {
        searchStore.loadingTransactions = false
        searchStore.results = response.transactions || []
        searchStore.updateCursorCache(response.cursor)
        searchStore.searchError = undefined
      })
      .catch((error: any) => {
        searchStore.loadingTransactions = false
        searchStore.searchError = error
        searchStore.results = []
      })
  }

  return cursor
}
