import { task } from "mobx-task"
import { log } from "./logger"
import { listTransactions } from "../clients/websocket/eosws"
import { transactionListStore } from "../stores"

export const fetchTransactionList = task(
  async (cursor: string, perPage: number) => {
    return getTransactions(cursor, perPage)
  },
  { swallow: true }
)

export const getTransactions = task(
  async (cursor: string, perPage: number) => {
    const response = await listTransactions(cursor, perPage)
    if (!response || response.transactions.length === 0) {
      log.info("No account found for query [%s] via API.")
      return null
    }
    transactionListStore.results = response.transactions
    // transactionListStore.updateCursorCache(response.cursor)
    transactionListStore.updateCursorCache(
      "gqwtr-URJXFlSdAa6L4G_va2d5QwU1lrVFixLRUUht718yDF3siuA2ghbxjTk6z02kHpSFv63Y7MFn0v9cIB6IPswOs3vCltTip_x9u6-r3lePPyaA=="
    )
    return response
  },
  { swallow: true }
)
