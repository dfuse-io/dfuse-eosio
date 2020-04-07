import {
  extractValueWithUnits,
  formatBytes,
  getAmount,
  hex2sha256,
  secondsToTime
} from "../../../helpers/formatters"
import { ActionTrace, Action } from "@dfuse/client"
import { sha256 } from "js-sha256"
import { TraceInfo } from "../../../models/pill-templates"
import { Config } from "../../../models/config"

export function getClaimAmounts(traceInfo?: TraceInfo) {
  const inlineTraces = (traceInfo ? traceInfo.inline_traces : []) || []

  const vpayAction = inlineTraces.find((trace: ActionTrace<any>) => {
    return trace.act.data.from === "eosio.vpay"
  })

  const bpayAction = inlineTraces.find((trace: ActionTrace<any>) => {
    return trace.act.data.from === "eosio.bpay"
  })

  const ppayAction = inlineTraces.find((trace: ActionTrace<any>) => {
    return trace.act.data.from === "eosio.ppay"
  })

  const unit = Config.price_ticker_name

  const vpay = vpayAction ? getAmount(vpayAction.act.data.quantity) : 0

  const bpay = bpayAction ? getAmount(bpayAction.act.data.quantity) : 0
  const ppay = ppayAction ? getAmount(ppayAction.act.data.quantity) : 0

  const total = `${(vpay + bpay + ppay).toFixed(4)} ${unit}`
  return [total, bpay, vpay, ppay]
}

export function getNewAccountInTraces(traceInfo?: TraceInfo): string | undefined {
  const inlineTraces = (traceInfo ? traceInfo.inline_traces : []) || []

  const newAccountAction = inlineTraces.find((trace: ActionTrace<any>) => {
    return trace.act.name === "newaccount" && trace.act.account === "eosio"
  })

  if (newAccountAction) {
    return newAccountAction.act.data.name
  }
  return undefined
}

export function getPixeosClaimAmounts(traceInfo?: TraceInfo) {
  const inlineTraces = (traceInfo ? traceInfo.inline_traces : []) || []

  const total = inlineTraces.find((trace: ActionTrace<any>) => {
    return trace.act.data.from === "pixeos1paint"
  })

  if (total) {
    return total.act.data.quantity
  }

  return "unknown quantity"
}

export function getResolveBetAmounts(traceInfo?: TraceInfo) {
  const inlineTraces = (traceInfo ? traceInfo.inline_traces : []) || []

  const transferAction = inlineTraces.find((trace: ActionTrace<any>) => {
    return trace.act.name === "transfer"
  })

  const [EOSAmount, unit] = transferAction
    ? extractValueWithUnits(transferAction.act.data.quantity)
    : [0, " - "]
  const receiver = transferAction ? transferAction.act.data.to : ""
  return [EOSAmount, unit, receiver]
}

export function getBlobUrlFromPayload(payload: string | Uint8Array, downloadUrl: string = "") {
  if (downloadUrl.length > 0) {
    URL.revokeObjectURL(downloadUrl)
  }

  downloadUrl = URL.createObjectURL(
    new Blob([payload], {
      type: "text/plain;charset=utf-8"
    })
  )
  return [sha256(payload), downloadUrl]
}

export function getRefundTransfer(traceInfo?: TraceInfo): ActionTrace<any> | undefined {
  const inlineTraces = (traceInfo ? traceInfo.inline_traces : []) || []

  const transferAction = inlineTraces.find((trace: ActionTrace<any>) => {
    return trace.act.name === "transfer"
  })
  return transferAction
}

// ******************************************************************************************* //

export function getBetReceiptLevel1Fields(action: Action<any>) {
  return [
    {
      type: "accountLink",
      value: action.data.bettor,
      name: "account"
    },
    { type: "bold", value: action.data.payout, name: "EOSAmount" },
    { type: "bold", value: action.data.random_roll, name: "roll" }
  ]
}

export function getBuyRamBytesLevel1Fields(action: Action<any>) {
  return [
    {
      name: "payer",
      value: action.data.payer,
      type: "accountLink"
    },
    { name: "bytes", value: formatBytes(action.data.bytes), type: "bold" },
    {
      name: "receiver",
      value: action.data.receiver,
      type: "accountLink"
    }
  ]
}

export function getBuyRamLevel1Fields(action: Action<any>) {
  return [
    { name: "payer", type: "accountLink", value: action.data.payer },
    { name: "amountEOS", type: "bold", value: action.data.quantity || action.data.quant },
    { name: "receiver", type: "accountLink", value: action.data.receiver }
  ]
}

export function getClaimRewardsLevel1Fields(action: Action<any>, traceInfo?: TraceInfo) {
  return [
    {
      name: "account",
      value: action.data.owner,
      type: "accountLink"
    },
    { name: "amountEOS", value: getClaimAmounts(traceInfo)[0], type: "bold" }
  ]
}

export function getPixeosClaimLevel1Fields(action: Action<any>, traceInfo?: TraceInfo) {
  return [
    {
      name: "account",
      value: action.data.owner,
      type: "accountLink"
    },
    { name: "amountEOS", value: getPixeosClaimAmounts(traceInfo), type: "bold" }
  ]
}

export function getKarmaClaimLevel1Fields(action: Action<any>) {
  return [
    {
      name: "account",
      value: action.data.owner,
      type: "accountLink"
    }
  ]
}

export function getDfuseEventLevel1Fields(action: Action<any>) {
  return [
    {
      name: "indexedField",
      value: "IndexedField",
      type: "bold"
    },
    {
      name: "fields",
      value: action.data.data
        .split("=")
        .join(" = ")
        .split("&")
        .join(", "),
      type: "plain"
    }
  ]
}

export function getCarbonIssueLevel1Fields(action: Action<any>) {
  return [
    {
      name: "amountCUSD",
      value: action.data.quantity,
      type: "bold"
    },
    {
      name: "to",
      value: action.data.to,
      type: "accountLink"
    }
  ]
}

export function getCarbonBurnLevel1Fields(action: Action<any>) {
  return [
    {
      name: "amountCUSD",
      value: action.data.quantity,
      type: "bold"
    },
    {
      name: "from",
      value: action.data.from,
      type: "accountLink"
    }
  ]
}

export function getKarmaPowerdownLevel1Fields(action: Action<any>) {
  return [
    {
      name: "account",
      value: action.data.owner,
      type: "accountLink"
    },
    { name: "amountKarma", value: action.data.quantity, type: "bold" }
  ]
}

export function getKarmaClaimPostLevel1Fields(action: Action<any>) {
  return [
    {
      name: "account",
      value: action.data.author,
      type: "accountLink"
    }
  ]
}

export function getPixeosAddToClaimLevel1Fields(action: Action<any>) {
  return [
    {
      name: "account",
      value: action.data.user,
      type: "accountLink"
    },
    {
      name: "amountEOS",
      value: `${(action.data.addbalance / 10000000000).toFixed(10)} EOS`,
      type: "bold"
    }
  ]
}

export function getKarmaPowerUpLevel1Fields(action: Action<any>) {
  return [
    {
      name: "account",
      value: action.data.owner,
      type: "accountLink"
    },
    { name: "amountKarma", value: action.data.quantity, type: "bold" }
  ]
}

export function getClaimRewardsLevel2Fields(action: Action<any>, traceInfo?: TraceInfo) {
  return [
    {
      name: "account",
      value: action.data.owner,
      type: "accountLink"
    },
    {
      name: "amountbEOS",
      value: `${getClaimAmounts(traceInfo)[1]} ${Config.price_ticker_name}`,
      type: "bold"
    },
    {
      name: "amountvEOS",
      value: `${getClaimAmounts(traceInfo)[2]} ${Config.price_ticker_name}`,
      type: "bold"
    }
  ]
}

export function getDelegatebwLevel1Fields(action: Action<any>) {
  return [
    {
      name: "from",
      type: "accountLink",
      value: action.data.from
    },
    { name: "amountCPU", type: "bold", value: action.data.stake_cpu_quantity },
    { name: "amountNET", type: "bold", value: action.data.stake_net_quantity },
    {
      name: "to",
      type: "accountLink",
      value: action.data.receiver
    }
  ]
}

export function getDelegatebwLevel2Fields(action: Action<any>) {
  return [
    { name: "amountCPU", type: "bold", value: action.data.stake_cpu_quantity },
    { name: "amountNET", type: "bold", value: action.data.stake_net_quantity }
  ]
}

export function getLinkAuthLevel1Fields(action: Action<any>) {
  return [
    { name: "account", type: "accountLink", value: action.data.account },
    { name: "requirement", type: "bold", value: action.data.requirement },
    { name: "type", type: "bold", value: action.data.type },
    { name: "code", type: "accountLink", value: action.data.code }
  ]
}

export function getLinkAuthLevel2Fields(action: Action<any>) {
  return [
    { name: "requirement", type: "bold", value: action.data.requirement },
    { name: "type", type: "bold", value: action.data.type },
    { name: "code", type: "bold", value: action.data.code }
  ]
}

export function getNewAccountLevel1Fields(action: Action<any>) {
  return [
    { name: "creator", type: "accountLink", value: action.data.creator },
    { name: "name", type: "accountLink", value: action.data.name }
  ]
}

export function getNewAccountLevel2Fields(permission: any, parentName: string, type: string) {
  if (type === "account") {
    return [
      { name: "permission", type: "bold", value: parentName },
      { name: "account", type: "accountLink", value: permission.permission.actor },
      { name: "accountPermission", type: "bold", value: permission.permission.permission }
    ]
  }

  if (type === "key") {
    return [
      { name: "permission", type: "bold", value: parentName },
      { name: "key", type: "plain", value: permission.key }
    ]
  }

  if (type === "wait") {
    return [
      { name: "permission", type: "bold", value: parentName },
      { name: "wait", type: "plain", value: permission.key }
    ]
  }

  return []
}

export function getRefundLevel1Fields(action: Action<any>, traceInfo?: TraceInfo) {
  const transferAction = getRefundTransfer(traceInfo)

  return [
    {
      name: "refundAmount",
      type: "bold",
      value: transferAction ? transferAction.act.data.quantity : "-"
    },
    { name: "owner", type: "accountLink", value: action.data.owner }
  ]
}

export function getResolveBetLevel1Fields(action: Action<any>, traceInfo?: TraceInfo) {
  const traceData = getResolveBetAmounts(traceInfo)
  return [
    { name: "account", type: "accountLink", value: traceData[2] },
    { name: "EOSAmount", type: "bold", value: `${traceData[0]} ${traceData[1]}` },
    { name: "betId", type: "bold", value: action.data.bet_id }
  ]
}

export function getUndelegatebwLevel1Fields(action: Action<any>) {
  return [
    { name: "from", type: "accountLink", value: action.data.from },
    { name: "amountCPU", type: "bold", value: action.data.unstake_cpu_quantity },
    { name: "amountNET", type: "bold", value: action.data.unstake_net_quantity }
  ]
}

export function getUndelegatebwLevel2Fields(action: Action<any>) {
  const cpuAmount = getAmount(action.data.unstake_cpu_quantity)
  const netAmount = getAmount(action.data.unstake_net_quantity)
  const unit = action.data.unstake_cpu_quantity.split(" ")[1]
  const total = `${(cpuAmount + netAmount).toFixed(4)} ${unit}`

  return [{ name: "total", type: "bold", value: total }]
}

export function getUpdateAuthLevel1Fields(action: Action<any>) {
  return [
    { name: "account", type: "accountLink", value: action.data.account },
    { name: "permission", type: "bold", value: action.data.permission }
  ]
}

export function getUpdateAuthLevel2Fields(permission: any, data: any, type: string) {
  if (type === "account") {
    return [
      { name: "permission", type: "bold", value: data.permission },
      { name: "account", type: "accountLink", value: permission.permission.actor },
      { name: "accountPermission", type: "bold", value: permission.permission.permission },
      { name: "parent", type: "bold", value: data.parent }
    ]
  }

  if (type === "key") {
    return [
      { name: "permission", type: "bold", value: data.permission },
      { name: "key", type: "bold", value: permission.key },
      { name: "parent", type: "bold", value: data.parent }
    ]
  }

  if (type === "wait") {
    return [
      { name: "permission", type: "bold", value: data.permission },
      { name: "wait", type: "plain", value: secondsToTime(permission.wait_sec) },
      { name: "parent", type: "plain", value: data.parent }
    ]
  }

  return []
}

export function getInfiniverseMakeOfferLevel1Fields(action: Action<any>) {
  return [
    { name: "buyer", type: "accountLink", value: action.data.buyer },
    { name: "quantity", type: "bold", value: action.data.price },
    { name: "land_id", type: "bold", value: action.data.land_id }
  ]
}

export function getInfiniverseMoveLandLevel1Fields(action: Action<any>) {
  const authorizations = action.authorization || []

  return [{ name: "authorizer", type: "accountLink", value: authorizations[0].actor }]
}

export function getInfiniversePersistPolyLevel1Fields(action: Action<any>) {
  return [
    { name: "landTitle", type: "bold", value: "Land ID:" },
    { name: "land_id", type: "plain", value: action.data.land_id },
    { name: "polyTitle", type: "bold", value: "Poly ID:" },
    { name: "poly_id", type: "plain", value: action.data.poly_id }
  ]
}

export function getInfiniverseRegisterlandLevel1Fields(action: Action<any>) {
  return [{ name: "owner", type: "accountLink", value: action.data.owner }]
}

export function getInfiniverseSetLandPriceLevel1Fields(action: Action<any>) {
  const authorizations = action.authorization || []

  return [
    { name: "authorizer", type: "accountLink", value: authorizations[0].actor },
    { name: "quantity", type: "bold", value: action.data.price },
    { name: "land_id", type: "bold", value: action.data.land_id }
  ]
}

export function getInfiniverseUpdatePersistLevel1Fields(action: Action<any>) {
  return [
    { name: "landTitle", type: "bold", value: "Land ID:" },
    { name: "land_id", type: "plain", value: action.data.land_id },
    { name: "polyTitle", type: "bold", value: "Persistent ID:" },
    { name: "poly_id", type: "plain", value: action.data.persistent_id }
  ]
}

export function getInfiniverseDeletePersistLevel1Fields(action: Action<any>) {
  const authorizations = action.authorization || []

  return [{ name: "authorizer", type: "accountLink", value: authorizations[0].actor }]
}

export function getNewAccountFromNameServiceFields(accountName: string) {
  return [
    { name: "account", type: "accountLink", value: accountName },
    { name: "link", type: "link", value: "https://eosnameservice.io" }
  ]
}

export function truncateJsonString(dataString: string, cutOff: number, croppedCallback: () => any) {
  return JSON.parse(dataString, (key, value) => {
    if (typeof value === "string" && value.length > cutOff) {
      croppedCallback()
      if (key === "code") {
        return `${value.substring(0, cutOff)}...  SHA256[${hex2sha256(value)}]`
      }

      return `${value.substring(0, cutOff)}... [+${value.length - cutOff}]`
    }

    return value
  })
}

export function truncateStringPlus(str: string, cutOff: number) {
  if (str.length > cutOff) {
    return `${str.substring(0, cutOff)}... [+${str.length - cutOff}]`
  }

  return str
}
