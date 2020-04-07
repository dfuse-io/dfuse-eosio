import * as React from "react"
import { Text } from "../text/text.component"

const errorToMessage = (error: any): string | undefined => {
  if (error == null) {
    return undefined
  }

  if (Array.isArray(error)) {
    return (error as any[]).map((element) => errorToMessage(element)).join(", ")
  }

  if (typeof error === "object" && error.path != null && error.message) {
    // FIXME: Format "path!"
    return error.message
  }

  if (typeof error === "object" && error.message) {
    return error.message
  }

  if (typeof error === "string") {
    return error
  }

  return `${error}`
}

export const DataError: React.SFC<{ error?: Error }> = ({ error }) => {
  return <Text fontSize={[4]}>{errorToMessage(error) || "An unknow error occurred"}</Text>
}
