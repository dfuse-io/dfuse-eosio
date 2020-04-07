import {
  getActionMock,
  getActionTraceMock,
  getTraceInfoMock
} from "../../__mocks__/transaction.mock"
import {
  getBetReceiptLevel1Fields,
  getBlobUrlFromPayload,
  getBuyRamBytesLevel1Fields,
  getBuyRamLevel1Fields,
  getClaimAmounts,
  getClaimRewardsLevel1Fields,
  getClaimRewardsLevel2Fields,
  getDelegatebwLevel1Fields,
  getDelegatebwLevel2Fields,
  getLinkAuthLevel1Fields,
  getLinkAuthLevel2Fields,
  getNewAccountLevel1Fields,
  getNewAccountLevel2Fields,
  getRefundLevel1Fields,
  getRefundTransfer,
  getResolveBetAmounts,
  getUndelegatebwLevel1Fields,
  getUndelegatebwLevel2Fields,
  getUpdateAuthLevel1Fields,
  getUpdateAuthLevel2Fields
} from "../../components/action-pills/templates/pill-template.helpers"
import { TraceInfo } from "../action.helpers"

function getClaimTraceInfo(): TraceInfo {
  let traceInfo = getTraceInfoMock({
    data: { from: "eosio.vpay", quantity: "30.0000 EOS" }
  })
  traceInfo.inline_traces.push(
    getActionTraceMock({ data: { from: "eosio.bpay", quantity: "50.0000 EOS" } })
  )
  return traceInfo
}

describe("getClaimAmounts", () => {
  it("should extract the amounts from the data", () => {
    expect(getClaimAmounts(getClaimTraceInfo())).toEqual(["80.0000 EOS", 50, 30, 0])
  })
})

describe("getResolveBetAmounts", () => {
  it("should...", () => {
    let traceInfo = getTraceInfoMock({
      data: { from: "eosbets", quantity: "30.0000 EOS", to: "winner" }
    })
    traceInfo.inline_traces[0].act.name = "transfer"
    expect(getResolveBetAmounts(traceInfo)).toEqual(["30.0000", "EOS", "winner"])
  })
})

describe("getBlobUrlFromPayload", () => {
  it("should call revoke and create if old url provided", () => {
    URL.revokeObjectURL = jest.fn()
    URL.createObjectURL = jest.fn()

    spyOn(URL, "revokeObjectURL")
    spyOn(URL, "createObjectURL").and.returnValue("test.url")

    expect(getBlobUrlFromPayload("abc", "old.url")).toEqual([
      "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
      "test.url"
    ])
    expect(URL.revokeObjectURL).toHaveBeenCalledWith("old.url")
    expect(URL.createObjectURL).toHaveBeenCalled()
  })
})

describe("getRefundTransfer", () => {
  it("should return the transfer actionTrace", () => {
    let traceInfo = getTraceInfoMock({
      data: { from: "eosbets", quantity: "30.0000 EOS", to: "winner" }
    })
    traceInfo.inline_traces[0].act.name = "transfer"
    traceInfo.inline_traces.push(
      getActionTraceMock({ data: { from: "eosio.bpay", quantity: "50.0000 EOS" } })
    )
    expect(getRefundTransfer(traceInfo)).toEqual(traceInfo.inline_traces[0])
  })
})

//******************************************************************************************* //

describe("getBetReceiptLevel1Fields", () => {
  it("should return the level 1 fields", () => {
    const action = getActionMock({
      data: { bettor: "bettor", payout: "30.0000 TOKENS", random_roll: "abc123" }
    })
    expect(getBetReceiptLevel1Fields(action)).toEqual([
      { name: "account", type: "accountLink", value: "bettor" },
      { name: "EOSAmount", type: "bold", value: "30.0000 TOKENS" },
      { name: "roll", type: "bold", value: "abc123" }
    ])
  })
})

describe("getBuyRamBytesLevel1Fields", () => {
  it("should return the level 2 fields", () => {
    const action = getActionMock({
      data: { payer: "payer", bytes: 12400, receiver: "receiver" }
    })
    expect(getBuyRamBytesLevel1Fields(action)).toEqual([
      { name: "payer", type: "accountLink", value: "payer" },
      { name: "bytes", type: "bold", value: "12.4 KB" },
      { name: "receiver", type: "accountLink", value: "receiver" }
    ])
  })
})

describe("getBuyRamLevel1Fields", () => {
  it("should return the level 1 fields", () => {
    const action = getActionMock({
      data: { payer: "payer", quantity: "20.0000 EOS", receiver: "receiver" }
    })
    expect(getBuyRamLevel1Fields(action)).toEqual([
      { name: "payer", type: "accountLink", value: "payer" },
      { name: "amountEOS", type: "bold", value: "20.0000 EOS" },
      { name: "receiver", type: "accountLink", value: "receiver" }
    ])
  })
})

describe("getClaimRewardsLevel1Fields", () => {
  it("should return the level 1 fields", () => {
    const action = getActionMock({
      data: { owner: "owner" }
    })
    expect(getClaimRewardsLevel1Fields(action, getClaimTraceInfo())).toEqual([
      { name: "account", type: "accountLink", value: "owner" },
      { name: "amountEOS", type: "bold", value: "80.0000 EOS" }
    ])
  })
})

describe("getClaimRewardsLevel2Fields", () => {
  it("should return the level 2 fields", () => {
    const action = getActionMock({
      data: { owner: "owner" }
    })
    expect(getClaimRewardsLevel2Fields(action, getClaimTraceInfo())).toEqual([
      { name: "account", type: "accountLink", value: "owner" },
      { name: "amountbEOS", type: "bold", value: "50 EOS" },
      { name: "amountvEOS", type: "bold", value: "30 EOS" }
    ])
  })
})

describe("getDelegatebwLevel1Fields", () => {
  it("should return the level 1 fields", () => {
    const action = getActionMock({
      data: {
        from: "from",
        receiver: "receiver",
        stake_cpu_quantity: "1.0000 EOS",
        stake_net_quantity: "2.0000 EOS"
      }
    })
    expect(getDelegatebwLevel1Fields(action)).toEqual([
      { name: "from", type: "accountLink", value: "from" },
      { name: "amountCPU", type: "bold", value: "1.0000 EOS" },
      { name: "amountNET", type: "bold", value: "2.0000 EOS" },
      { name: "to", type: "accountLink", value: "receiver" }
    ])
  })
})

describe("getDelegatebwLevel2Fields", () => {
  it("should return the level 2 fields", () => {
    const action = getActionMock({
      data: {
        from: "from",
        receiver: "receiver",
        stake_cpu_quantity: "1.0000 EOS",
        stake_net_quantity: "2.0000 EOS"
      }
    })

    expect(getDelegatebwLevel2Fields(action)).toEqual([
      { name: "amountCPU", type: "bold", value: "1.0000 EOS" },
      { name: "amountNET", type: "bold", value: "2.0000 EOS" }
    ])
  })
})

describe("getLinkAuthLevel1Fields", () => {
  it("should return the level 1 fields", () => {
    const action = getActionMock({
      data: {
        account: "account",
        requirement: "requirement",
        type: "type",
        code: "code.account"
      }
    })
    expect(getLinkAuthLevel1Fields(action)).toEqual([
      { name: "account", type: "accountLink", value: "account" },
      { name: "requirement", type: "bold", value: "requirement" },
      { name: "type", type: "bold", value: "type" },
      { name: "code", type: "accountLink", value: "code.account" }
    ])
  })
})

describe("getLinkAuthLevel2Fields", () => {
  it("should return the level 2 fields", () => {
    const action = getActionMock({
      data: {
        account: "account",
        requirement: "requirement",
        type: "type",
        code: "code.account"
      }
    })
    expect(getLinkAuthLevel2Fields(action)).toEqual([
      { name: "requirement", type: "bold", value: "requirement" },
      { name: "type", type: "bold", value: "type" },
      { name: "code", type: "bold", value: "code.account" }
    ])
  })
})

describe("getNewAccountLevel1Fields", () => {
  it("should return the level 1 fields", () => {
    const action = getActionMock({
      data: {
        creator: "creator",
        name: "name"
      }
    })
    expect(getNewAccountLevel1Fields(action)).toEqual([
      { name: "creator", type: "accountLink", value: "creator" },
      { name: "name", type: "accountLink", value: "name" }
    ])
  })
})

describe("getNewAccountLevel2Fields", () => {
  it("should return the level 2 fields for account", () => {
    const permission = {
      permission: {
        permission: "permissionName",
        actor: "actor"
      }
    }

    expect(getNewAccountLevel2Fields(permission, "owner", "account")).toEqual([
      { name: "permission", type: "bold", value: "owner" },
      { name: "account", type: "accountLink", value: "actor" },
      { name: "accountPermission", type: "bold", value: "permissionName" }
    ])
  })

  it("should return the level 2 fields for key", () => {
    const permission = {
      key: "key1234"
    }

    expect(getNewAccountLevel2Fields(permission, "owner", "key")).toEqual([
      { name: "permission", type: "bold", value: "owner" },
      { name: "key", type: "plain", value: "key1234" }
    ])
  })

  it("should for wait", () => {
    const permission = {
      key: "key1234"
    }

    expect(getNewAccountLevel2Fields(permission, "owner", "wait")).toEqual([
      { name: "permission", type: "bold", value: "owner" },
      { name: "wait", type: "plain", value: "key1234" }
    ])
  })
})

describe("getRefundLevel1Fields", () => {
  it("should return the level 1 fields", () => {
    let traceInfo = getTraceInfoMock({
      data: { from: "eosio", quantity: "15.0000 EOS", to: "winner" }
    })
    traceInfo.inline_traces[0].act.name = "transfer"

    const action = getActionMock({ data: { owner: "owner" } })
    expect(getRefundLevel1Fields(action, traceInfo)).toEqual([
      { name: "refundAmount", type: "bold", value: "15.0000 EOS" },
      { name: "owner", type: "accountLink", value: "owner" }
    ])
  })
})

describe("getResolveBetLevel1Fields", () => {
  it("should return the level 1 fields", () => {
    let traceInfo = getTraceInfoMock({
      data: { from: "eosbets", quantity: "30.0000 EOS", to: "winner" }
    })

    traceInfo.inline_traces[0].act.name = "transfer"
    const action = getActionMock({ data: { owner: "owner" } })
    expect(getRefundLevel1Fields(action, traceInfo)).toEqual([
      { name: "refundAmount", type: "bold", value: "30.0000 EOS" },
      { name: "owner", type: "accountLink", value: "owner" }
    ])
  })
})

describe("getUndelegatebwLevel1Fields", () => {
  it("should return the level 1 fields", () => {
    const action = getActionMock({
      data: { from: "from", unstake_cpu_quantity: "2.0000 EOS", unstake_net_quantity: "3.0000 EOS" }
    })
    expect(getUndelegatebwLevel1Fields(action)).toEqual([
      { name: "from", type: "accountLink", value: "from" },
      { name: "amountCPU", type: "bold", value: "2.0000 EOS" },
      { name: "amountNET", type: "bold", value: "3.0000 EOS" }
    ])
  })
})

describe("getUndelegatebwLevel2Fields", () => {
  it("should return the level 2 fields", () => {
    const action = getActionMock({
      data: { from: "from", unstake_cpu_quantity: "2.0000 EOS", unstake_net_quantity: "3.0000 EOS" }
    })
    expect(getUndelegatebwLevel2Fields(action)).toEqual([
      { name: "total", type: "bold", value: "5.0000 EOS" }
    ])
  })
})

describe("getUpdateAuthLevel1Fields", () => {
  it("should return the level 1 fields", () => {
    const action = getActionMock({
      data: { account: "account", permission: "permission" }
    })
    expect(getUpdateAuthLevel1Fields(action)).toEqual([
      { name: "account", type: "accountLink", value: "account" },
      { name: "permission", type: "bold", value: "permission" }
    ])
  })
})

describe("getUpdateAuthLevel2Fields", () => {
  it("should return the level 2 fields for account", () => {
    const permission = {
      permission: {
        permission: "accountPermission",
        actor: "actor"
      }
    }
    const data = {
      permission: "permission",
      parent: "parent"
    }
    expect(getUpdateAuthLevel2Fields(permission, data, "account")).toEqual([
      { name: "permission", type: "bold", value: "permission" },
      { name: "account", type: "accountLink", value: "actor" },
      { name: "accountPermission", type: "bold", value: "accountPermission" },
      { name: "parent", type: "bold", value: "parent" }
    ])
  })

  it("should return the level 2 fields for key", () => {
    const permission = {
      key: "key1234"
    }
    const data = {
      permission: "permission",
      parent: "parent"
    }
    expect(getUpdateAuthLevel2Fields(permission, data, "key")).toEqual([
      { name: "permission", type: "bold", value: "permission" },
      { name: "key", type: "bold", value: "key1234" },
      { name: "parent", type: "bold", value: "parent" }
    ])
  })

  it("should return the level 2 fields for wait", () => {
    const permission = {
      wait_sec: 12345
    }
    const data = {
      permission: "permission",
      parent: "parent"
    }
    expect(getUpdateAuthLevel2Fields(permission, data, "wait")).toEqual([
      { name: "permission", type: "bold", value: "permission" },
      { name: "wait", type: "plain", value: "3:25:45" },
      { name: "parent", type: "plain", value: "parent" }
    ])
  })
})
