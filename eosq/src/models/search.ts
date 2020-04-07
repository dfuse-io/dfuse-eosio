export interface SearchQueryParams {
  q: string
  sort: "asc" | "desc"
  startBlock?: number
  blockCount?: number
  cursor?: string
  withReversible?: boolean
  limit?: number
}

// Those params where like that (snake case) before a refactor
// to make them fit our name casing even in URL params. Usually,
// we would not care about those.
//
// But the params are actually used for by our monitoring tools
// that make query with those exact params name. So we decided
// to support them.
export interface LegacySearchQueryParams {
  start_block?: number
  block_count?: number
  with_reversible?: boolean
}

export function upgradeLegacySearchQueryParams(
  params: SearchQueryParams & LegacySearchQueryParams
): SearchQueryParams {
  if (params.block_count != null && params.blockCount == null) {
    params.blockCount = params.block_count
    delete params.block_count
  }

  if (params.start_block != null && params.startBlock == null) {
    params.startBlock = params.start_block
    delete params.start_block
  }

  if (params.with_reversible != null && params.withReversible == null) {
    params.withReversible = params.with_reversible
    delete params.with_reversible
  }

  return params
}
