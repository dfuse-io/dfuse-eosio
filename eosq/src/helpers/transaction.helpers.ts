import { TransactionReceiptStatus } from "../models/transaction"
import { StatusBadgeVariant } from "../atoms/status-badge/status-badge"
import { RAMOp } from "@dfuse/client"
import { ListTransactionsResponse } from "../clients/websocket/eosws"

export function getTransactionStatusColor(status: TransactionReceiptStatus): string {
  if (
    status === TransactionReceiptStatus.HARD_FAIL ||
    status === TransactionReceiptStatus.SOFT_FAIL ||
    status === TransactionReceiptStatus.EXPIRED ||
    status === TransactionReceiptStatus.CANCELED
  ) {
    return "statusBadgeBan"
  }

  if (status === TransactionReceiptStatus.DELAYED) {
    return "statusBadgeClock"
  }

  if (status === TransactionReceiptStatus.EXECUTED) {
    return "statusBadgeCheck"
  }

  return "text"
}

export function getStatusBadgeVariant(status: TransactionReceiptStatus): StatusBadgeVariant | null {
  if (
    status === TransactionReceiptStatus.HARD_FAIL ||
    status === TransactionReceiptStatus.SOFT_FAIL ||
    status === TransactionReceiptStatus.EXPIRED ||
    status === TransactionReceiptStatus.CANCELED
  ) {
    return StatusBadgeVariant.BAN
  }

  if (status === TransactionReceiptStatus.DELAYED) {
    return StatusBadgeVariant.CLOCK
  }

  if (status === TransactionReceiptStatus.EXECUTED) {
    return StatusBadgeVariant.CHECK
  }

  return null
}

export function summarizeRamOps(ramops: RAMOp[]): RAMOp[] {
  const ramOpsSummary: { [key: string]: RAMOp } = {}
  Object.assign([], ramops).forEach((ramop: RAMOp) => {
    const reference = ramOpsSummary[ramop.payer]
    if (!reference) {
      ramOpsSummary[ramop.payer] = { ...ramop }
      return
    }

    ramOpsSummary[ramop.payer].delta += ramop.delta
    if (ramOpsSummary[ramop.payer].action_idx < ramop.action_idx) {
      ramOpsSummary[ramop.payer].usage = ramop.usage
    }
  })

  return Object.keys(ramOpsSummary).map((key: string) => {
    return ramOpsSummary[key]
  })
}

export function isTransactionResponseEmpty(response: ListTransactionsResponse) {
  return !response || !response.transactions || response.transactions.length === 0
}
