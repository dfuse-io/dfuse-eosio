import {
  InboundMessage,
  InboundMessageType,
  TransactionLifecycle,
  ErrorData,
  TransactionLifecycleData
} from "@dfuse/client"

import { getDfuseClient } from "@dfuse/explorer"

export async function registerTransactionLifecycleListener(
  transactionID: string,
  successCallback: (lifecycle: TransactionLifecycle) => void,
  errorCallback: (error: ErrorData) => void
) {
  const stream = await getDfuseClient().streamTransaction(
    { id: transactionID },
    (message: InboundMessage) => {
      if (message.type === InboundMessageType.ERROR) {
        errorCallback(message.data as ErrorData)
        return
      }

      if (message.type === InboundMessageType.TRANSACTION_LIFECYCLE) {
        successCallback((message.data as TransactionLifecycleData).lifecycle)
      }
    }
  )

  return stream
}
