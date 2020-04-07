import * as React from "react"
import { t } from "i18next"
import { observer } from "mobx-react"
import { styled } from "../../../theme"
import { Cell, Grid } from "../../../atoms/ui-grid/ui-grid.component"
import { Account } from "../../../models/account"
import { ListTransactions } from "../../../components/list-transactions/list-transactions.component"
import { transactionSearchResultsToTransactionInfo } from "../../../helpers/legacy.helpers"
import { SearchTransactionRow } from "@dfuse/client"
import { Links } from "../../../routes"
import { BorderLessPanel } from "../../../atoms/panel/panel.component"
import { ListContentLoaderComponent } from "../../../components/list-content-loader/list-content-loader.component"
import { RouteComponentProps } from "react-router"
import { searchStore } from "../../../stores"
import { performStructuredSearch } from "../../../services/search"
import { RangeOptions } from "../../../models/search-filters"

const PanelContentWrapper: React.ComponentType<any> = styled(Cell)`
  width: 100%;
`

interface Props extends RouteComponentProps<any> {
  account: Account
}

export interface TransactionSearchResultResponse {
  cursor: string
  transactions: SearchTransactionRow[]
}

@observer
export class AccountTransactions extends ListContentLoaderComponent<Props> {
  constructor(props: Props) {
    super(props)
    this.cursorCache = searchStore.cursorCache
  }

  componentDidMount() {
    this.componentDidMountHandler()
  }

  resetInternalState() {
    this.cursorCache.resetAll()
    this.setState({
      loadingTransactions: false
    })
  }

  componentDidUpdate(prevProps: Props) {
    if (prevProps.account.account_name !== this.props.account.account_name) {
      this.resetInternalState()
      this.fetchListForCursor("")
    }
  }

  cursoredUrl = (cursor: string) => {
    let url = Links.viewAccountTabs({
      id: this.props.account.account_name,
      currentTab: "transactions"
    })

    if (cursor.length > 0) {
      url = `${url}?cursor=${encodeURIComponent(cursor)}`
    }

    return url
  }

  fetchListForCursor(cursor: string) {
    searchStore.rangeOption = RangeOptions.ALL
    searchStore.query = `(auth:${this.props.account.account_name} OR receiver:${this.props.account.account_name})`

    this.search(cursor)
  }

  search = (cursor?: string) => {
    if (!cursor || cursor.length === 0) {
      cursor = this.cursorCache.currentCursor
    }
    searchStore.limit = 5
    performStructuredSearch(cursor || "")
  }

  renderNavigationContainer() {
    return (
      <Grid gridTemplateColumns={["1fr"]}>
        <Cell justifySelf="right" alignSelf="right" px={[3]} py={[2]}>
          {this.renderNavigation("light", searchStore.hasNextPage)}
        </Cell>
      </Grid>
    )
  }

  renderSearchResults() {
    const transactionInfos = transactionSearchResultsToTransactionInfo(searchStore.results || [])
    return (
      <Cell>
        {this.renderNavigationContainer()}
        <Cell overflowX="auto">
          <ListTransactions
            transactionInfos={transactionInfos}
            pageContext={{ accountName: this.props.account.account_name }}
          />
        </Cell>
        {this.renderNavigationContainer()}
      </Cell>
    )
  }

  render() {
    let content

    if (searchStore.loadingTransactions) {
      content = this.renderLoading(t("transaction.list.loading"))
    } else if (searchStore.results.length > 0) {
      content = this.renderSearchResults()
    } else {
      content = (
        <Cell p={[3]} px={[6]}>
          {this.renderError({ name: "not_found", message: "Nothing found" })}
        </Cell>
      )
    }

    return (
      <Cell>
        <BorderLessPanel>
          <PanelContentWrapper>{content}</PanelContentWrapper>
        </BorderLessPanel>
      </Cell>
    )
  }
}
