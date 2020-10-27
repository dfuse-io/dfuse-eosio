import { getStatusBadgeVariant, getTransactionStatusColor } from "../transaction.helpers"
import { getActionTraceMock, getTransactionLifeCycleMock } from "../../__mocks__/transaction.mock"
import { TransactionReceiptStatus } from "../../models/transaction"
import { StatusBadgeVariant } from "../../atoms/status-badge/status-badge"
import { TransactionLifecycleWrap } from "../../services/transaction-lifecycle"

describe("executionTrace", () => {
  it("should return the execution trace if any", () => {
    const lifecycleWrap = new TransactionLifecycleWrap(getTransactionLifeCycleMock())

    expect(lifecycleWrap.executionTrace).toEqual(getTransactionLifeCycleMock().execution_trace)
  })
})

describe("blockNum", () => {
  it("should get the block num from the transaction", () => {
    const lifeCycle = getTransactionLifeCycleMock()

    lifeCycle.execution_trace = undefined!

    const lifecycleWrap = new TransactionLifecycleWrap(lifeCycle)
    expect(lifecycleWrap.blockNum).toEqual(lifeCycle.created_by!.block_num)
  })

  it("should get it from the traces if no transaction", () => {
    const lifeCycle = getTransactionLifeCycleMock()
    const lifecycleWrap = new TransactionLifecycleWrap(lifeCycle)

    expect(lifecycleWrap.blockNum).toEqual(lifeCycle.execution_trace!.block_num)
  })
})

describe("blockId", () => {
  it("should get the block id from the transaction", () => {
    const lifecycleWrap = new TransactionLifecycleWrap(getTransactionLifeCycleMock())

    expect(lifecycleWrap.blockId).toEqual(
      getTransactionLifeCycleMock().execution_trace!.producer_block_id
    )
  })
  it("should get it from the traces if no transaction", () => {
    const lifeCycle = getTransactionLifeCycleMock()
    lifeCycle.execution_trace = undefined!
    const lifecycleWrap = new TransactionLifecycleWrap(lifeCycle)

    expect(lifecycleWrap.blockId).toEqual(getTransactionLifeCycleMock().created_by!.block_id)
  })
})

describe("getTransactionStatusColor", () => {
  it("should get the red color from a failed status", () => {
    expect(getTransactionStatusColor(TransactionReceiptStatus.SOFT_FAIL)).toEqual("statusBadgeBan")
    expect(getTransactionStatusColor(TransactionReceiptStatus.HARD_FAIL)).toEqual("statusBadgeBan")
    expect(getTransactionStatusColor(TransactionReceiptStatus.CANCELED)).toEqual("statusBadgeBan")
    expect(getTransactionStatusColor(TransactionReceiptStatus.EXPIRED)).toEqual("statusBadgeBan")
  })
  it("should get the yellow color from a delayed", () => {
    expect(getTransactionStatusColor(TransactionReceiptStatus.DELAYED)).toEqual("statusBadgeClock")
  })
  it("should get the green color from a delayed", () => {
    expect(getTransactionStatusColor(TransactionReceiptStatus.EXECUTED)).toEqual("statusBadgeCheck")
  })
})

describe("getStatusBadgeVariant", () => {
  it("should get the red variant from a failed status", () => {
    expect(getStatusBadgeVariant(TransactionReceiptStatus.SOFT_FAIL)).toEqual(
      StatusBadgeVariant.BAN
    )
    expect(getStatusBadgeVariant(TransactionReceiptStatus.HARD_FAIL)).toEqual(
      StatusBadgeVariant.BAN
    )
    expect(getStatusBadgeVariant(TransactionReceiptStatus.CANCELED)).toEqual(StatusBadgeVariant.BAN)
    expect(getStatusBadgeVariant(TransactionReceiptStatus.EXPIRED)).toEqual(StatusBadgeVariant.BAN)
  })
  it("should get the yellow color from a delayed", () => {
    expect(getStatusBadgeVariant(TransactionReceiptStatus.DELAYED)).toEqual(
      StatusBadgeVariant.CLOCK
    )
  })
  it("should get the green color from a delayed", () => {
    expect(getStatusBadgeVariant(TransactionReceiptStatus.EXECUTED)).toEqual(
      StatusBadgeVariant.CHECK
    )
  })
})

describe("totalActionCount", () => {
  it("get the total count of actions with 1 action trace", () => {
    const lifecycleWrap = new TransactionLifecycleWrap(getTransactionLifeCycleMock())

    expect(lifecycleWrap.totalActionCount).toEqual(1)
  })
  it("get the total count of actions with 3 action traces", () => {
    const lifecycleWrap = new TransactionLifecycleWrap(getTransactionLifeCycleMock())

    lifecycleWrap.lifecycle.execution_trace.action_traces[0].inline_traces = [
      getActionTraceMock({ data: {} }),
      getActionTraceMock({ data: {} })
    ]
    expect(lifecycleWrap.totalActionCount).toEqual(3)
  })
})
