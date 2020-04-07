declare module "mobx-task" {
  type NoArgWorker<U> = () => U
  type OneArgWorker<T1, U> = (a: T1) => U
  type TwoArgWorker<T1, T2, U> = (a: T1, b: T2) => U
  type ThreeArgWorker<T1, T2, T3, U> = (a: T1, b: T2, c: T3) => U

  interface TaskStatusAware<U> {
    match: any

    result: U

    pending: boolean
    resolved: boolean
    rejected: boolean
  }

  interface NoArgTask<U> extends TaskStatusAware<U> {
    (): U
  }

  interface OneArgTask<T1, U> extends TaskStatusAware<U> {
    (a: T1): U
  }

  interface TwoArgTask<T1, T2, U> extends TaskStatusAware<U> {
    (a: T1, b: T2): U
  }

  interface ThreeArgTask<T1, T2, T3, U> extends TaskStatusAware<U> {
    (a: T1, b: T2, c: T3): U
  }

  export function task<U>(worker: NoArgWorker<U>, options?: Object): NoArgTask<U>
  export function task<T1, U>(worker: OneArgWorker<T1, U>, options?: Object): OneArgTask<T1, U>
  export function task<T1, T2, U>(
    worker: TwoArgWorker<T1, T2, U>,
    options?: Object
  ): TwoArgTask<T1, T2, U>

  export function task<T1, T2, T3, U>(
    worker: ThreeArgWorker<T1, T2, T3, U>,
    options?: Object
  ): ThreeArgTask<T1, T2, T3, U>
}
