import * as React from "react"
import { DataLoading, DataError } from "@dfuse/explorer"
import { debugLog } from "../../services/logger"
import { TaskStatusAware } from "mobx-task"
import { observer } from "mobx-react"

@observer
export class ContentLoaderComponent<T, S = {}> extends React.Component<T, S> {
  renderLoading = (message: string) => {
    return <DataLoading text={message} />
  }

  renderError = (error?: Error) => {
    if (error && error.name !== "not_found") {
      debugLog("An error occurred while fetching data.", error)
    }

    return <DataError error={error} />
  }

  renderContent = (args: any): React.ReactNode => {
    throw new Error(`not implemented for args: ${args}`)
  }

  handleRender = (service: TaskStatusAware<any>, loadingText: string): React.ReactNode => {
    return service.match({
      rejected: this.renderError,
      pending: () => this.renderLoading(loadingText),
      resolved: this.renderContent
    })
  }
}
