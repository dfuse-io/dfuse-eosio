import { PillHeaderParams } from "../models/pill-templates"
import { Action } from "@dfuse/client"

export interface LabelValue {
  label: string
  value: string
}

export function getMemoText(action: Action<any>): string {
  return action.data.memo ? action.data.memo : null
}

const PILL_HEADER_PARAMS_MAP: { [key: string]: PillHeaderParams } = {
  eosio: { color: "#002343", text: "Sy", hoverTitle: "eosio" },
  "eosio.forum": { color: "#5449ba", text: "Fo", hoverTitle: "eosio.forum" },
  "eosio.token": { color: "#5449ba", text: "Tk", hoverTitle: "eosio.token" }
}

export function getHeaderParams(account: string, receiver: string): PillHeaderParams {
  const headerInfo = PILL_HEADER_PARAMS_MAP[account]
  const genericInfo = {
    color: "traceAccountGenericBackground",
    text: receiver,
    hoverTitle: receiver
  }

  return headerInfo || genericInfo
}

export function getHeaderAndTitle(
  action: Action<any>,
  receiver: string
): { header: PillHeaderParams; title: string } {
  const header = { ...getHeaderParams(action.account, receiver) }
  let title = action.name
  if (action.account !== receiver) {
    title = ""
    header.text = `notification:${receiver}`
  }

  return { header, title }
}
