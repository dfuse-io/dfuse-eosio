import * as React from "react"
import { GenericPillComponent, PillRenderingContext } from "../generic-pill.component"
import Box from "../../../../atoms/ui-box/ui-box.component"
import { FormattedText } from "../../../formatted-text/formatted-text"
import { getBuyRamBytesLevel1Fields } from "../pill-template.helpers"

export class BuyRamBytesPillComponent extends GenericPillComponent {
  static requireFields: string[] = ["payer", "receiver", "bytes"]

  static contextForRendering = (): PillRenderingContext => {
    return {
      networks: ["all"],
      validActions: [{ contract: "eosio", action: "buyrambytes" }]
    }
  }

  static i18n() {
    return {
      en: {
        buyrambytes: {
          summary: "<0>{{payer}}</0> bought <1>{{bytes}}</1> of RAM for <2>{{receiver}}</2>"
        }
      },
      zh: {
        buyrambytes: {
          summary: "<0>{{payer}}</0> 买了 <1>{{bytes}}</1> 的RAM 给 <2>{{receiver}}</2>"
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
          fields={getBuyRamBytesLevel1Fields(action)}
          i18nKey="pillTemplates.buyrambytes.summary"
          fontSize={[1]}
        />
      </Box>
    )
  }

  renderLevel2Template = () => {
    return null
  }
}
