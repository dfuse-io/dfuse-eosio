import { task } from "mobx-task"
import { BlockSummary } from "../models/block"
import { blockStore } from "../stores"
import { log } from "./logger"
import {
  getBlock as getBlockApi,
  listBlocks,
  listBlockTransactions,
  ListTransactionsResponse
} from "../clients/websocket/eosws"

export const fetchBlock = task(
  async (blockId: string) => {
    log.info("Searching for block id [%s].", blockId)
    const block = blockStore.findById(blockId)
    if (block !== undefined) {
      log.info("Found block [%s] in blocks cache.", blockId, block)
      return block
    }

    const result = await getBlock(blockId)
    if (result !== null) {
      log.info("Found block [%s] via search API.", blockId, result)
      return result as BlockSummary
    }

    log.info("Block [%s] not found anywhere.", blockId)
    return null
  },
  { swallow: true }
)

export const fetchBlockList = task(
  async (offset: number) => {
    const perPage = 100
    return getBlocks(offset, perPage)
  },
  { swallow: true }
)

export const getBlocks = task(
  async (offset: number, perPage: number) => {
    const blocks = await listBlocks(offset, perPage)
    if (!blocks || blocks.length === 0) {
      log.info("No account found for query [%s] via API.")
      return null
    }

    return blocks
  },
  { swallow: true }
)

export const getBlock = task(
  async (id: string) => {
    const block = await getBlockApi(id)
    if (!block) {
      log.info("No block found for query [%s] via API.")
      return null
    }

    return block
  },
  { swallow: true }
)

function isEmptyTransactionResponse(response?: ListTransactionsResponse): boolean {
  return !response || response.transactions == null || response.transactions.length === 0
}

export const fetchBlockTransactions = async (id: string, cursor: string, perPage: number) => {
  const transactionResponse = await listBlockTransactions(id, cursor, perPage)
  if (isEmptyTransactionResponse(transactionResponse)) {
    log.info("No transactions found for block [%s] via API.", id)
    return null
  }

  return transactionResponse
}
