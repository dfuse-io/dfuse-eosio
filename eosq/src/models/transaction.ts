import { ActionTrace, Action } from "@dfuse/client"

export enum TransactionReceiptStatus {
  EXECUTED = "executed",
  SOFT_FAIL = "soft_fail",
  HARD_FAIL = "hard_fail",
  DELAYED = "delayed",
  EXPIRED = "expired",
  PENDING = "pending",
  CANCELED = "canceled"
}

export interface DeferredOperation {
  transaction_id: string
  action_index: number
  by_transaction_id?: string
  operation: string
  sender: string
  sender_id: string
  payer: string
  published_at: string
  delay_until: string
  expiration_at: string
  related_transactions: string[]
}

export interface TraceLevel {
  index: number
  actionTrace: ActionTrace<any>
  group: number
  level: number
}

/**
 * Represents a transaction as it passes through the P2P channel, not yet
 * executed since not part of a block.
 */
export interface Transaction {
  id: string
  block_num: number
  block_id: string
  producer: string
  ref_block_num: number
  ref_block_prefix: number
  max_net_usage_words: number
  delay_sec: number
  expiration: string
  max_cpu_usage_ms: number
  context_free_actions: Action<any>[]
  actions: Action<any>[]
  transaction_extensions: any[]
  signatures: string[]
  public_keys: string[]
  irreversible: boolean
}

/**
 * Represents a transaction action aggregate to be displayed in the list.
 * This models aggregates is an action with extended information from the
 * transaction itself (id, block_num, date)
 */
export interface TransactionAction extends Action<any> {
  action_num: string
  cfa: boolean
}

export enum RAMOpTypes {
  CREATE_TABLE = "create_table",
  DEFERRED_TRX_ADD = "deferred_trx_add",
  DEFERRED_TRX_CANCEL = "deferred_trx_cancel",
  DEFERRED_TRX_PUSHED = "deferred_trx_pushed",
  DEFERRED_TRX_REMOVED = "deferred_trx_removed",
  DELETEAUTH = "deleteauth",
  LINKAUTH = "linkauth",
  NEWACCOUNT = "newaccount",
  PRIMARY_INDEX_ADD = "primary_index_add",
  PRIMARY_INDEX_REMOVE = "primary_index_remove",
  PRIMARY_INDEX_UPDATE = "primary_index_update",
  PRIMARY_INDEX_UPDATE_ADD_NEW_PAYER = "primary_index_update_add_new_payer",
  PRIMARY_INDEX_UPDATE_REMOVE_OLD_PAYER = "primary_index_update_remove_old_payer",
  REMOVE_TABLE = "remove_table",
  SECONDARY_INDEX_ADD = "secondary_index_add",
  SECONDARY_INDEX_REMOVE = "secondary_index_remove",
  SECONDARY_INDEX_UPDATE_ADD_NEW_PAYER = "secondary_index_update_add_new_payer",
  SECONDARY_INDEX_UPDATE_REMOVE_OLD_PAYER = "secondary_index_update_remove_old_payer",
  SETABI = "setabi",
  SETCODE = "setcode",
  UNLINKAUTH = "unlinkauth",
  UPDATEAUTH_CREATE = "updateauth_create",
  UPDATEAUTH_UPDATE = "updateauth_update"
}

export function computeTransactionTrustPercentage(
  blockNum: number | undefined,
  headBlockNum: number,
  lastIrreversibleBlockNum: number
) {
  if (blockNum === undefined) return 0.0
  if (lastIrreversibleBlockNum >= blockNum) return 1.0

  const blockPassedCount = headBlockNum - blockNum
  if (blockPassedCount >= 360) {
    return 0.9999
  }

  if (blockPassedCount < 4) {
    return blockPassedCount / 4
  }

  if (blockPassedCount === 4) {
    return 0.99
  }

  return (blockPassedCount / 360) * 0.01 + 0.99
}
