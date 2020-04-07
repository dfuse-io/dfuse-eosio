import { generateTransactionTrace } from "../../__mocks__/transaction.mock"
import { computeTransactionTrustPercentage } from "../transaction"
import {
  computeTraceWithLevel,
  groupTracesByTraceLevel
} from "../../services/transaction-traces-wrap"

describe("transaction", () => {
  describe("computeTransactionTrustPercentage", () => {
    it("gives 0.0 on undefined block num", () => {
      expect(computeTransactionTrustPercentage(undefined, 0, 0)).toBe(0.0)
    })

    it("gives 1.0 when irreversible block > 1", () => {
      expect(computeTransactionTrustPercentage(undefined, 0, 0)).toBe(0.0)
    })

    it("gives 1.0 when irreversible block num > block_num", () => {
      expect(computeTransactionTrustPercentage(10, 12, 11)).toBe(1.0)
    })

    it("gives 0.9999 if more than 360 block passed", () => {
      expect(computeTransactionTrustPercentage(10, 370, 0)).toBe(0.9999)
    })

    it("gives 0.25 if 1 block passed", () => {
      expect(computeTransactionTrustPercentage(10, 11, 0)).toBe(0.25)
    })

    it("gives 0.50 if 2 block passed", () => {
      expect(computeTransactionTrustPercentage(10, 12, 0)).toBe(0.5)
    })

    it("gives 0.75 if 3 block passed", () => {
      expect(computeTransactionTrustPercentage(10, 13, 0)).toBe(0.75)
    })

    it("gives 0.99 if 4 block passed", () => {
      expect(computeTransactionTrustPercentage(10, 14, 0)).toBe(0.99)
    })

    it("gives fraction of 0.99 remainder to 1.0 if > 4 block && < 360 passed", () => {
      expect(computeTransactionTrustPercentage(10, 130, 0)).toBe(0.9933333333333333)
      expect(computeTransactionTrustPercentage(10, 270, 0)).toBe(0.9972222222222222)
      expect(computeTransactionTrustPercentage(10, 369, 0)).toBe(0.9999722222222223)
    })
  })

  describe("computeTraceWithLevel", () => {
    it("should flatten and adding group & level properties", () => {
      const traceLevels = computeTraceWithLevel(transactionTraces())

      expect(traceLevels[0].length).toBe(9)
      const results = traceLevels[0]

      expect(results[0]).toMatchObject({ group: 0, level: 0, index: 0 })
      expect(results[1]).toMatchObject({ group: 0, level: 1, index: 1 })
      expect(results[2]).toMatchObject({ group: 0, level: 2, index: 2 })
      expect(results[3]).toMatchObject({ group: 0, level: 2, index: 3 })
      expect(results[4]).toMatchObject({ group: 0, level: 2, index: 4 })
      expect(results[5]).toMatchObject({ group: 1, level: 0, index: 5 })
      expect(results[6]).toMatchObject({ group: 2, level: 0, index: 6 })
      expect(results[7]).toMatchObject({ group: 2, level: 1, index: 7 })
      expect(results[8]).toMatchObject({ group: 2, level: 1, index: 8 })
    })
  })

  describe("groupTracesByTraceLevel", () => {
    it("should grouped by group property", () => {
      const groups = groupTracesByTraceLevel(transactionTraces())

      expect(Object.keys(groups).length).toBe(3)

      expect(groups[0].length).toBe(5)
      expect(groups[1].length).toBe(1)
      expect(groups[2].length).toBe(3)

      expect(groups[0][0]).toMatchObject({ group: 0, level: 0 })
      expect(groups[0][1]).toMatchObject({ group: 0, level: 1 })
      expect(groups[0][2]).toMatchObject({ group: 0, level: 2 })
      expect(groups[0][3]).toMatchObject({ group: 0, level: 2 })
      expect(groups[0][4]).toMatchObject({ group: 0, level: 2 })

      expect(groups[1][0]).toMatchObject({ group: 1, level: 0 })

      expect(groups[2][0]).toMatchObject({ group: 2, level: 0 })
      expect(groups[2][1]).toMatchObject({ group: 2, level: 1 })
      expect(groups[2][2]).toMatchObject({ group: 2, level: 1 })
    })
  })
})

function transactionTraces() {
  let index = 0

  return [
    generateTransactionTrace(index++, [
      generateTransactionTrace(index++, [
        generateTransactionTrace(index++),
        generateTransactionTrace(index++),
        generateTransactionTrace(index++)
      ])
    ]),

    generateTransactionTrace(index++),

    generateTransactionTrace(index++, [
      generateTransactionTrace(index++),
      generateTransactionTrace(index++)
    ])
  ]
}
