import { initializeDfuseClient } from "@dfuse/explore"
import { DfuseClient, createDfuseClient } from "@dfuse/client"
import { Config } from "../models/config"

let dfuseClient: DfuseClient
export const getDfuseClient = () => dfuseClient

export const initializeDfuseClientFromConfig = () => {
  initializeDfuseClient(
    createDfuseClient({
      apiKey: Config.dfuse_io_api_key,
      network: Config.dfuse_io_endpoint,
      authUrl: Config.dfuse_auth_endpoint,
      secure: Config.secure !== undefined && Config.secure
    })
  )
}
