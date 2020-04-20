export type PromiseState<T, E = any> = (PromisePending | PromiseResolved<T> | PromiseRejected<E>) &
  PromiseHelpers<T>

export type PromiseHelpers<T> = {
  resultOr(orFallbackTo: T): T
}

export type PromisePending = {
  state: "pending"
}

export type PromiseResolved<T> = {
  state: "resolved"
  result: T
}

export type PromiseRejected<E> = {
  state: "rejected"
  error: E
}

export function promiseStatePending<T, E = any>(): PromiseState<T, E> {
  return {
    resultOr: (other: T): T => other,
    state: "pending"
  }
}

export function promiseStateRejected<T, E = any>(error: E): PromiseState<T, E> {
  return {
    resultOr: (other: T): T => other,
    state: "rejected",
    error
  }
}

export function promiseStateResolved<T, E = any>(result: T): PromiseState<T, E> {
  return {
    resultOr: (): T => result,
    state: "resolved",
    result
  }
}

/**
 * This is only use for re-typing purposes. Ideally, it would not be a function call,
 * but we are ready to pay the small footprint it adds for now.
 */
export function promiseStateRetype<T, E>(state: PromiseState<any, any>): PromiseState<T, E> {
  return state as any
}

// Incomplete work for now. Ideally, we generalize `use-graphql-query` logic
// of the Promise execution + try/catch the transaformation to proper state
// object. For now, only the base re-usable types and helpers are defined.
// export function usePromise<T, E = any>(promiseFactory:()=>PromiseLike<T>): PromiseState<T, E> {
//   const [state, setState] = useState<PromiseState<T, E>>(promiseStatePending())

//   useEffect(() => {
//     setState(promiseStatePending())
//     ;(async () => {
//       try {
//         const response = await promiseFactory()

//         // Some composable hooks will need to hook here somehow, to turn the receive value in either
//         // resolved/reject states, see `use-graphql-query.ts`.
//         setState(promiseStateResolved(response))
//       } catch (error) {
//         setState(promiseStateRejected(error))
//       }
//     })()
//     // eslint-disable-next-line react-hooks/exhaustive-deps
//   }, /* Actual dependencies is more defined by user of this hook, maybe would need to pass them in in `usePromise` see `use-graphql-query.ts` */)

//   return state
// }
