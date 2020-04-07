import * as React from "react"
import { task } from "mobx-task"
import { fetchBlockTransactions } from "../../services/block"
import { RouteComponentProps } from "react-router-dom"
import { t } from "i18next"
import { observer } from "mobx-react"
import { Panel } from "../../atoms/panel/panel.component"
import { styled } from "../../theme"
import { space } from "styled-system"
import { Cell, Grid } from "../../atoms/ui-grid/ui-grid.component"
import {
  ListTransactions,
  TransactionListInfo
} from "../../components/list-transactions/list-transactions.component"
import { transactionLifecyclesToTransactionInfo } from "../../helpers/legacy.helpers"
import { Links } from "../../routes"
import { ListContentLoaderComponent } from "../../components/list-content-loader/list-content-loader.component"
import { TransactionLifecycle } from "@dfuse/client"
import { isTransactionResponseEmpty } from "../../helpers/transaction.helpers"
import { ListTransactionsResponse } from "../../clients/websocket/eosws"

const BlockTransactionContainer: React.ComponentType<any> = styled.div`
  ${space};
`

interface Props
  extends RouteComponentProps<{
    id: string
  }> {}

@observer
export class BlockTransactions extends ListContentLoaderComponent<Props> {
  PER_PAGE = 30
  fetcher = task(fetchBlockTransactions, { swallow: true })

  cursoredUrl = (cursor: string) => {
    return `${Links.viewBlock({ id: this.props.match.params.id })}?cursor=${encodeURIComponent(
      cursor
    )}`
  }

  componentDidUpdate(prevProps: Props) {
    if (prevProps.match.params.id !== this.props.match.params.id) {
      this.componentDidMountHandler()
    }
  }

  fetchListForCursor(cursor: string) {
    this.fetcher(this.props.match.params.id, cursor || "", this.PER_PAGE)
  }

  prepareRenderContent = (response: ListTransactionsResponse): React.ReactNode => {
    if (isTransactionResponseEmpty(response)) {
      return this.renderEmpty()
    }

    this.cursorCache.prepareNextCursor(response.cursor)

    return this.renderContent(response.transactions)
  }

  renderContent = (transactions: TransactionLifecycle[]): React.ReactNode => {
    const transactionInfos: TransactionListInfo[] = transactionLifecyclesToTransactionInfo(
      transactions
    )
    const showNext = this.cursorCache.hasNextPage

    return (
      <Panel
        title={t("transaction.list.title")}
        renderSideTitle={() => this.renderNavigation("light", showNext)}
      >
        <Cell>
          <Cell overflowX="auto">
            <ListTransactions
              displayFields={["id", "blockTime"]}
              collapseAll={true}
              transactionInfos={transactionInfos}
            />
          </Cell>
          <Grid gridTemplateColumns={["1fr"]}>
            <Cell justifySelf="right" alignSelf="right" p={[4]}>
              {this.renderNavigation("light", showNext)}
            </Cell>
          </Grid>
        </Cell>
      </Panel>
    )
  }

  render() {
    return (
      <BlockTransactionContainer mt={[0]}>
        {this.handleRender(this.fetcher, t("transaction.list.loading"))}
      </BlockTransactionContainer>
    )
  }
}
