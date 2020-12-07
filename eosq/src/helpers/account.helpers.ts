import { extractValueWithUnits } from "@dfuse/explorer"
import { Account, Permission } from "../models/account"
import { Vote } from "../models/vote"
import { DonutData } from "../atoms/pie-chart/donut-chart"
import { t } from "i18next"
import { theme } from "../theme"
import numeral from "numeral"
import { Config } from "../models/config"

export interface HierarchyData {
  depth: number
  lastChild: boolean
  parentDepths: number[]
  permission: Permission
  index?: number
  hasChilds: boolean
}

export interface StakeDetail {
  from: string
  to: string
  cpu_weight: string
  net_weight: string
}

export function assignHierarchy(
  permissions: Permission[],
  currentHierarchy: HierarchyData[],
  parentHierarchy?: HierarchyData
): HierarchyData[] {
  if (currentHierarchy.length === 0 && !parentHierarchy) {
    const hierarchyEntry = buildTopLevelHierarchyEntry(permissions)

    currentHierarchy = assignHierarchy(permissions, [hierarchyEntry], hierarchyEntry)
  } else if (parentHierarchy) {
    const childPermissions = getChilds(permissions, parentHierarchy.permission)

    if (childPermissions.length === 0) {
      return currentHierarchy
    }
    childPermissions.forEach((permission: Permission, index: number) => {
      const lastChild = index === childPermissions.length - 1
      const hierarchyEntry = buildHierarchyEntry(
        permissions,
        permission,
        lastChild,
        parentHierarchy
      )
      currentHierarchy = [...currentHierarchy, hierarchyEntry]
      currentHierarchy = assignHierarchy(permissions, currentHierarchy, hierarchyEntry)
    })
  } else {
    return []
  }

  return [...currentHierarchy]
}

export function buildTopLevelHierarchyEntry(permissions: Permission[]) {
  const hierarchyDataEntry = {
    lastChild: true,
    parentDepths: [],
    permission: permissions.find((permission: Permission) => !permission.parent),
    depth: 0,
    hasChilds: false
  } as HierarchyData
  hierarchyDataEntry.hasChilds = getChilds(permissions, hierarchyDataEntry.permission).length > 0
  return hierarchyDataEntry
}

export function getChilds(permissions: Permission[], parentPermission: Permission) {
  return permissions.filter(
    (permission: Permission) => permission.parent === parentPermission.perm_name
  )
}

export function buildHierarchyEntry(
  permissions: Permission[],
  permission: Permission,
  lastChild: boolean,
  parentHierarchy: HierarchyData
) {
  const hierarchyEntry = {
    lastChild,
    parentDepths: [],
    permission,
    depth: parentHierarchy.depth + 1,
    hasChilds: false
  } as HierarchyData
  hierarchyEntry.parentDepths = getParentDepths(parentHierarchy, hierarchyEntry)
  hierarchyEntry.hasChilds = getChilds(permissions, hierarchyEntry.permission).length > 0

  return hierarchyEntry
}

export function getParentDepths(
  parentHierarchyEntry: HierarchyData,
  hierarchyEntry: HierarchyData
) {
  const parentDepths = Object.assign([], parentHierarchyEntry.parentDepths)
  if (!hierarchyEntry.lastChild) {
    parentDepths.push(parentHierarchyEntry.depth)
  }
  return parentDepths
}

// ************************************************************************************* //

export function getRankInfo(
  account: Account,
  votes: Vote[]
): { rank: number; votePercent: number; website: string } {
  let rank = 0
  let votePercent = 0
  let website = ""
  votes.forEach((vote, index) => {
    if (vote.producer === account.account_name) {
      rank = index + 1
      votePercent = vote.votePercent
      website = vote.website
    }
  })
  return { rank, votePercent, website }
}

export function getWebsiteInfo(account: Account, votes: Vote[]) {
  let link: string
  let verified = false

  if (!account.account_verifications) {
    link = getRankInfo(account, votes).website
  } else if (
    account.account_verifications.website &&
    account.account_verifications.website.handle !== ""
  ) {
    link = account.account_verifications.website.handle
    verified = account.account_verifications.website.verified
  } else {
    link = getRankInfo(account, votes).website
  }

  return { link, verified }
}

export function getRankStatus(rankInfo: { rank: number; votePercent: number; website: string }) {
  const { rank } = rankInfo
  if (rank < 22) {
    return t("vote.list.legend.active")
  }

  if (rankInfo.votePercent < 0.5) {
    return t("vote.list.legend.runnerUps")
  }

  return t("vote.list.legend.standBy")
}

export function getRankBgColor(rankInfo: { rank: number; votePercent: number; website?: string }) {
  const { rank } = rankInfo
  if (rank < 22) {
    return rank % 2 ? "#00c8b1" : "#27cfb7"
  }

  // The logic is not correct, the Stand-By should be based on the condition Daily EOS Reward > 100 EOS
  // if (rankInfo.votePercent >= 0.5) {
  //   return rank % 2 ? "#fbac53" : "#ffb866"
  // }

  // Let's all by runner-ups for now
  return rank % 2 ? "#bfbfbf" : "#d0d0d0"
}

export function sumCPUStakes(stakes: StakeDetail[], accountName: string): number {
  return stakes.reduce((a: number, b: StakeDetail) => {
    if (b.to !== accountName) {
      a += parseFloat(b.cpu_weight.split(" ")[0])
    }
    return a
  }, 0.0)
}

export function sumNETStakes(stakes: StakeDetail[], accountName: string): number {
  return stakes.reduce((a: number, b: StakeDetail) => {
    if (b.to !== accountName) {
      a += parseFloat(b.net_weight.split(" ")[0])
    }
    return a
  }, 0.0)
}

export interface AccountResources {
  cpu: {
    stakedTotal: number
    stakedFromOthers: number
    selfStaked: number
    stakedToOthers: number
  }
  net: {
    stakedTotal: number
    stakedFromOthers: number
    selfStaked: number
    stakedToOthers: number
  }
  rexLiquid: number
  rexFunds: number
  availableFunds: number
  pendingRefund: number
  totalOwnerShip: number
  stakes: StakeDetail[]
  unit: string
}

export function getAccountResources(account: Account, stakes: StakeDetail[]): AccountResources {
  const totalResources = account.total_resources
  const selfDelegated = account.self_delegated_bandwidth
  const refundRequest = account.refund_request
  const rexTokens = account.rex_balance
    ? account.rex_balance.vote_stake
    : `0.0000 ${Config.chain_core_symbol_code}`
  const rexFunds = account.rex_funds
    ? account.rex_funds.balance
    : `0.0000 ${Config.chain_core_symbol_code}`
  const rexCpuLoans = account.cpu_loans ? account.cpu_loans : 0
  const rexNetLoans = account.net_loans ? account.net_loans : 0
  const unit =
    extractValueWithUnits(totalResources.cpu_weight)[1] || ` ${Config.chain_core_symbol_code}`
  let stakedCpu = parseFloat(extractValueWithUnits(totalResources.cpu_weight)[0])
  const availableFunds = parseFloat(extractValueWithUnits(account.core_liquid_balance)[0])
  const selfStakedCpu = parseFloat(extractValueWithUnits(selfDelegated.cpu_weight)[0])
  const rexStake = parseFloat(extractValueWithUnits(rexTokens)[0])
  const rexFundsAmount = parseFloat(extractValueWithUnits(rexFunds)[0])
  const stakedCpuFromOthers = stakedCpu - selfStakedCpu

  if (stakes.length > 0) {
    stakedCpu += sumCPUStakes(stakes, account.account_name)
  }

  let stakedNetwork = parseFloat(extractValueWithUnits(totalResources.net_weight)[0])
  const selfStakedNet = parseFloat(extractValueWithUnits(selfDelegated.net_weight)[0])
  const stakedNetworkFromOthers = stakedNetwork - selfStakedNet
  if (stakes.length > 0) {
    stakedNetwork += sumNETStakes(stakes, account.account_name)
  }

  let pendingRefund = 0.0
  if (refundRequest) {
    pendingRefund = parseFloat(extractValueWithUnits(refundRequest.net_amount)[0])
    pendingRefund += parseFloat(extractValueWithUnits(refundRequest.cpu_amount)[0])
  }

  const totalOwnerShip =
    stakedCpu +
    stakedNetwork +
    rexStake +
    rexFundsAmount +
    rexCpuLoans +
    rexNetLoans +
    pendingRefund +
    availableFunds -
    stakedNetworkFromOthers -
    stakedCpuFromOthers

  return {
    net: {
      stakedTotal: stakedNetwork,
      stakedFromOthers: stakedNetworkFromOthers,
      selfStaked: selfStakedNet,
      stakedToOthers: stakedNetwork - selfStakedNet - stakedNetworkFromOthers
    },
    cpu: {
      stakedTotal: stakedCpu,
      stakedFromOthers: stakedCpuFromOthers,
      selfStaked: selfStakedCpu,
      stakedToOthers: stakedCpu - selfStakedCpu - stakedCpuFromOthers
    },
    rexLiquid: rexStake,
    rexFunds: rexNetLoans + rexCpuLoans + rexFundsAmount,
    availableFunds,
    pendingRefund,
    totalOwnerShip,
    stakes: stakes.filter((stake: StakeDetail) => stake.to !== account.account_name),
    unit
  }
}

export interface PieChartParams {
  pieChartData: DonutData[]
  pieChartCenter: string
  pieChartColorsForPie: string[]
  pieChartColors: string[]
  pieChartDataForPie: DonutData[]
}

export function getPieChartParams(
  accountResources: AccountResources,
  wrapperRenderer: (accountResources: AccountResources, type: string, value: number) => JSX.Element
): PieChartParams {
  const pieChartData: DonutData[] = [
    {
      label: t("account.pie_chart.labels.staked_cpu"),
      value: accountResources.cpu.stakedTotal,
      renderWrapper: (value: any) => wrapperRenderer(accountResources, "cpu", value)
    },
    {
      label: t("account.pie_chart.labels.staked_network"),
      value: accountResources.net.stakedTotal,
      renderWrapper: (value: any) => wrapperRenderer(accountResources, "net", value)
    },
    {
      label: t("account.pie_chart.labels.rex"),
      value: accountResources.rexLiquid,
      renderWrapper: (value: any) => wrapperRenderer(accountResources, "REX", value)
    },
    {
      label: t("account.pie_chart.labels.rex_funds"),
      value: accountResources.rexFunds,
      renderWrapper: (value: any) => wrapperRenderer(accountResources, "REX_FUNDS", value)
    },
    {
      label: t("account.pie_chart.labels.pending_refund"),
      value: accountResources.pendingRefund,
      renderWrapper: (value: any) => wrapperRenderer(accountResources, "refund", value)
    },
    {
      label: t("account.pie_chart.labels.available_funds"),
      value: accountResources.availableFunds,
      renderWrapper: (value: any) => wrapperRenderer(accountResources, "available_funds", value)
    }
  ]

  const pieChartColors = [
    theme.colors.stakeCPU,
    theme.colors.stakeNetwork,
    theme.colors.stakeREX,
    theme.colors.stakeREXFunds,
    theme.colors.secondHighlight,
    theme.colors.ternary
  ]

  const pieChartCenter =
    // eslint-disable-next-line prefer-template
    numeral(accountResources.totalOwnerShip).format("0,0") + " " + accountResources.unit

  let pieChartDataForPie = pieChartData
  let pieChartColorsForPie = pieChartColors
  if (accountResources.totalOwnerShip === 0.0) {
    pieChartColorsForPie = [theme.colors.text]
    pieChartDataForPie = [{ label: "", value: 1 }]
  }

  return {
    pieChartData,
    pieChartCenter,
    pieChartColors,
    pieChartColorsForPie,
    pieChartDataForPie
  }
}
