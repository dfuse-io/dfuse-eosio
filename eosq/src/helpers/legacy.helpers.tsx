import { DeferredOperation } from "../models/transaction"
import { DTrxOp, TransactionLifecycle, SearchTransactionRow } from "@dfuse/client"
import { TransactionListInfo } from "../components/list-transactions/list-transactions.component"
import { TransactionLifecycleWrap } from "../services/transaction-lifecycle"

export function convertDTrxOpToDeferredOperation(id: string, dtrxOp: DTrxOp): DeferredOperation {
  return {
    transaction_id: dtrxOp.trx_id,
    action_index: dtrxOp.action_idx,
    by_transaction_id: id,
    operation: dtrxOp.op,
    sender: dtrxOp.sender,
    sender_id: dtrxOp.sender_id,
    payer: dtrxOp.payer,
    published_at: dtrxOp.published_at,
    delay_until: dtrxOp.delay_until,
    expiration_at: dtrxOp.expiration_at,
    related_transactions: [dtrxOp.trx_id, id]
  }
}

export function convertDTrxOpsToDeferredOperations(
  id: string,
  dtrxops: DTrxOp[]
): DeferredOperation[] {
  return dtrxops.map((dtrxop: DTrxOp) => {
    return convertDTrxOpToDeferredOperation(id, dtrxop)
  })
}

export function transactionSearchResultsToTransactionInfo(
  searchResults: SearchTransactionRow[]
): TransactionListInfo[] {
  return (searchResults || []).map((result: SearchTransactionRow) => {
    const lifecycleWrap = new TransactionLifecycleWrap(result.lifecycle)
    return {
      id: result.lifecycle.id,
      blockNum: lifecycleWrap.blockNum || undefined,
      blockId: lifecycleWrap.blockId || undefined,
      blockTime: lifecycleWrap.blockTimestamp || undefined,
      irreversible: result.lifecycle.execution_irreversible,
      actionTraces: lifecycleWrap.actionTraces,
      status: lifecycleWrap.status,
      actionIndexes: result.action_idx,
      dtrxops: result.lifecycle.dtrxops,
      dbops: result.lifecycle.dbops,
      ramops: result.lifecycle.ramops,
      actions: lifecycleWrap.actions,
      tableops: lifecycleWrap.lifecycle.tableops
    }
  })
}

export function transactionLifecyclesToTransactionInfo(
  searchResults: TransactionLifecycle[]
): TransactionListInfo[] {
  return (searchResults || []).map((result: TransactionLifecycle) => {
    const lifecycleWrap = new TransactionLifecycleWrap(result)
    return {
      id: lifecycleWrap.lifecycle.id,
      blockNum: lifecycleWrap.blockNum || undefined,
      blockId: lifecycleWrap.blockId || undefined,
      blockTime: lifecycleWrap.blockTimestamp || undefined,
      irreversible: lifecycleWrap.lifecycle.execution_irreversible,
      actionTraces: lifecycleWrap.actionTraces,
      status: lifecycleWrap.status,
      action_indexes: [],
      dtrxops: lifecycleWrap.lifecycle.dtrxops,
      actions: lifecycleWrap.actions,
      dbops: lifecycleWrap.lifecycle.dbops,
      ramops: lifecycleWrap.lifecycle.ramops,
      tableops: lifecycleWrap.lifecycle.tableops
    }
  })
}
