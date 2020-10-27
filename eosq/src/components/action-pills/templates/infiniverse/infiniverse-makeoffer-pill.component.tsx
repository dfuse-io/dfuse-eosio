import * as React from "react"
import { GenericPillComponent, PillRenderingContext } from "../generic-pill.component"
import { Box, PillLogoProps } from "@dfuse/explorer"
import { FormattedText } from "../../../formatted-text/formatted-text"
import { ExternalTextLink } from "../../../../atoms/text/text.component"

import { getInfiniverseMakeOfferLevel1Fields } from "../pill-template.helpers"

export class MakeOfferPillComponent extends GenericPillComponent {
  get logoParams(): PillLogoProps | undefined {
    return {
      path: "/images/pill-logos/logo-contract-infiniverse-01.svg",
      website: "https://infiniverse.net"
    }
  }

  static requireFields: string[] = ["buyer", "land_id", "price"]

  static contextForRendering = (): PillRenderingContext => {
    return {
      networks: ["eos-mainnet"],

      validActions: [{ contract: "infiniverse1", action: "makeoffer" }]
    }
  }

  static i18n() {
    return {
      en: {
        infiniversemakeoffer: {
          summary: "<0>{{buyer}}</0> offers <1>{{quantity}}</1> for Land <2>{{land_id}}</2>"
        }
      },
      zh: {
        infiniversemakeoffer: {
          summary: "<0>{{buyer}}</0> 为 Land <2>{{land_id}}</2> 报价 <1>{{quantity}}</1>"
        }
      }
    }
  }

  renderContent = () => {
    const { action } = this.props

    return (
      <Box fontSize={[1]} mx={[2]} minWidth="10px" minHeight="26px" alignItems="center">
        <FormattedText
          fields={getInfiniverseMakeOfferLevel1Fields(action)}
          i18nKey="pillTemplates.infiniversemakeoffer.summary"
          fontSize={[1]}
        />
      </Box>
    )
  }

  renderLevel2Template = () => {
    return (
      <Box fontSize={[1]} mx={[2]} minWidth="10px" minHeight="26px" alignItems="center">
        <ExternalTextLink to="https://infiniverse.net">https://infiniverse.net</ExternalTextLink>
      </Box>
    )
  }
}
