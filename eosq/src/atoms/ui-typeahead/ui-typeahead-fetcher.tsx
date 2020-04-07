import { debounce } from "ts-debounce"
import * as React from "react"
import { SuggestionSection } from "../../models/typeahead"

interface Props {
  searchValue: string | null
  fetchData: (input: string, options: any) => Promise<SuggestionSection[]>
  onLoaded: (suggestions?: SuggestionSection[], error?: Error) => void
  children: FetcherChildrenFunction<any>
}

export type FetcherControllerStateAndHelpers<Item> = State

export type FetcherChildrenFunction<Item> = (
  options: FetcherControllerStateAndHelpers<Item>
) => React.ReactNode

interface State {
  suggestions: SuggestionSection[]
  loading: boolean
  error?: Error
}

export class UiTypeaheadFetcher extends React.Component<Props, State> {
  requestId = 0
  state = { loading: false, error: undefined, suggestions: [] }
  mounted = false

  fetch = debounce(() => {
    if (!this.mounted) {
      return
    }
    this.requestId++
    this.props
      .fetchData(this.props.searchValue ? this.props.searchValue : "", {
        requestId: this.requestId
      })
      .then(
        (suggestions: SuggestionSection[]) => {
          if (this.mounted) {
            this.props.onLoaded(suggestions, undefined)
            this.setState({ loading: false, suggestions })
          }
        },
        (error: Error) => {
          if (this.mounted) {
            this.props.onLoaded(undefined, error)
            this.setState({ loading: false, error })
          }
        }
      )
  }, 300)

  reset(overrides: any) {
    this.setState({ loading: false, error: null, suggestions: [], ...overrides })
  }

  prepareFetch() {
    this.reset({ loading: true })
  }
  componentDidMount() {
    this.mounted = true
    this.prepareFetch()
    this.fetch()
  }
  componentDidUpdate(prevProps: Props) {
    if (prevProps.searchValue !== this.props.searchValue) {
      this.prepareFetch()
      this.fetch()
    }
  }
  componentWillUnmount() {
    this.mounted = false
  }
  render() {
    return this.props.children ? this.props.children(this.state) : null
  }
}
