import * as React from "react"
import { GenericPillComponent, PillRenderingContext } from "../generic-pill.component"
import Box from "../../../../atoms/ui-box/ui-box.component"

import { FormattedText } from "../../../formatted-text/formatted-text"
import { getBuyRamLevel1Fields } from "../pill-template.helpers"

export class BuyRamPillComponent extends GenericPillComponent {
  static requireFields: string[] = ["payer", "receiver"]

  static contextForRendering = (): PillRenderingContext => {
    return {
      networks: ["all"],
      validActions: [{ contract: "eosio", action: "buyram" }]
    }
  }

  static i18n() {
    return {
      en: {
        buyram: {
          summary: "<0>{{payer}}</0> spent <1>{{amountEOS}}</1> on RAM for <2>{{receiver}}</2>"
        }
      },
      zh: {
        buyram: {
          summary: "<0>{{payer}}</0> 花费 <1>{{amountEOS}}</1> 在RAM上 给 <2>{{receiver}}</2>"
        }
      }
    }
  }

  isReceiveTransfer() {
    return (
      this.props.pageContext &&
      this.props.pageContext.accountName === this.props.action.data.receiver
    )
  }

  renderContent = () => {
    const { action } = this.props

    return (
      <Box fontSize={[1]} mx={[2]} minWidth="10px" minHeight="26px" alignItems="center">
        <FormattedText
          fields={getBuyRamLevel1Fields(action)}
          i18nKey="pillTemplates.buyram.summary"
          fontSize={[1]}
        />
      </Box>
    )
  }

  renderLevel2Template = () => {
    return null
  }
}
