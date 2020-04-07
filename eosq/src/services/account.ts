import { TokenInfo } from "../helpers/airdrops-list"

export interface GroupedTokenTableRow {
  rows: TokenTableRow[]
  account: string
  scope: string
}

export interface TokenTableRow {
  json: any
  key: string
  payer: string
}

export interface AccountTokenResponse {
  tables: GroupedTokenTableRow[]
  last_irreversible_block_id: string
  last_irreversible_block_num: number
  up_to_block_id: string
  up_to_block_num: number
}

export interface TokenBalance {
  balance: string
  claimed: boolean | undefined
  tokenInfo: TokenInfo
}
