import { task } from "mobx-task"
import {
  requestAccountAbi,
  requestContractTableRows,
  requestStateTable
} from "../clients/rest/account"
import { contractTableStore } from "../stores"
import { GetTableRowParams } from "../clients/websocket/eosws"

export const fetchContractTableRows = task(
  async (params: GetTableRowParams) => {
    return requestContractTableRows(params)
  },
  { swallow: true }
)

export const fetchContractTableRowsOnContractPage = task(
  async (params: GetTableRowParams) => {
    contractTableStore.loading = true
    contractTableStore.error = false
    return requestContractTableRows(params)
      .then((tableRows: any) => {
        contractTableStore.loading = false
        contractTableStore.tableRows = tableRows
        return contractTableStore.tableRows
      })
      .catch(() => {
        contractTableStore.loading = false
        contractTableStore.error = true
      })
  },
  { swallow: true }
)

export const fetchContractTableRowsFromEOSWS = task(
  async (params: any) => {
    return requestStateTable(params)
  },
  { swallow: true }
)

export const fetchContractAbi = task(
  async (accountName: string) => {
    return requestAccountAbi(accountName)
  },
  { swallow: true }
)
