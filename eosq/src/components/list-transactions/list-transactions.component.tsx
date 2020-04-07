import { t } from "i18next"
import * as React from "react"
import { formatTransactionID } from "../../helpers/formatters"
import { TransactionReceiptStatus } from "../../models/transaction"
import { Links } from "../../routes"

import { observer } from "mobx-react"
import {
  UiTable,
  UiTableBody,
  UiTableCell,
  UiTableHead,
  UiTableRow
} from "../../atoms/ui-table/ui-table.component"
import { TextLink } from "../../atoms/text/text.component"
import { formatDateFromString } from "../../helpers/moment.helpers"
import { Cell } from "../../atoms/ui-grid/ui-grid.component"
import { BlockInfoBox } from "../block-info-box/block-info-box.component"
import { PageContext } from "../../models/core"
import { ActionTrace, Action, DTrxOp, DbOp, RAMOp, TableOp } from "@dfuse/client"
import { ListTransactionsRow } from "./list-transactions-row.component"

export interface TransactionListInfo {
  id: string
  actionIndexes?: number[]
  blockId?: string
  blockNum?: number
  irreversible: boolean
  blockTime?: string
  status: TransactionReceiptStatus
  actionTraces?: ActionTrace<any>[]
  actions?: Action<any>[]
  expandAll?: boolean
  dtrxops?: DTrxOp[]
  dbops?: DbOp[]
  ramops?: RAMOp[]
  tableops?: TableOp[]
}

interface Props {
  transactionInfos: TransactionListInfo[]
  displayFields?: string[]
  pageContext?: PageContext
  collapseAll?: boolean
}

@observer
export class ListTransactions extends React.Component<Props> {
  static defaultProps = {
    displayFields: ["id", "blockId", "blockTime"],
    collapseAll: true
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

  renderTransactionId(id: string) {
    return (
      <TextLink mr={[1]} key={id} fontSize={[2]} to={Links.viewTransaction({ id })}>
        {formatTransactionID(id)}
      </TextLink>
    )
  }

  renderBlockInfo(blockTime?: string, blockNum?: number, blockId?: string, irreversible?: boolean) {
    if (blockNum && blockId && irreversible !== undefined) {
      return (
        <Cell display="inline-block" key={blockId}>
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

  renderItem = (transaction: TransactionListInfo) => {
    return (
      <ListTransactionsRow
        key={transaction.id}
        transactionInfo={transaction}
        displayFields={this.props.displayFields!}
        initialCollapse={true}
        pageContext={this.props.pageContext}
      />
    )
  }

  renderItems = () => {
    if (!this.props.transactionInfos) {
      return []
    }

    return (
      <UiTableBody>
        {this.props.transactionInfos.map((transaction: TransactionListInfo) =>
          this.renderItem(transaction)
        )}
      </UiTableBody>
    )
  }

  renderHeader = () => {
    return (
      <UiTableHead>
        <UiTableRow>
          <UiTableCell fontSize={[2]}>
            {this.props.displayFields!.includes("blockId")
              ? t(`transaction.list.header.summary`)
              : t(`transaction.list.header.id`)}
          </UiTableCell>
          <UiTableCell fontSize={[2]}>{t(`transaction.list.header.action`)}</UiTableCell>
          <UiTableCell textAlign="right" fontSize={[2]}>
            {t(`transaction.list.header.moreActions`)}
          </UiTableCell>
        </UiTableRow>
      </UiTableHead>
    )
  }

  render() {
    return (
      <UiTable>
        {this.renderHeader()}
        {this.renderItems()}
      </UiTable>
    )
  }
}
