// import { useGraphqlQuery } from "./use-graphql-query"
// import { PromiseState, promiseStateResolved } from "../hooks/use-promise"
// import { GraphqlResponseError } from "@dfuse/client"
import { useQuery } from "@apollo/react-hooks"
import gql from "graphql-tag"
import * as mockData from "./nft-mock.json"

const buildQuery = (filters: NFTFilter) => {
  const fetchAssetsFilters = []
  let filtersString: string
  if (filters.owners && filters.owners.length > 0) {
    fetchAssetsFilters.push(`owner: { _in: [${filters.owners.map((f) => `"${f}"`).join()}]}`)
  }
  if (filters.authors && filters.authors.length > 0) {
    fetchAssetsFilters.push(`author: { _in: [${filters.authors.map((f) => `"${f}"`).join()}]}`)
  }
  if (filters.categories && filters.categories.length > 0) {
    fetchAssetsFilters.push(`category: { _in: [${filters.categories.map((f) => `"${f}"`).join()}]}`)
  }

  filtersString = fetchAssetsFilters.join()

  if (filters.id && filters.id !== "") {
    filtersString = `id: {_eq: ${filters.id}}`
  }
  const fetchAssetQuery = gql`
    {
      v2_simpleassets_sassets(limit: 20, where: {${filtersString}}) {
        id
        author
        owner
        idata
        mdata
        category
      }
    }
  `
  return fetchAssetQuery
}

const allAuthorsQuery = gql`
  {
    v2_simpleassets_sassets(distinct_on: author) {
      author
    }
  }
`

const allCategoriesQuery = gql`
  {
    v2_simpleassets_sassets(distinct_on: category) {
      category
    }
  }
`

export type NFT = {
  id: string
  owner: string
  author: string
  category?: string
  idata?: any
  mdata?: any
}

export type NFTFilter = {
  owners: string[]
  authors: string[]
  categories: string[]
  id?: string
}

export function useNft(filters: NFTFilter): NFT[] {
  const query = buildQuery(filters)
  const { loading, error, data } = useQuery(query)
  if (loading || error) return []
  return data.v2_simpleassets_sassets as NFT[]
}

export function useSingleNFT(id: string): NFT | undefined {
  const query = buildQuery({ id } as NFTFilter)
  const { loading, error, data } = useQuery(query)
  if (loading || error || !data.v2_simpleassets_sassets || data.v2_simpleassets_sassets.length <= 0)
    return undefined
  return data.v2_simpleassets_sassets[0] as NFT
}

export function useNftFilters(): NFTFilter {
  const authorsRes = useQuery(allAuthorsQuery).data
  const authors =
    authorsRes?.v2_simpleassets_sassets?.map((a: any) => (typeof a === "string" ? a : a.author)) ||
    []
  const categoriesRes = useQuery(allCategoriesQuery).data
  const categories =
    categoriesRes?.v2_simpleassets_sassets?.map((c: any) =>
      typeof c === "string" ? c : c.category
    ) || []
  return {
    owners: [], // too many owners for now, do not use as filter
    authors,
    categories,
    id: ""
  }
}
