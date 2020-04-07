import { useEffect, useState } from "react"
import { GraphqlResponseError } from "@dfuse/client"
import { DocumentNode } from "graphql"
import { print as printGraphqlDocument } from "graphql/language/printer"
import { getDfuseClient } from "../data/dfuse"
import {
  PromiseState,
  promiseStatePending,
  promiseStateRejected,
  promiseStateResolved
} from "./use-promise"

export function useGraphqlQuery<T = any>(
  document: string | DocumentNode,
  variables: Record<string, unknown> = {}
): PromiseState<T, GraphqlResponseError[]> {
  const [state, setState] = useState<PromiseState<T, GraphqlResponseError[]>>(promiseStatePending())

  let stringDocument = document as string
  if (typeof document !== "string") {
    stringDocument = printGraphqlDocument(document)
  }

  useEffect(() => {
    setState(promiseStatePending())
    ;(async () => {
      try {
        const response = await getDfuseClient().graphql<T>(stringDocument, { variables })
        if (response.errors) {
          setState(promiseStateRejected(response.errors))
        } else {
          setState(promiseStateResolved(response.data))
        }
      } catch (error) {
        setState(
          promiseStateRejected([
            {
              message: `${error}`,
              extensions: { cause: error }
            }
          ])
        )
      }
    })()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [stringDocument, JSON.stringify(variables)])

  return state
}
