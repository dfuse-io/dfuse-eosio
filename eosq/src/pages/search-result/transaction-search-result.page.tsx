import * as React from "react"
import { observer } from "mobx-react"
import { RouteComponentProps } from "react-router-dom"
import { Box, JsonWrapper } from "@dfuse/explorer"
import { ErrorData } from "@dfuse/client"
import { Button } from "@material-ui/core"
import { Panel } from "../../atoms/panel/panel.component"
import { Text } from "../../atoms/text/text.component"
import { fontSize } from "styled-system"
import { Cell, Grid } from "../../atoms/ui-grid/ui-grid.component"
import { PageContainer } from "../../components/page-container/page-container"
import { transactionSearchResultsToTransactionInfo } from "../../helpers/legacy.helpers"
import { ListTransactions } from "../../components/list-transactions/list-transactions.component"
import { t } from "i18next"
import { ListContentLoaderComponent } from "../../components/list-content-loader/list-content-loader.component"
import { formatNumber, NBSP } from "../../helpers/formatters"
import { searchStore } from "../../stores"
import { performStructuredSearch } from "../../services/search"
import { SearchQueryParams, LegacySearchQueryParams } from "../../models/search"
import { FormattedError } from "../../components/formatted-error/formatted-error"
import { FilterModal } from "./filter-modal"
import { FilterTypes, RangeOptions } from "../../models/search-filters"
import { theme, styled } from "../../theme"
import { BLOCK_NUM_5M } from "../../models/block"

interface Props extends RouteComponentProps<any> {}

const BoldText: React.ComponentType<any> = styled.span`
  font-weight: bold;
  ${fontSize};
  font-family: "Roboto Condensed", sans-serif;
`

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
  min-height: 700px;
`

const ResultInfoContainer: React.ComponentType<any> = styled(Box)`
  border: 1px solid #25c2ab;
  background-color: rgba(37, 194, 171, 0.12);
  padding: 40px;
  margin: 80px auto;
`

interface State {
  filtersOpened: boolean
}

@observer
export class TransactionSearchResultPage extends ListContentLoaderComponent<Props, State> {
  lastQuery = ""

  constructor(props: Props) {
    super(props)
    this.cursorCache = searchStore.cursorCache
  }

  componentDidMount(): void {
    window.scrollTo(0, 0)
    this.parseUrlParams()
    this.componentDidMountHandler()
  }

  componentDidUpdate(): void {
    if (this.parseQuery() !== this.lastQuery) {
      window.scrollTo(0, 0)
      this.parseUrlParams()
      this.componentDidMountHandler()
    }
  }

  parseQuery() {
    if (this.parsed.q && this.parsed.q.length > 0) {
      return decodeURIComponent(this.parsed.q)
    }

    return ""
  }

  parseUrlParams() {
    searchStore.updateFromUrlParams(this.parsed as SearchQueryParams & LegacySearchQueryParams)
  }

  cursoredUrl = (cursor: string) => {
    return searchStore.cursoredUrl(cursor)
  }

  fetchListForCursor(cursor: string) {
    if (searchStore.query.length > 0) {
      this.lastQuery = searchStore.query
      this.search(searchStore.query, cursor)
    }
  }

  renderResultsTitle() {
    return (
      <Text color="header" fontSize={[5]}>
        {t("search.searchResultsFor")}{" "}
      </Text>
    )
  }

  search = (query: string, cursor?: string) => {
    if (!cursor || cursor.length === 0) {
      cursor = this.cursorCache.currentCursor
    }

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

  renderSearchResults(showMore: boolean) {
    const transactionInfos = transactionSearchResultsToTransactionInfo(searchStore.results || [])
    return (
      <Cell>
        {this.renderNavigationContainer()}
        <Cell overflowX="auto">
          <ListTransactions transactionInfos={transactionInfos} />
        </Cell>
        <Cell px={[4]}>{showMore ? this.renderExtendSearchBox() : null}</Cell>
        {this.renderNavigationContainer()}
      </Cell>
    )
  }

  renderSideTitle = (): JSX.Element => {
    const title = t(`filters.currentFilter.${searchStore.filterCombination}`, {
      min: formatNumber(searchStore.blockRange.min || 0),
      max: formatNumber(searchStore.blockRange.max || 0),
      lastBlocks: formatNumber(searchStore.blockRange.lastBlocks || 0)
    })
    return <FilterModal {...this.props} title={title} />
  }

  renderSearchError(error: ErrorData) {
    const i18nkey = error.code === "request_validation_error" ? error.code : "generic_error"

    return (
      <Grid p={[4]} alignItems="center">
        <Cell pb={[2]} width="100%">
          <BoldText display="inline-block" fontSize={[4]}>
            {t("search.result.errors.label")}
          </BoldText>
          {NBSP}
          <Text display="inline-block" fontSize={[4]}>
            {t(`search.result.errors.${i18nkey}`)}
          </Text>
        </Cell>
        <br />
        <Cell>
          <JsonWrapper>{JSON.stringify(error, null, 2)}</JsonWrapper>
        </Cell>
      </Grid>
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

  renderExtendSearchBox() {
    if (searchStore.blockRange.option === RangeOptions.LAST_BLOCKS) {
      return (
        <ResultInfoContainer mt={[4]} mb={[4]} alignItems="center" justifyContent="center">
          <Cell textAlign="center" p={[3]}>
            <Text color={theme.colors.green5} fontSize={[5]} mb="20px">
              {t("transaction.list.noMoreResultsExtend", {
                lastBlocks: searchStore.blockRange.lastBlocks
              })}
            </Text>
            <br />
            <StyledButton onClick={() => this.extendSearch()}>
              {t("transaction.list.extendSearch")}
            </StyledButton>
            <Cell mt="20px" color={theme.colors.grey5}>
              <FilterModal
                title={t("transaction.list.advancedOptions")}
                {...this.props}
                color={theme.colors.grey7}
              />
            </Cell>
          </Cell>
        </ResultInfoContainer>
      )
    }
    return null
  }

  renderNoResultsExtendSearchBox() {
    if (searchStore.blockRange.option === RangeOptions.LAST_BLOCKS) {
      return (
        <ResultInfoContainer mt={[4]} mb={[4]} alignItems="center" justifyContent="center">
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

            <Cell mt="20px">
              <FilterModal
                title={t("transaction.list.advancedOptions")}
                {...this.props}
                color={theme.colors.grey7}
              />
            </Cell>
          </Cell>
        </ResultInfoContainer>
      )
    }
    return null
  }

  render() {
    let content = null

    if (searchStore.loadingTransactions) {
      content = this.renderLoading(t("transaction.list.loading"))
    } else if (searchStore.searchError) {
      const i18nkey =
        searchStore.searchError.code === "request_validation_error"
          ? searchStore.searchError.code
          : "generic_error"

      content = (
        <FormattedError
          title={t(`search.result.errors.${i18nkey}`)}
          error={searchStore.searchError}
        />
      )
    } else if (
      searchStore.results.length > 0 &&
      searchStore.results.length < searchStore.DEFAULT_PARAMS.limit
    ) {
      content = this.renderSearchResults(true)
    } else if (searchStore.results.length > 0) {
      content = this.renderSearchResults(false)
    } else if (!searchStore.query || searchStore.query.length === 0) {
      content = null
    } else {
      content = (
        <Cell p={[4]} minHeight="700px">
          <Text fontSize={[4]}>
            {t("search.result.noResultFoundFor")}{" "}
            <BoldText fontSize={[4]}>{decodeURIComponent(searchStore.query)}</BoldText>
          </Text>
          {this.renderNoResultsExtendSearchBox()}
        </Cell>
      )
    }

    return (
      <PageContainer>
        <Panel
          title={searchStore.query ? this.renderResultsTitle() : ""}
          renderSideTitle={this.renderSideTitle}
        >
          <PanelContentWrapper>{content}</PanelContentWrapper>
        </Panel>
      </PageContainer>
    )
  }
}
