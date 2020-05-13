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

export function useNft(query: string): PromiseState<NFT[], GraphqlResponseError[]> {
  // TODO: connect to GQL query and return promise
  //   const response = useGraphqlQuery<Document>(document, { query })
  //   if (response.state === "pending" || response.state === "rejected") {
  //     return promiseStateRetype(response)
  //   }

  const balances: NFT[] = data.rows

  return promiseStateResolved(balances)
}
