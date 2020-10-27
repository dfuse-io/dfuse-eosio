import { t } from "i18next"
import { observer } from "mobx-react"
import * as React from "react"
import { DataEmpty } from "@dfuse/explorer"
import { TransactionLifecycle } from "@dfuse/client"
import { RouteComponentProps } from "react-router"
import {
  ListTransactions,
  TransactionListInfo
} from "../../components/list-transactions/list-transactions.component"
import { Panel } from "../../atoms/panel/panel.component"
import { fetchTransactionList } from "../../services/transaction"
import { Cell, Grid } from "../../atoms/ui-grid/ui-grid.component"
import { transactionLifecyclesToTransactionInfo } from "../../helpers/legacy.helpers"
import { ListContentLoaderComponent } from "../../components/list-content-loader/list-content-loader.component"
import { isTransactionResponseEmpty } from "../../helpers/transaction.helpers"
import { ListTransactionsResponse } from "../../clients/websocket/eosws"
import { transactionListStore } from "../../stores"

interface Props extends RouteComponentProps<any> {}

@observer
export class PagedTransactions extends ListContentLoaderComponent<Props, any> {
  constructor(props: Props) {
    super(props)
    this.cursorCache = transactionListStore.cursorCache
  }
  fetchListForCursor(cursor: string) {
    fetchTransactionList(cursor, this.PER_PAGE)
  }

  cursoredUrl = (cursor: string) => {
    return `?cursor=${encodeURIComponent(cursor)}`
  }

  renderEmpty() {
    return <DataEmpty text={t("transaction.list.empty")} />
  }

  prepareRenderContent = (response: ListTransactionsResponse): React.ReactNode => {
    if (isTransactionResponseEmpty(response)) {
      return this.renderEmpty()
    }
    return this.renderContent(response.transactions)
  }

  renderContent = (transactions: TransactionLifecycle[]): React.ReactNode => {
    const transactionInfos: TransactionListInfo[] = transactionLifecyclesToTransactionInfo(
      transactions
    )
    return (
      <Cell>
        <Cell overflowX="auto">
          <ListTransactions collapseAll={true} transactionInfos={transactionInfos} />
        </Cell>
        <Grid gridTemplateColumns={["1fr"]}>
          <Cell justifySelf="right" alignSelf="right" p={[4]}>
            {this.renderNavigation("light", transactionListStore.hasNextPage)}
          </Cell>
        </Grid>
      </Cell>
    )
  }

  render() {
    return (
      <Panel
        title={t("transaction.list.title")}
        renderSideTitle={() => this.renderNavigation("light", transactionListStore.hasNextPage)}
      >
        {this.handleRender(fetchTransactionList, t("transaction.list.loading"))}
      </Panel>
    )
  }
}
