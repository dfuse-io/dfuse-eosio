import { streamVoteTally, isInboundMessageType, VoteTallyData } from "../clients/websocket/eosws"
import { InboundMessage, InboundMessageType, ErrorData } from "@dfuse/client"
import { voteStore } from "../stores"

import { getDfuseClient } from "@dfuse/explore"

export async function registerVoteTallyStream(errorCallback: (error: ErrorData) => void) {
  return streamVoteTally(getDfuseClient(), (message: InboundMessage) => {
    if (message.type === InboundMessageType.ERROR) {
      errorCallback(message.data as ErrorData)
    }

    if (isInboundMessageType(message, "vote_tally")) {
      voteStore.update((message.data as VoteTallyData).vote_tally)
    }
  })
}
