import { decodedResponseToDBOps, groupDBOpHex } from "../dbop.helpers"
import { DBOp } from "@dfuse/client"

export function getDBOPMocks(): DBOp[] {
  return [
    {
      op: "ins",
      action_idx: 0,
      account: "account_a",
      table: "table_a",
      scope: "scope",
      key: "key",
      old: { payer: "payer", hex: "abcf1" },
      new: { payer: "payer", hex: "abcf2" }
    },
    {
      op: "ins",
      action_idx: 0,
      account: "account_b",
      table: "table_b",
      scope: "scope",
      old: { payer: "payer", hex: "abcf3" },
      new: { payer: "payer", hex: "abcf4" },
      key: "key"
    }
  ]
}
describe("groupDBOpHex", () => {
  it("should group the data by account::table pairs", () => {
    expect(groupDBOpHex(getDBOPMocks())).toEqual({
      "account_a::table_a": ["abcf1", "abcf2"],
      "account_b::table_b": ["abcf3", "abcf4"]
    })
  })
})

describe("decodedResponseToDBOps", () => {
  it("should take the decoded responses and update the dbops with it", () => {
    const decodedResponses = [
      {
        block_num: 123,
        account: "account_a",
        table: "table_a",
        rows: [{ data: "foo" }, { data: "bar" }]
      },
      {
        block_num: 123,
        account: "account_b",
        table: "table_b",
        rows: [{ data: "foo2" }, { data: "bar2" }]
      }
    ]

    expect(decodedResponseToDBOps(decodedResponses, getDBOPMocks())).toEqual([
      {
        account: "account_a",
        action_idx: 0,
        key: "key",
        new: { hex: "abcf2", json: { data: "bar" }, payer: "payer" },
        old: { hex: "abcf1", json: { data: "foo" }, payer: "payer" },
        op: "ins",
        scope: "scope",
        table: "table_a"
      },
      {
        account: "account_b",
        action_idx: 0,
        key: "key",
        new: { hex: "abcf4", json: { data: "bar2" }, payer: "payer" },
        old: { hex: "abcf3", json: { data: "foo2" }, payer: "payer" },
        op: "ins",
        scope: "scope",
        table: "table_b"
      }
    ])
  })
})
