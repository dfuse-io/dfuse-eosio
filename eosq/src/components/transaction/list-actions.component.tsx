import * as React from "react"
import { DeferredOperation } from "../../models/transaction"
import { ActionPill } from "../action-pills/action-pill.component"
import { Cell } from "../../atoms/ui-grid/ui-grid.component"
import { DeferredLink } from "../deferred-link/deferred-link"
import { DbOp, RAMOp, Action } from "@dfuse/client"

interface Props {
  actions: Action<any>[]
  deferredOperations?: DeferredOperation[]
  ramops?: RAMOp[]
  dbops?: DbOp[]
  actionIndexes?: number[]
}

export class ListActions extends React.Component<Props> {
  renderItem = (action: Action<any>, index: number) => {
    let operations: DeferredOperation[] = []

    if (this.props.deferredOperations) {
      operations = this.props.deferredOperations.filter(
        (operation) => operation.action_index === index && operation.operation !== "PUSH_CREATE"
      )
    }

    const ramops = (this.props.ramops || []).filter(
      (ramop: RAMOp) => ramop.action_idx === index && ramop.op !== "deferred_trx_removed"
    )
    const dbops = (this.props.dbops || []).filter((dbop: DbOp) => dbop.action_idx === index)

    return (
      <Cell key={index}>
        <Cell>
          <ActionPill ramops={ramops} action={action} dbops={dbops} />
        </Cell>
        <Cell pl="25px">
          {operations.map((operation: DeferredOperation, idx: number) => {
            return (
              <Cell key={idx} mt={[1]}>
                <DeferredLink
                  transactionId={operation.transaction_id}
                  operation={operation.operation}
                  delay={operation.delay_until}
                />
              </Cell>
            )
          })}
        </Cell>
      </Cell>
    )
  }

  renderItems = () => {
    return this.props.actions.map((action: Action<any>, index: number) => {
      return this.renderItem(action, index)
    })
  }

  render() {
    return <Cell mb={[2]}>{this.renderItems()}</Cell>
  }
}
