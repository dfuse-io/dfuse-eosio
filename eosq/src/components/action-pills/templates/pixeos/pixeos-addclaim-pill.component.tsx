import * as React from "react"
import { GenericPillComponent, PillRenderingContext } from "../generic-pill.component"
import { Box, PillLogoProps } from "@dfuse/explorer"
import { getPixeosAddToClaimLevel1Fields } from "../pill-template.helpers"
import { FormattedText } from "../../../formatted-text/formatted-text"
import { ExternalTextLink } from "../../../../atoms/text/text.component"

export class PixeosAddClaimPillComponent extends GenericPillComponent {
  get logoParams(): PillLogoProps | undefined {
    return {
      path: "/images/pill-logos/logo-contract-pixeos-01.svg",
      website: "https://paint.pixeos.art"
    }
  }

  static requireFields: string[] = ["user", "addbalance"]

  static contextForRendering = (): PillRenderingContext => {
    return {
      networks: ["eos-mainnet"],
      validActions: [{ contract: "pixeos1paint", action: "addclaimbal" }]
    }
  }

  static i18n() {
    return {
      en: {
        addclaimpixeosrewards: {
          summary: "<0>{{account}}</0> added <1>{{amountEOS}}</1> to their balance"
        }
      },
      zh: {
        addclaimpixeosrewards: {
          summary: "<0>{{account}}</0> 的余额增加了<1>{{amountEOS}}</1>"
        }
      }
    }
  }

  renderContent = () => {
    const { action } = this.props

    return (
      <Box fontSize={[1]} mx={[2]} minWidth="10px" minHeight="26px" alignItems="center">
        <FormattedText
          fields={getPixeosAddToClaimLevel1Fields(action)}
          i18nKey="pillTemplates.addclaimpixeosrewards.summary"
          fontSize={[1]}
        />
      </Box>
    )
  }

  renderLevel2Template = () => {
    return (
      <Box fontSize={[1]} mx={[2]} minWidth="10px" minHeight="26px" alignItems="center">
        <ExternalTextLink to="https://paint.pixeos.art/">
          https://paint.pixeos.art/
        </ExternalTextLink>
      </Box>
    )
  }
}
