import { ErrorData } from "@dfuse/client"

export class ApiError extends Error {
  statusCode: number

  constructor(statusCode: number) {
    super()
    this.statusCode = statusCode

    Object.setPrototypeOf(this, ApiError.prototype)
  }
}

export class AssertionError extends Error {
  constructor(message: string) {
    super(message)

    Object.setPrototypeOf(this, AssertionError.prototype)
  }
}

export function getUnhandledError(): ErrorData {
  return {
    code: "unhandled",
    trace_id: "unknown",
    message: "",
    details: {}
  }
}
