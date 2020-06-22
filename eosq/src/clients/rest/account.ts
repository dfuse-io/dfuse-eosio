import { getTableRows, GetTableRowParams, getProducerSchedule } from "../websocket/eosws"
import { legacyHandleDfuseApiError } from "./api"
import { Abi } from "@dfuse/client"
// eslint-disable-next-line import/no-extraneous-dependencies
import { getDfuseClient } from "@dfuse/explore"

export async function requestProducerSchedule() {
  return getProducerSchedule()
}

export type StateTableParams = {
  code: string
  scope: string
  table: string
}

export async function requestStateTable(params: StateTableParams) {
  return getDfuseClient()
    .stateTable(params.code, params.scope || params.code, params.table)
    .catch(legacyHandleDfuseApiError)
}

export async function requestContractTableRows(params: GetTableRowParams) {
  const response = await getTableRows(params)
  if (response === undefined) {
    return []
  }

  return response
}

export async function requestProducerAccountTableRows(accountName: string) {
  // FIXME: Replacable by `getDfuseClient().stateTableRow`
  return requestContractTableRows({
    scope: "producerjson",
    table: "producerjson",
    code: "producerjson",
    lower_bound: accountName,
    limit: 1
  })
}

export async function requestAccountLinkedPermissions(accountName: string, blockNum: number) {
  return getDfuseClient()
    .statePermissionLinks(accountName, { blockNum })
    .catch(legacyHandleDfuseApiError)
}

export async function requestAccountAbi(accountName: string): Promise<{ abi: Abi } | undefined> {
  return getDfuseClient()
    .stateAbi(accountName)
    .catch(legacyHandleDfuseApiError)
}
