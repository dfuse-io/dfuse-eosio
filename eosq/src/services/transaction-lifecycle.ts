import {
  TransactionLifecycle,
  Action,
  Transaction,
  Authorization,
  DTrxOp,
  ActionTrace
} from "@dfuse/client"
import { uniq } from "ramda"
import { TransactionReceiptStatus } from "../models/transaction"

export class TransactionLifecycleWrap {
  public lifecycle: TransactionLifecycle

  constructor(transactionLifeCycle: TransactionLifecycle) {
    this.lifecycle = transactionLifeCycle
  }

  get status(): TransactionReceiptStatus {
    return this.lifecycle.transaction_status as TransactionReceiptStatus
  }

  get actions(): Action<any>[] {
    let actions: Action<any>[] = []
    if (this.transaction) {
      actions = this.transaction.actions || []
    }

    if (
      this.lifecycle.created_by &&
      this.lifecycle.created_by.op === "PUSH_CREATE" &&
      this.lifecycle.dtrxops
    ) {
      const dtrxop = this.lifecycle.dtrxops.find((dtrxOp: DTrxOp) => {
        return dtrxOp.op === "PUSH_CREATE" && dtrxOp.trx_id === this.lifecycle.id
      })
      if (dtrxop && dtrxop.trx && dtrxop.trx.actions) {
        actions = dtrxop.trx.actions
      }
    }
    return actions
  }

  get actionTraces(): ActionTrace<any>[] {
    if (this.executionTrace && this.executionTrace.action_traces) {
      return this.executionTrace.action_traces || []
    }
    return []
  }

  get hasActions() {
    return this.actions.length > 0 || this.actionTraces.length > 0
  }

  get transaction(): Transaction | undefined {
    return this.lifecycle.transaction
  }

  get executionTrace() {
    return this.lifecycle.execution_trace
  }

  get noBlockInfo(): boolean {
    return !this.blockNum || !this.blockId
  }

  get blockNum() {
    if (this.lifecycle.execution_trace && this.lifecycle.execution_trace.block_num) {
      return this.lifecycle.execution_trace.block_num
    }

    if (this.lifecycle.created_by && this.lifecycle.created_by.block_num > 0) {
      return this.lifecycle.created_by.block_num
    }

    return 0
  }

  get blockId(): string | null {
    if (this.lifecycle.execution_trace && this.lifecycle.execution_trace.producer_block_id) {
      return this.lifecycle.execution_trace.producer_block_id
    }

    if (this.lifecycle.created_by && this.lifecycle.created_by.block_id) {
      return this.lifecycle.created_by.block_id
    }

    return null
  }

  get blockTimestamp() {
    if (this.lifecycle.execution_block_header) {
      return this.lifecycle.execution_block_header.timestamp
    }

    if (this.lifecycle.created_by && this.lifecycle.created_by.block_time) {
      return this.lifecycle.created_by.block_time
    }

    return null
  }

  get authorizations() {
    const authorizationList: Authorization[] = []
    const actionTraces = (this.executionTrace ? this.executionTrace.action_traces : []) || []
    // TODO Add actions[].authorization[] from the original packed transaction once available
    actionTraces.forEach((actionTrace: ActionTrace<any>) => {
      ;(actionTrace.act.authorization || []).forEach((auth: Authorization) => {
        authorizationList.push(auth)
      })
    })

    return uniq(authorizationList)
  }

  get exceptMessage() {
    if (this.executionTrace && this.executionTrace.except) {
      let message = ""
      if (this.executionTrace.except.stack && this.executionTrace.except.stack.length > 0) {
        const firstStackItem = this.executionTrace.except.stack[0]
        message = firstStackItem.format
        Object.keys(firstStackItem.data).forEach((key: string) => {
          message = message.replace(`\${${key}}`, firstStackItem.data[key])
        })
        return message
      }

      return this.executionTrace.except.message
    }
    return ""
  }

  get totalActionCount() {
    let count = 0
    this.actionTraces.forEach((actionTrace: ActionTrace<any>) => {
      count += 1
      count += this.actionCount(actionTrace.inline_traces || [])
    })

    return count
  }

  private actionCount(actionTraces: ActionTrace<any>[]): number {
    let count = 0
    actionTraces.forEach((actionTrace: ActionTrace<any>) => {
      count += this.actionCount(actionTrace.inline_traces || [])
      count += 1
    })

    return count
  }
}
