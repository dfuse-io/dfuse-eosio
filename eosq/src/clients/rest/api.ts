import { DfuseError } from "@dfuse/client"
import { debugLog } from '../../services/logger'

export function legacyHandleDfuseApiError(error: any) {
  if (error instanceof DfuseError) {
    debugLog("API Error", error)
  }

  // TODO: Before, we were turning a 404 into `undefined`, not sure
  //       it's possible anymore using the current `@dfuse/client` to
  //       "know" this.

  return undefined
}
