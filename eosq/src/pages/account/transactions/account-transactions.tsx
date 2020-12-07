import * as React from "react"
import { t } from "i18next"
import { observer } from "mobx-react"
import { Button } from "@material-ui/core"
import { theme, styled } from "../../../theme"
import { Cell, Grid } from "../../../atoms/ui-grid/ui-grid.component"
import { Text } from "../../../atoms/text/text.component"
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
import { FilterTypes, RangeOptions } from "../../../models/search-filters"
import { BLOCK_NUM_5M } from "../../../models/block"

const StyledButton: React.ComponentType<any> = styled(Button)`
  padding: 12px 30px !important;
  background-color: ${(props) => props.theme.colors.ternary} !important;
  border: none !important;
  font-weight: bold !important;
  border-radius: 0px !important;
  min-height: 35px !important;
  color: ${(props) => props.theme.colors.primary} !important;
`

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
  extendSearch() {
    const lastBlocks = searchStore.parseField(
      "lastBlocks",
      searchStore.blockRange.lastBlocks!
    ) as number
    searchStore.updateFilter(FilterTypes.BLOCK_RANGE, "lastBlocks", lastBlocks + BLOCK_NUM_5M)
    performStructuredSearch(this.cursorCache.currentCursor || "")
    this.props.history.push(this.cursoredUrl(this.cursorCache.currentCursor || ""))
  }

  renderNoResultsExtendSearchBox() {
    if (searchStore.blockRange.option === RangeOptions.LAST_BLOCKS) {
      return (
          <Cell textAlign="center" p={[3]}>
            <Text color={theme.colors.green5} fontSize={[5]} mb="20px">
              {t("transaction.list.noResultsExtend", {
                lastBlocks: searchStore.blockRange.lastBlocks
              })}
            </Text>
            <br />
            <StyledButton onClick={() => this.extendSearch()}>
              {t("transaction.list.extendSearch")}
            </StyledButton>
          </Cell>
      )
    }
    return null
  }
  render() {
    let content

    if (searchStore.loadingTransactions) {
      content = this.renderLoading(t("transaction.list.loading"))
    } else if (searchStore.results.length > 0) {
      content = this.renderSearchResults()
    } else {
      content = (
        <Cell p={[4]}>
          {this.renderNoResultsExtendSearchBox()}
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
