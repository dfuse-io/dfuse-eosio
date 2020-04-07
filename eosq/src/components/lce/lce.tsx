import * as React from "react"
import { DataLoading } from "../../atoms/data-loading/data-loading.component"
import { DataError } from "../../atoms/data-error/data-error.component"
import { PromiseState } from "../../hooks/use-promise"

type Props = { promise: PromiseState<any>; loadingMessage?: string; children?: React.ReactNode }

interface LCEComponent {
  (props: Props, context?: any): React.ReactElement | null
  defaultProps?: Partial<Props>
}

export const LCE: LCEComponent = ({ promise, loadingMessage, children }) => {
  if (promise.state === "pending") {
    return <DataLoading text={loadingMessage} />
  }

  if (promise.state === "rejected") {
    return <DataError error={promise.error} />
  }

  // We expect the caller to pass a renderable component!
  return children as any
}
