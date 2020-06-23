import {
  DfuseClient,
  StreamOptions,
  OnStreamMessage,
  Stream,
  InboundMessage,
  TransactionLifecycle
} from "@dfuse/client"
import { Account } from "../../models/account"
import { VoteTally } from "../../models/vote"
import { legacyHandleDfuseApiError } from "../rest/api"
import { BlockSummary } from "../../models/block"
import { SuggestionSection } from "../../models/typeahead"
import { log } from "../../services/logger"

import { getDfuseClient } from "@dfuse/explore"

// Account

export interface AccountData {
  account: Account
}

export async function streamAccount(
  client: DfuseClient,
  account: string,
  onMessage: OnStreamMessage,
  options: StreamOptions = {}
): Promise<Stream> {
  return client.websocketStream(onMessage, (messageCreator, withDefaultOptions) => {
    return messageCreator(
      // @ts-ignore Private outbound message type not exposed publicly
      "get_account",
      { name: account },
      withDefaultOptions({ fetch: true, ...options })
    )
  })
}

// Blocks

export async function getBlock(blockID: string): Promise<BlockSummary | undefined> {
  return getDfuseClient()
    .apiRequest<BlockSummary>(`/v0/blocks/${blockID}`, "GET", undefined, undefined)
    .catch(legacyHandleDfuseApiError)
}

export async function listBlocks(skip: number, perPage: number): Promise<BlockSummary[]> {
  return getDfuseClient()
    .apiRequest<BlockSummary[]>("/v0/blocks", "GET", { skip, limit: perPage }, undefined)
    .catch(legacyHandleDfuseApiError)
    .then((value) => value || [])
}

export async function listBlockTransactions(
  blockID: string,
  cursor: string,
  perPage: number
): Promise<ListTransactionsResponse | undefined> {
  return getDfuseClient()
    .apiRequest<ListTransactionsResponse>(
      `/v0/blocks/${blockID}/transactions`,
      "GET",
      { cursor, limit: perPage },
      undefined
    )
    .catch(legacyHandleDfuseApiError)
}

// Completion

export async function fetchTypeaheadSuggestions(prefix: string): Promise<SuggestionSection[]> {
  return getDfuseClient()
    .apiRequest<SuggestionSection[]>("/v0/search/completion", "GET", { prefix }, undefined)
    .catch(legacyHandleDfuseApiError)
    .then((value) => value || [])
}

// Get Table Rows

export interface GetTableRowParams {
  json?: boolean
  scope: string
  table: string
  code: string
  table_key?: string
  lower_bound?: string
  upper_bound?: string
  limit?: number
  key_type?: string
  index_position?: string
}

export async function getTableRows(params: GetTableRowParams): Promise<unknown | undefined> {
  return getDfuseClient()
    .apiRequest(
      "/v1/chain/get_table_rows",
      "POST",
      {},
      {
        json: params.json !== undefined ? params.json : true,
        scope: params.scope === "" ? params.code : params.scope,
        table: params.table,
        code: params.code,
        table_key: params.table_key,
        lower_bound: params.lower_bound,
        limit: params.limit,
        key_type: params.key_type,
        index_position: params.index_position
      }
    )
    .catch(legacyHandleDfuseApiError)
}

export interface ProducerScheduleResponse {
  active: {
    version: number
    producers: ProducerScheduleItem[]
  }
}

export interface ProducerScheduleItem {
  producer_name: string
  block_signing_key: string
}

export async function getProducerSchedule() {
  return getDfuseClient()
    .apiRequest<ProducerScheduleResponse | undefined>(
      "/v1/chain/get_producer_schedule",
      "GET",
      {},
      undefined
    )
    .catch(legacyHandleDfuseApiError)
}

// Price

export type PriceData = {
  symbol: string
  price: number
  variation: number
  last_updated: string
}

export async function streamPrice(
  client: DfuseClient,
  onMessage: OnStreamMessage,
  options: StreamOptions = {}
): Promise<Stream> {
  return client.websocketStream(onMessage, (messageCreator, withDefaultOptions) => {
    return messageCreator(
      // @ts-ignore Private outbound message type not exposed publicly
      "get_price",
      {},
      withDefaultOptions({ fetch: true, listen: true, ...options })
    )
  })
}

// Search

export type OmniSearchResponse =
  | BlockOmniSearchResponse
  | AccountOmniSearchResponse
  | TransactionOmniSearchResponse
  | GenesisRegisteredOmniSearchResponse

type AccountOmniSearchResponse = { type: "account"; data: Account }
type BlockOmniSearchResponse = { type: "block"; data: BlockSummary }
type TransactionOmniSearchResponse = { type: "transaction"; data: TransactionLifecycle }
type GenesisRegisteredOmniSearchResponse = {
  type: "eth_registered" | "eth_unregistered"
  data: string
}

export async function omniSearch(query: string): Promise<OmniSearchResponse | undefined> {
  log.info("Performing search query with query [%s].", query)
  const result = await getDfuseClient()
    .apiRequest<OmniSearchResponse>(
      "/v0/simple_search",
      "GET",
      { q: query.replace(/,/g, "").trim() },
      undefined
    )
    .catch(legacyHandleDfuseApiError)

  if (result == null) {
    log.info("No search result found for query [%s] via API.", query)
    return undefined
  }

  log.info("Search result for query [%s] found via API.", query, result)
  return result
}

// Transactions

export interface ListTransactionsResponse {
  transactions: TransactionLifecycle[]
  cursor: string
}

export async function listTransactions(
  cursor: string,
  perPage: number
): Promise<ListTransactionsResponse | undefined> {
  return getDfuseClient()
    .apiRequest<ListTransactionsResponse>(
      "/v0/transactions",
      "GET",
      { cursor, limit: perPage },
      undefined
    )
    .catch(legacyHandleDfuseApiError)
}

// Vote Tally

export type VoteTallyData = {
  vote_tally: VoteTally
}

export async function streamVoteTally(
  client: DfuseClient,
  onMessage: OnStreamMessage,
  options: StreamOptions = {}
): Promise<Stream> {
  return client.websocketStream(onMessage, (messageCreator, withDefaultOptions) => {
    return messageCreator(
      // @ts-ignore Private outbound message type not exposed publicly
      "get_vote_tally",
      {},
      withDefaultOptions({ fetch: true, listen: true, ...options })
    )
  })
}

export function isInboundMessageType(message: InboundMessage, expectedType: string): boolean {
  // @ts-ignore Private non-public message type
  return message.type === expectedType
}
