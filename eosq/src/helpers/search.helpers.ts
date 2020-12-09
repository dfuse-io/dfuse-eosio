import { DropDownOption } from "../atoms/ui-dropdown/ui-dropdown.component"
import { TokenInfo, getTokenInfosForNetwork } from "./airdrops-list"
import { t } from "i18next"
import { Config } from "../models/config"

export function getSearchTransfersOptions(accountName: string): DropDownOption[] {
  const tokenInfos = getTokenInfosForNetwork(Config.network_id)
  if (tokenInfos.length === 0) {
    return []
  }

  const popularTokensCondition = tokenInfos
    .map((airdrop: TokenInfo) => {
      return `account:${airdrop.account}`
    })
    .join(" OR ")
  return [
    {
      label: "...",
      value: `(auth:${accountName} OR receiver:${accountName})`,
    },
    {
      label: t("transactionSearch.dropdowns.tokens.allTokens"),
      value: `action:transfer (data.to:${accountName} OR data.from:${accountName})`,
    },
    {
      label: t("transactionSearch.dropdowns.tokens.eos"),
      value: `action:transfer account:eosio.token (data.to:${accountName} OR data.from:${accountName})`,
    },
    {
      label: t("transactionSearch.dropdowns.tokens.popularTokens"),
      value: `action:transfer (data.to:${accountName} OR data.from:${accountName}) (${popularTokensCondition})`,
    },
  ]
}

export function getSearchSystemOptions(accountName: string): DropDownOption[] {
  return [
    {
      label: "...",
      value: `(auth:${accountName} OR receiver:${accountName})`,
    },
    {
      label: t("transactionSearch.dropdowns.system.claimRewards"),
      value: `action:claimrewards account:eosio data.owner:${accountName}`,
    },
    {
      label: t("transactionSearch.dropdowns.system.delegateBandwidth"),
      value: `action:delegatebw account:eosio (data.from:${accountName} OR data.receiver:${accountName})`,
    },
    {
      label: t("transactionSearch.dropdowns.system.undelegateBandwidth"),
      value: `action:undelegatebw account:eosio (data.from:${accountName} OR data.receiver:${accountName})`,
    },
    {
      label: t("transactionSearch.dropdowns.system.regProducer"),
      value: `action:regproducer account:eosio data.producer:${accountName}`,
    },
    {
      label: t("transactionSearch.dropdowns.system.setCode"),
      value: `action:setcode account:eosio data.account:${accountName}`,
    },
  ]
}
