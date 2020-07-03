import * as React from "react"
// temp ignore for dev

import { DataLoading, DataError, DataEmpty } from "@dfuse/explorer"
import { log } from "../../services/logger"
import { TaskStatusAware } from "mobx-task"
import { observer } from "mobx-react"
import { RouteComponentProps } from "react-router"
import { CursorCache } from "../../services/cursor-store"
import { NavigationButtons } from "../../atoms/navigation-buttons/navigation-buttons"
import queryString from "query-string"
import { t } from "i18next"

@observer
export class ListContentLoaderComponent<
  T extends RouteComponentProps<any>,
  S = {}
> extends React.Component<T, S> {
  PER_PAGE = 25

  cursorCache: CursorCache

  constructor(props: T) {
    super(props)
    this.cursorCache = new CursorCache()
  }

  componentDidMount(): void {
    this.componentDidMountHandler()
  }

  componentDidMountHandler(): void {
    if (this.parsed.cursor && this.parsed.cursor.length > 0) {
      this.cursorCache.setCurrentCursor(decodeURIComponent(this.parsed.cursor as string))
    }
    this.fetchListForCursor(this.cursorCache.currentCursor)
  }

  get parsed() {
    return queryString.parse(this.props.location.search)
  }

  cursoredUrl = (cursor: string): string => {
    throw new Error(`not implemented for args ${cursor}`)
  }

  fetchListForCursor(cursor: string) {
    throw new Error(`not implemented for args ${cursor}`)
  }

  onNext = () => {
    const cursor = this.cursorCache.shiftToNextCursor()
    this.props.history.replace(this.cursoredUrl(cursor))
    this.fetchListForCursor(cursor)
  }

  onPrev = () => {
    if (this.cursorCache.hasPreviousPage) {
      const cursor = this.cursorCache.shiftToPreviousCursor()
      this.props.history.replace(this.cursoredUrl(cursor))
      this.fetchListForCursor(cursor)
    }
  }

  onFirst = () => {
    const cursor = ""
    this.cursorCache.resetAll()
    this.props.history.replace(this.cursoredUrl(cursor))
    this.fetchListForCursor(cursor)
  }

  renderNavigation = (variant: string, showNext: boolean) => {
    return (
      <NavigationButtons
        showFirst={this.cursorCache.currentCursor !== ""}
        onFirst={this.onFirst}
        onNext={this.onNext}
        onPrev={this.onPrev}
        showNext={showNext}
        showPrev={this.cursorCache.hasPreviousPage}
        variant={variant}
      />
    )
  }

  renderEmpty() {
    return <DataEmpty text={t("transaction.list.empty")} />
  }

  renderLoading = (message: string) => {
    return <DataLoading text={message} />
  }

  renderError = (error?: Error) => {
    if (error && error.name !== "not_found") {
      log.error("An error occurred while fetching data.", error)
    }

    return <DataError error={error} />
  }

  renderContent = (args: any): React.ReactNode => {
    throw new Error(`not implemented for args: ${args}`)
  }

  prepareRenderContent = (collection: any[] | any): React.ReactNode => {
    if (!collection || collection.length === 0) {
      return this.renderEmpty()
    }

    this.cursorCache.prepareNextCursor(collection[collection.length - 1].key)

    return this.renderContent(collection)
  }

  handleRender = (service: TaskStatusAware<any>, loadingText: string): React.ReactNode => {
    return service.match({
      rejected: this.renderError,
      pending: () => this.renderLoading(loadingText),
      resolved: this.prepareRenderContent
    })
  }
}
