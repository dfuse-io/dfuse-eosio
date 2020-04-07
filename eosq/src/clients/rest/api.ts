import { DfuseError } from "@dfuse/client"

export function legacyHandleDfuseApiError(error: any) {
  if (error instanceof DfuseError) {
    console.warn("API Error", JSON.stringify(error))
  }

  // TODO: Before, we were turning a 404 into `undefined`, not sure
  //       it's possible anymore using the current `@dfuse/client` to
  //       "know" this.

  return undefined
}
