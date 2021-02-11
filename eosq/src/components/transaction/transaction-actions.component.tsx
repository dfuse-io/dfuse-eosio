import { t } from "i18next"
import * as React from "react"
import { styled } from "../../theme"
import { DeferredOperation } from "../../models/transaction"
import { DataEmpty } from "@dfuse/explorer"
import { Cell } from "../../atoms/ui-grid/ui-grid.component"
import { ListActions } from "./list-actions.component"
import { RAMOp, ActionTrace, DbOp, CreationNode, TableOp, Action } from "@dfuse/client"
import { PageContext } from "../../models/core"
import { ListActionTraces } from "./list-action-traces.component"
import { Text } from "../../atoms/text/text.component"
import { UiSwitch } from "../../atoms/ui-switch/switch"

const ListWrapper: React.ComponentType<any> = styled(Cell)`
  border-top: 1px solid ${(props) => props.theme.colors.border};
  min-width: 900px;
  padding-left: 10px;
  padding-right: 10px;
  padding-bottom: 20px;
  padding-top: 15px;
`

const PendingCell: React.ComponentType<any> = styled(Cell)`
  min-height: 10vh;
  width: 100%;
`

interface Props {
  actionTraces?: ActionTrace<any>[]
  actions?: Action<any>[]
  deferredOperations?: DeferredOperation[]
  ramops?: RAMOp[]
  dbops?: DbOp[]
  pageContext?: PageContext
  actionIndexes?: number[]
  creationTree?: CreationNode[]
  tableops?: TableOp[]
}

interface State {
  displayCreationTrace: boolean
}

export class TransactionActions extends React.Component<Props, State> {
  state = { displayCreationTrace: false }

  noActions() {
    return (
      (this.props.actions === undefined || this.props.actions === null) &&
      (this.props.actionTraces === undefined || this.props.actionTraces === null)
    )
  }

  renderEmpty() {
    return <DataEmpty text={t("transaction.traces.empty")} />
  }

  renderActionTable(actions: Action<any>[]) {
    return (
      <Cell overflowX="auto">
        <ListWrapper display={["table", "block"]}>
          <ListActions
            ramops={this.props.ramops}
            dbops={this.props.dbops}
            actions={actions}
            deferredOperations={this.props.deferredOperations}
            actionIndexes={this.props.actionIndexes}
          />
        </ListWrapper>
      </Cell>
    )
  }
  renderActionTraceTable(actionTraces: ActionTrace<any>[]) {
    return (
      <Cell overflowX="auto">
        <ListWrapper display={["table", "block"]}>
          <ListActionTraces
            creationTree={this.props.creationTree}
            displayCreationTrace={this.state.displayCreationTrace}
            collapsed={false}
            dbops={this.props.dbops}
            ramops={this.props.ramops}
            tableops={this.props.tableops}
            actionTraces={actionTraces}
            deferredOperations={this.props.deferredOperations}
            pageContext={this.props.pageContext}
            actionIndexes={this.props.actionIndexes}
          />
        </ListWrapper>
      </Cell>
    )
  }

  onToggleActionsView = (checked: boolean) => {
    this.setState({ displayCreationTrace: checked })
  }

  renderToggle() {
    return (
      <Cell p="0 32px" textAlign="right">
        <Text display="inline-block">{t("transaction.displayedTree.executionTree")}</Text>
        <UiSwitch onChange={this.onToggleActionsView} />
        <Text display="inline-block">{t("transaction.displayedTree.creationTree")}</Text>
      </Cell>
    )
  }

  renderContent() {
    const { actions } = this.props
    const { actionTraces } = this.props
    if (this.noActions()) {
      return <PendingCell />
    }

    if ((actionTraces || []).length > 0) {
      return (
        <Cell>
          {this.renderToggle()}
          {this.renderActionTraceTable(actionTraces!)}
        </Cell>
      )
    }

    if ((actions || []).length > 0) {
      return this.renderActionTable(actions!)
    }

    return this.renderEmpty()
  }

  render() {
    return this.renderContent()
  }
}
