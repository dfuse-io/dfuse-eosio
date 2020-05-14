// import gql from "graphql-tag"
// import { useGraphqlQuery } from "./use-graphql-query"
import { PromiseState, promiseStateResolved } from "./use-promise"
import { GraphqlResponseError } from "@dfuse/client"
import * as data from "./nft-mock.json"

export type NFT = {
  id: string
  owner: string
  author: string
  category: string
  idata: any
  mdata: any
}

export type NFTFilter = {
  owners: string[]
  authors: string[]
  categories: string[]
  id?: string
}

// const document = gql`
//   query($query: String!) {
//     nft(query: $query) {
//       tokens {
//         id
//         owner
//         author
//         category
//         idata
//         mdata
//       }
//     }
//   }
// `

// type Document = {
//   nft: {
//     tokens: NFT[]
//   }
// }

export function fetchNft(query: string): NFT[] {
  // TODO: connect to GQL query and return promise
  //   const response = useGraphqlQuery<Document>(document, { query })
  //   if (response.state === "pending" || response.state === "rejected") {
  //     return promiseStateRetype(response)
  //   }
  let assets: NFT[]

  const formattedQuery = query.toLowerCase()
  if (
    !formattedQuery.includes("authors") &&
    !formattedQuery.includes("owners") &&
    !formattedQuery.includes("categories") &&
    !formattedQuery.includes("id")
  ) {
    return data.rows
  }

  const queries = formattedQuery.split(" ").map((q) => q.trim())
  let filters: NFTFilter = {
    owners: [],
    authors: [],
    categories: []
  }

  queries.forEach((q) => {
    console.log(q)
    const filterName = q.split(":")[0]
    if (filterName === "id") {
      filters = { ...filters, id: q.split(":")[1] }
    }
    const filterValues = q
      .split(":")[1]
      .split(",")
      .filter((s) => s !== "")
    if (filterName in filters) {
      filters[filterName] = filterValues
    }
  })

  console.log(filters)

  return data.rows.filter(
    (a) =>
      (filters.owners.length <= 0 || filters.owners.includes(a.owner)) &&
      (filters.authors.length <= 0 || filters.authors.includes(a.author)) &&
      (filters.categories.length <= 0 || filters.categories.includes(a.category)) &&
      (!filters.id || filters.id === "" || filters.id === a.id)
  )
}

// TODO: potentially merge with general purpose GQL hook
export function useSingleNFT(id: string): NFT | undefined {
  const asset: NFT | undefined = data.rows.find((r) => r.id === id)

  return asset
}

const onlyUnique = (value: any, index: number, self: any[]) => {
  return self.indexOf(value) === index
}

export function useNftFilters(): NFTFilter {
  const owners = data.rows.map((r) => r.owner).filter(onlyUnique)
  const authors = data.rows.map((r) => r.author).filter(onlyUnique)
  const categories = data.rows.map((r) => r.category).filter(onlyUnique)
  return {
    owners,
    authors,
    categories,
    id: ""
  }
}
