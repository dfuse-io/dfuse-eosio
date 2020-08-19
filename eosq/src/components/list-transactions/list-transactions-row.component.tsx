import { t } from "i18next"
import * as React from "react"
import { compactString, formatNumber, NBSP } from "@dfuse/explorer"
import { DeferredOperation, TransactionReceiptStatus } from "../../models/transaction"
import { Links } from "../../routes"

import { observer } from "mobx-react"
import {
  UiTableCellPill,
  UiTableCellTop,
  UiTableRowAlternated
} from "../../atoms/ui-table/ui-table.component"
import { Text, TextLink } from "../../atoms/text/text.component"
import { formatDateFromString } from "../../helpers/moment.helpers"
import { Cell } from "../../atoms/ui-grid/ui-grid.component"
import { BlockInfoBox } from "../block-info-box/block-info-box.component"
import { ListActionTraces } from "../transaction/list-action-traces.component"
import { convertDTrxOpsToDeferredOperations } from "../../helpers/legacy.helpers"
import { ListActions } from "../transaction/list-actions.component"
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome"
import {
  faCaretRight,
  faCaretDown,
  faExclamationCircle,
  faClock
} from "@fortawesome/free-solid-svg-icons"
import { theme, styled } from "../../theme"
import { TransactionTracesWrap } from "../../services/transaction-traces-wrap"
import { TransactionListInfo } from "./list-transactions.component"
import { getTransactionStatusColor } from "../../helpers/transaction.helpers"
import { PageContext } from "../../models/core"

const Row: React.ComponentType<any> = styled(UiTableRowAlternated)`
  align-items: center;
  padding-top: 8px;
  padding-bottom: 8px;
`

interface Props {
  transactionInfo: TransactionListInfo
  displayFields: string[]
  initialCollapse: boolean
  pageContext?: PageContext
}

interface State {
  collapsed: boolean
}

const UiTableCellTopHover = styled(UiTableCellTop)`
  &:hover {
    cursor: pointer;
  }
`

@observer
export class ListTransactionsRow extends React.Component<Props, State> {
  static defaultProps = {
    displayFields: ["id", "blockId", "blockTime"]
  }

  state = {
    collapsed: true
  }

  renderTimeStamp(timestamp: string) {
    if (!timestamp || timestamp === "") {
      return null
    }
    return (
      <Cell color="text" title={formatDateFromString(timestamp, true)}>
        {formatDateFromString(timestamp, false)}
      </Cell>
    )
  }

  renderTransactionId(id: string, status: TransactionReceiptStatus, index: number) {
    const color = getTransactionStatusColor(status)
    let icon
    if (color === "statusBadgeClock") {
      icon = faClock
    } else if (color === "statusBadgeBan") {
      icon = faExclamationCircle
    }

    return (
      <Cell display="inline-block" key={index}>
        {color !== "statusBadgeCheck" ? (
          <Cell title={t(`transaction.status.${status}`)} display="inline-block" mr={[2]}>
            <FontAwesomeIcon color={theme.colors[color]} icon={icon as any} />
          </Cell>
        ) : null}
        <TextLink mr={[1]} key={id} fontSize={[2]} to={Links.viewTransaction({ id })}>
          {compactString(id, 12, 0)}
        </TextLink>
      </Cell>
    )
  }

  renderBlockInfo(blockTime?: string, blockNum?: number, blockId?: string, irreversible?: boolean) {
    if (blockNum && blockId && irreversible !== undefined) {
      return (
        <Cell display={["block", "inline-block"]} pt={[1, 0]} key={blockId}>
          <BlockInfoBox
            blockTime={blockTime}
            blockNum={blockNum}
            blockId={blockId}
            irreversible={irreversible}
          />
        </Cell>
      )
    }
    return null
  }

  renderPills(transactionInfo: TransactionListInfo, deferredOperations?: DeferredOperation[]) {
    const actionTraces = transactionInfo.actionTraces || []
    const { actions } = transactionInfo
    if (actionTraces.length === 0 && actions && actions.length > 0) {
      return <ListActions actions={actions} />
    }

    return [
      transactionInfo.status === TransactionReceiptStatus.EXPIRED ? (
        <Cell key="0">{t("transaction.detailPanel.statuses.expired")}</Cell>
      ) : (
        ""
      ),
      <ListActionTraces
        pageContext={{
          blockNum: transactionInfo.blockNum,
          accountName: this.props.pageContext ? this.props.pageContext.accountName : undefined
        }}
        collapsed={this.state.collapsed}
        key="1"
        deferredOperations={deferredOperations}
        actionTraces={actionTraces}
        actionIndexes={transactionInfo.actionIndexes || []}
        dbops={transactionInfo.dbops}
        ramops={transactionInfo.ramops}
        tableops={transactionInfo.tableops}
      />
    ]
  }

  renderSummaryFields(transaction: TransactionListInfo) {
    let render: any = []
    this.props.displayFields!.forEach((columnName: string, index: number) => {
      switch (columnName) {
        case "id":
          render = render.concat([
            this.renderTransactionId(transaction.id, transaction.status, index)
          ])
          break
        case "blockId":
          render = render.concat([
            // eslint-disable-next-line react/no-array-index-key
            <Text key={index} display="inline-block" fontSize={[2]} mr={[1]}>
              in
            </Text>,
            this.renderBlockInfo(
              transaction.blockTime,
              transaction.blockNum,
              transaction.blockId,
              transaction.irreversible
            )
          ])
          break
        default:
        // Do nothing
      }
    })

    return <Cell>{render}</Cell>
  }

  togglePills = () => {
    this.setState((prevState) => ({
      collapsed: !prevState.collapsed
    }))
  }

  renderMoreActionCTA() {
    const transaction = this.props.transactionInfo

    const traceWrap = new TransactionTracesWrap(
      transaction.actionTraces || [],
      convertDTrxOpsToDeferredOperations(transaction.id, transaction.dtrxops || []),
      transaction.actionIndexes
    )

    const extraDeferred = transaction.dtrxops ? transaction.dtrxops.length : 0
    const caret = this.state.collapsed ? faCaretRight : faCaretDown
    const extraActionCount = traceWrap.hiddenActionsCount()
    if (extraActionCount === 0 && extraDeferred === 0) {
      return null
    }

    return (
      <Cell>
        {this.state.collapsed && extraActionCount > 0 ? (
          <Text color={theme.colors.bleu8} display="inline-block">
            +{formatNumber(extraActionCount)}
            {extraDeferred > 0 ? "," : null}
            {NBSP}
          </Text>
        ) : null}
        {this.state.collapsed && extraDeferred > 0 ? (
          <Text color={theme.colors.statusBadgeClock} display="inline-block">
            +{formatNumber(extraDeferred)} def.
          </Text>
        ) : null}

        <Cell color={theme.colors.bleu8} ml={[2]} display="inline-block">
          <FontAwesomeIcon size="lg" icon={caret as any} />
        </Cell>
      </Cell>
    )
  }

  render() {
    const transaction = this.props.transactionInfo

    return (
      <Row key={transaction.id}>
        <UiTableCellTop fontSize={[2]}>{this.renderSummaryFields(transaction)}</UiTableCellTop>
        <UiTableCellPill fontSize={[2]}>
          {this.renderPills(
            transaction,
            convertDTrxOpsToDeferredOperations(transaction.id, transaction.dtrxops || [])
          )}
        </UiTableCellPill>
        <UiTableCellTopHover textAlign="right" fontSize={[2]} onClick={this.togglePills}>
          {this.renderMoreActionCTA()}
        </UiTableCellTopHover>
      </Row>
    )
  }
}
