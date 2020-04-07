import { task } from "mobx-task"
import { log } from "./logger"
import { listTransactions } from "../clients/websocket/eosws"

export const fetchTransactionList = task(
  async (cursor: string, perPage: number) => {
    return getTransactions(cursor, perPage)
  },
  { swallow: true }
)

export const getTransactions = task(
  async (cursor: string, perPage: number) => {
    const transactionResponse = await listTransactions(cursor, perPage)
    if (!transactionResponse || transactionResponse.transactions.length === 0) {
      log.info("No account found for query [%s] via API.")
      return null
    }

    return transactionResponse
  },
  { swallow: true }
)
