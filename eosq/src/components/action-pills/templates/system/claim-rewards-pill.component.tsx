import * as React from "react"
import { GenericPillComponent, PillRenderingContext } from "../generic-pill.component"
import { Box } from "@dfuse/explorer"
import { getClaimRewardsLevel1Fields, getClaimRewardsLevel2Fields } from "../pill-template.helpers"
import { FormattedText } from "../../../formatted-text/formatted-text"

export class ClaimRewardsPillComponent extends GenericPillComponent {
  static requireFields: string[] = ["owner"]

  static contextForRendering = (): PillRenderingContext => {
    return {
      networks: ["all"],
      validActions: [{ contract: "eosio", action: "claimrewards" }]
    }
  }

  static i18n() {
    return {
      en: {
        claimrewards: {
          summary: "<0>{{account}}</0> claimed <1>{{amountEOS}}</1> rewards",
          detail:
            "<0>{{account}}</0> claimed <1>{{amountbEOS}}</1> for Block Rewards and <2>{{amountvEOS}}</2> for Vote Rewards "
        }
      },
      zh: {
        claimrewards: {
          summary: "<0>{{account}}</0> 认领了 <1>{{amountEOS}}</1> 的奖励",
          detail:
            "<0>{{account}}</0> 认领了 <1>{{amountbEOS}}</1> 的区块奖励 和 <2>{{amountvEOS}}</2> 的投票奖励"
        }
      }
    }
  }

  renderContent = () => {
    const { action } = this.props

    if (this.hasInlineTraces()) {
      return (
        <Box fontSize={[1]} mx={[2]} minWidth="10px" minHeight="26px" alignItems="center">
          <FormattedText
            fields={getClaimRewardsLevel1Fields(action, this.props.traceInfo)}
            i18nKey="pillTemplates.claimrewards.summary"
            fontSize={[1]}
          />
        </Box>
      )
    }

    return this.renderDefaultContent()
  }

  renderLevel2Template = () => {
    const { action } = this.props
    if (this.hasInlineTraces()) {
      return (
        <Box fontSize={[1]} mx={[2]} minWidth="10px" minHeight="26px" alignItems="center">
          <FormattedText
            fields={getClaimRewardsLevel2Fields(action, this.props.traceInfo)}
            i18nKey="pillTemplates.claimrewards.detail"
            fontSize={[1]}
          />
        </Box>
      )
    }

    return null
  }
}
