import {
  assignHierarchy,
  buildTopLevelHierarchyEntry,
  getChilds,
  buildHierarchyEntry,
  getParentDepths,
  HierarchyData,
  getRankInfo,
  getWebsiteInfo,
  getRankBgColor,
  StakeDetail,
//ultra-andrey-bezrukov --- BLOCK-80 Integrate ultra power into dfuse and remove rex related tables
//  sumCPUStakes,
//  sumNETStakes,
  sumPowerStakes,
  getAccountResources
} from "../account.helpers"
import { Permission } from "../../models/account"
import { getAccountMock } from "../../__mocks__/account.mock"

const permissions: Permission[] = [
  {
    perm_name: "active",
    parent: "owner",
    required_auth: {
      threshold: 4,
      keys: [],
      accounts: [],
      waits: []
    }
  },
  {
    perm_name: "test",
    parent: "owner",
    required_auth: {
      threshold: 4,
      keys: [],
      accounts: [],
      waits: []
    }
  },
  {
    perm_name: "owner",
    parent: "",
    required_auth: {
      threshold: 4,
      keys: [],
      accounts: [],
      waits: []
    }
  }
]

describe("assignHierarchy", () => {
  it("should create the whole hierarchy", () => {
    expect(assignHierarchy(permissions, [])).toEqual([
      {
        depth: 0,
        hasChilds: true,
        lastChild: true,
        parentDepths: [],
        permission: permissions[2]
      },
      {
        depth: 1,
        hasChilds: false,
        lastChild: false,
        parentDepths: [0],
        permission: permissions[0]
      },
      {
        depth: 1,
        hasChilds: false,
        lastChild: true,
        parentDepths: [],
        permission: permissions[1]
      }
    ])
  })
})

describe("buildTopLevelHierarchyEntry", () => {
  it("should return a hierarchy data object with the top level permission", () => {
    expect(buildTopLevelHierarchyEntry(permissions)).toEqual({
      lastChild: true,
      parentDepths: [],
      permission: permissions[2],
      depth: 0,
      hasChilds: true
    } as HierarchyData)
  })
})

describe("getChilds", () => {
  describe("method", () => {
    it("should...", () => {
      expect(getChilds(permissions, permissions[2])).toEqual([permissions[0], permissions[1]])
    })
  })
})

describe("buildHierarchyEntry", () => {
  it("should build a hierarchy entry", () => {
    expect(
      buildHierarchyEntry(permissions, permissions[1], true, {
        lastChild: true,
        parentDepths: [],
        permission: permissions[2],
        depth: 0,
        hasChilds: true
      })
    ).toEqual({
      depth: 1,
      hasChilds: false,
      lastChild: true,
      parentDepths: [],
      permission: permissions[1]
    })
  })
  it("should ", () => {
    expect(
      buildHierarchyEntry(permissions, permissions[0], false, {
        lastChild: true,
        parentDepths: [],
        permission: permissions[2],
        depth: 0,
        hasChilds: true
      })
    ).toEqual({
      depth: 1,
      hasChilds: false,
      lastChild: false,
      parentDepths: [0],
      permission: permissions[0]
    })
  })
})

describe("getParentDepths", () => {
  it("should get the proper parentDepth", () => {
    const parentHierarchy = {
      lastChild: true,
      parentDepths: [],
      permission: permissions[2],
      depth: 0,
      hasChilds: true
    }

    const hierarchyEntry = {
      depth: 1,
      hasChilds: false,
      lastChild: false,
      parentDepths: [0],
      permission: permissions[0]
    }

    expect(getParentDepths(hierarchyEntry, parentHierarchy)).toEqual([0])
  })
})

describe("getRankInfo", () => {
  it("should get the rank info from the vote tally and the account", () => {
    expect(
      getRankInfo(getAccountMock(), [
        {
          producer: "eosfirstcom",
          votePercent: 15,
          decayedVote: 15,
          website: "test.com"
        },
        {
          producer: "eoscanadacom",
          votePercent: 12,
          decayedVote: 12,
          website: "eoscanada.com"
        }
      ])
    ).toEqual({ rank: 2, votePercent: 12, website: "eoscanada.com" })
  })
})

describe("getWebsiteInfo", () => {
  it("should get website info", () => {
    expect(
      getWebsiteInfo(getAccountMock(), [
        {
          producer: "eosfirstcom",
          votePercent: 15,
          decayedVote: 15,
          website: "test.com"
        },
        {
          producer: "eoscanadacom",
          votePercent: 12,
          decayedVote: 12,
          website: "eoscanada.com"
        }
      ])
    ).toEqual({ link: "eoscanada.com", verified: false })
  })
})

describe("getRankBgColor", () => {
  it("should...", () => {
    expect(getRankBgColor({ rank: 2, votePercent: 12, website: "" })).toEqual("#27cfb7")
    expect(getRankBgColor({ rank: 3, votePercent: 12, website: "" })).toEqual("#00c8b1")
    expect(getRankBgColor({ rank: 30, votePercent: 2, website: "" })).toEqual("#d0d0d0")
    expect(getRankBgColor({ rank: 31, votePercent: 2, website: "" })).toEqual("#bfbfbf")

    expect(getRankBgColor({ rank: 35, votePercent: 0.2, website: "" })).toEqual("#bfbfbf")
    expect(getRankBgColor({ rank: 36, votePercent: 0.2, website: "" })).toEqual("#d0d0d0")
  })
})

//ultra-andrey-bezrukov --- BLOCK-80 Integrate ultra power into dfuse and remove rex related tables
//describe("sumCPUStakes", () => {
//  it("should add the cpu stakes", () => {
//    const stakeDetails: StakeDetail[] = [
//      {
//        from: "from",
//        to: "to",
//        cpu_weight: "3.0000 EOS",
//        net_weight: "4.5000 EOS"
//      },
//      {
//        from: "from",
//        to: "to",
//        cpu_weight: "8.0000 EOS",
//        net_weight: "1.5000 EOS"
//      },
//      {
//        from: "foo",
//        to: "target",
//        cpu_weight: "8.0000 EOS",
//        net_weight: "1.5000 EOS"
//      }
//    ]
//
//    expect(sumCPUStakes(stakeDetails, "target")).toEqual(11.0)
//  })
//})

//describe("sumNETStakes", () => {
//  it("should add the net stakes", () => {
//    const stakeDetails: StakeDetail[] = [
//      {
//        from: "from",
//        to: "to",
//        cpu_weight: "3.0000 EOS",
//        net_weight: "4.5000 EOS"
//      },
//      {
//        from: "from",
//        to: "to",
//        cpu_weight: "8.0000 EOS",
//        net_weight: "1.5000 EOS"
//      },
//      {
//        from: "foo",
//        to: "target",
//        cpu_weight: "8.0000 EOS",
//        net_weight: "1.5000 EOS"
//      }
//    ]
//
//    expect(sumNETStakes(stakeDetails, "target")).toEqual(6.0)
//  })
//})

describe("sumPowerStakes", () => {
  it("should add the power stakes", () => {
    const stakeDetails: StakeDetail[] = [
      {
        from: "from",
        to: "to",
        power_weight: "7.5000 EOS"
      },
      {
        from: "from",
        to: "to",
        power_weight: "9.5000 EOS"
      },
      {
        from: "foo",
        to: "target",
        power_weight: "9.5000 EOS"
      }
    ]

    expect(sumPowerStakes(stakeDetails, "target")).toEqual(17.0)
  })
})

describe("getAccountResources", () => {
  it("should...", () => {
    const account = getAccountMock()

    const stakeDetails: StakeDetail[] = [
      {
        from: "from",
        to: "to",
//ultra-andrey-bezrukov --- BLOCK-80 Integrate ultra power into dfuse and remove rex related tables
//        cpu_weight: "3.0000 EOS",
//        net_weight: "4.5000 EOS"
        power_weight: "7.5000 EOS"
      },
      {
        from: "eoscanadacom",
        to: "to",
//        cpu_weight: "8.0000 EOS",
//        net_weight: "1.5000 EOS"
        power_weight: "9.5000 EOS"
      },
      {
        from: "foo",
        to: "eoscanadacom",
//        cpu_weight: "8.0000 EOS",
//        net_weight: "1.5000 EOS"
        power_weight: "9.5000 EOS"
      }
    ]
    expect(getAccountResources(account, stakeDetails)).toEqual({
      availableFunds: 16,
//      cpu: { selfStaked: 1.3, stakedFromOthers: 4, stakedToOthers: 11, stakedTotal: 16.3 },
//      net: {
//        selfStaked: 2.2,
//        stakedFromOthers: 2,
//        stakedToOthers: 5.999999999999999,
//        stakedTotal: 10.2
//      },
      power: { selfStaked: 3.5, stakedFromOthers: 6, stakedToOthers: 17, stakedTotal: 26.5 },
//      rexFunds: 0,
//      rexLiquid: 0,
      pendingRefund: 0,
      stakes: [
//        { cpu_weight: "3.0000 EOS", from: "from", net_weight: "4.5000 EOS", to: "to" },
//        { cpu_weight: "8.0000 EOS", from: "eoscanadacom", net_weight: "1.5000 EOS", to: "to" }
        { from: "from", power_weight: "7.5000 EOS", to: "to" },
        { from: "eoscanadacom", power_weight: "9.5000 EOS", to: "to" }
      ],
      totalOwnerShip: 36.5,
//      unit: "EOS"
    })
  })
})
