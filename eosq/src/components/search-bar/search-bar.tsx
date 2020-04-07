import { observer } from "mobx-react"
import * as React from "react"
import { SuggestionSection } from "../../models/typeahead"
import { UiTypeahead } from "../../atoms/ui-typeahead/ui-typeahead"
import { Links } from "../../routes"
import { RouteComponentProps } from "react-router"
import queryString from "query-string"
import { metricsStore, searchStore } from "../../stores"
import { formatNumber, NBSP } from "../../helpers/formatters"
import { t } from "i18next"
import { performStructuredSearch } from "../../services/search"
import { ExternalTextLink } from "../../atoms/text/text.component"
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome"
import { faQuestionCircle } from "@fortawesome/free-solid-svg-icons"
import { Cell } from "../../atoms/ui-grid/ui-grid.component"
import { theme } from "../../theme"
import {
  OmniSearchResponse,
  fetchTypeaheadSuggestions,
  omniSearch,
} from "../../clients/websocket/eosws"

interface Props extends RouteComponentProps<{}> {}

@observer
export class SearchBar extends React.Component<Props> {
  async getItems(query: string | null): Promise<SuggestionSection[]> {
    if (query === null || query.length < 1) {
      return []
    }

    query = query.replace(/,/g, "").toLowerCase()

    return fetchTypeaheadSuggestions(query).then((suggestions: SuggestionSection[]) => {
      const blockNumCandidate = parseInt(query!, 10)
      if (Number.isInteger(blockNumCandidate) && blockNumCandidate < metricsStore.headBlockNum) {
        const blockSuggestion: SuggestionSection[] = [
          {
            id: "blocks",
            suggestions: [
              { label: formatNumber(blockNumCandidate), key: blockNumCandidate.toString() },
            ],
          },
        ]
        suggestions = [...blockSuggestion, ...suggestions]
      }
      return suggestions
    })
  }

  onSubmit = (query: string): Promise<any> => {
    searchStore.query = query

    if (query.trim().length <= 0) {
      return Promise.resolve({ done: true })
    }

    const sqeMatcher: RegExp = /[a-z]:/
    if (sqeMatcher.test(query)) {
      this.onStructuredSearchSubmit(query)
      return Promise.resolve({ done: true })
    }

    return this.onSimpleSearchSubmit(query)
  }

  handleSearchResult = (result?: OmniSearchResponse) => {
    if (result === undefined) {
      this.props.history.push(`${Links.searchResults()}?query=${searchStore.query}`)
    } else if (result.type === "block") {
      window.location.href = Links.viewBlock({ id: result.data.id })
    } else if (result.type === "account") {
      this.props.history.push(Links.viewAccount({ id: result.data.account_name }))
    } else if (result.type === "transaction") {
      window.location.href = Links.viewTransaction({ id: result.data.id })
    } else if (result.type === "eth_registered") {
      this.props.history.push(Links.viewAccount({ id: result.data }))
    } else if (result.type === "eth_unregistered") {
      this.props.history.push(`${Links.searchResults()}?query=${searchStore.query}`)
    } else {
      this.props.history.push(`${Links.searchResults()}?query=${searchStore.query}`)
    }
  }

  onSimpleSearchSubmit = (query: string): Promise<any> => {
    if (
      searchStore.query === "eosio" ||
      searchStore.query === "eosio.prods" ||
      searchStore.query === "eosio.null"
    ) {
      this.props.history.push(`${Links.viewAccount({ id: searchStore.query })}`)
      return Promise.resolve()
    }

    return omniSearch(query)
      .then(this.handleSearchResult)
      .catch((error: Error) => {})
  }

  onStructuredSearchSubmit = (query: string) => {
    this.props.history.push(
      `${Links.viewTransactionSearch()}?${queryString.stringify({
        q: encodeURIComponent(query),
      })}`
    )
    performStructuredSearch("")
  }

  renderHelp() {
    return (
      <Cell textAlign="right" width="100%" px={[3]} py={[2]} bg={theme.colors.bleu10}>
        <ExternalTextLink
          fontWeight="600"
          fontSize={[2]}
          color="white"
          to="https://docs.dfuse.io/reference/eosio/search-terms"
        >
          {t("search.sqeDocumentation")}
          {NBSP}
          <FontAwesomeIcon icon={faQuestionCircle as any} />
        </ExternalTextLink>
      </Cell>
    )
  }

  render() {
    return (
      <UiTypeahead
        placeholder={t("search.placeholder")}
        defaultQuery={searchStore.query}
        getItems={this.getItems}
        onSubmit={this.onSubmit}
        help={this.renderHelp()}
        clickToFollowTypes={["accounts", "blocks", "query"]}
      />
    )
  }
}
