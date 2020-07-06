import * as React from "react"
import { TraceLevel, DeferredOperation } from "../../models/transaction"
import { ActionTracePill } from "../action-pills/action-pill.component"
import { Cell, Grid } from "../../atoms/ui-grid/ui-grid.component"
import { DeferredLink } from "../deferred-link/deferred-link"
import moment from "moment"
import { secondsToTime } from "@dfuse/explorer"
import { ActionTrace, DbOp, RAMOp, CreationNode, TableOp } from "@dfuse/client"
import { PageContext } from "../../models/core"
import { TransactionTracesWrap } from "../../services/transaction-traces-wrap"

interface Props {
  displayCreationTrace?: boolean
  actionTraces: ActionTrace<any>[]
  deferredOperations?: DeferredOperation[]
  ramops?: RAMOp[]
  dbops?: DbOp[]
  pageContext?: PageContext
  actionIndexes?: number[]
  collapsed?: boolean
  creationTree?: CreationNode[]
  tableops?: TableOp[]
}

export class ListActionTraces extends React.Component<Props, any> {
  static defaultProps = {
    collapsed: false
  }

  renderChildItem = (traceLevel: TraceLevel, index: number) => {
    let operations: DeferredOperation[] = []

    if (this.props.deferredOperations && !this.props.collapsed) {
      operations = this.props.deferredOperations.filter(
        (operation) =>
          operation.action_index === traceLevel.index && operation.operation !== "PUSH_CREATE"
      )
    }

    const ramops = (this.props.ramops || []).filter(
      (ramop: RAMOp) => ramop.action_idx === traceLevel.index && ramop.op !== "deferred_trx_removed"
    )
    const dbops = (this.props.dbops || []).filter(
      (dbop: DbOp) => dbop.action_idx === traceLevel.index
    )

    const tableops = (this.props.tableops || []).filter(
      (tableop: TableOp) => tableop.action_idx === traceLevel.index
    )

    return (
      <Cell key={index}>
        <Cell pl={`${traceLevel.level * 25}px`}>
          <ActionTracePill
            highlighted={this.computeHighlighted(traceLevel)}
            dbops={dbops}
            ramops={ramops}
            tableops={tableops}
            actionTrace={traceLevel.actionTrace}
            pageContext={this.props.pageContext}
          />
        </Cell>
        <Cell pl={`${traceLevel.level * 25 + 31}px`}>
          {operations.map((operation: DeferredOperation, idx: number) => {
            const delaySec = moment.duration(
              moment(operation.delay_until).diff(operation.published_at)
            )

            return (
              <Cell key={idx} mt={[1]}>
                <DeferredLink
                  transactionId={operation.transaction_id}
                  operation={operation.operation}
                  delay={secondsToTime(delaySec.asSeconds())}
                />
              </Cell>
            )
          })}
        </Cell>
      </Cell>
    )
  }

  computeHighlighted(traceLevel: TraceLevel) {
    if (this.props.actionIndexes && this.props.actionIndexes.includes(traceLevel.index)) {
      return true
    }
    return false
  }

  renderGroupItem = (
    key: number,
    traceLevels: TraceLevel[],
    collapsedTraceLevels: TraceLevel[]
  ): JSX.Element => {
    const displayedTraceLevels = this.props.collapsed ? collapsedTraceLevels : traceLevels

    const contents = displayedTraceLevels.map((traceLevel: TraceLevel) => {
      return this.renderChildItem(traceLevel, traceLevel.index)
    })

    return <Grid key={key}>{contents}</Grid>
  }

  renderItems = () => {
    const traceWrap = new TransactionTracesWrap(
      this.props.actionTraces,
      this.props.deferredOperations,
      this.props.actionIndexes,
      this.props.creationTree,
      this.props.displayCreationTrace
    )
    // const extraDeferred = this.props.deferredOperations ? this.props.deferredOperations.length : 0

    return traceWrap.mapGroups((groupedTraces: TraceLevel[], key: string) => {
      return this.renderGroupItem(
        parseInt(key, 10),
        groupedTraces,
        traceWrap.collapsedTraces(groupedTraces)
      )
    })
  }

  render() {
    return <Cell>{this.renderItems()}</Cell>
  }
}
