import * as React from "react"
import { GenericPillComponent, PillRenderingContext } from "../generic-pill.component"
import Box from "../../../../atoms/ui-box/ui-box.component"
import { getPixeosClaimLevel1Fields } from "../pill-template.helpers"
import { FormattedText } from "../../../formatted-text/formatted-text"
import { ExternalTextLink } from "../../../../atoms/text/text.component"
import { PillLogoProps } from "../../../../atoms/pills/pill"

export class PixeosClaimRewardsPillComponent extends GenericPillComponent {
  get logoParams(): PillLogoProps | undefined {
    return {
      path: "/images/pill-logos/logo-contract-pixeos-01.svg",
      website: "https://paint.pixeos.art"
    }
  }

  static requireFields: string[] = ["owner"]

  static contextForRendering = (): PillRenderingContext => {
    return {
      networks: ["eos-mainnet"],
      validActions: [{ contract: "pixeos1paint", action: "claimbal" }]
    }
  }

  static i18n() {
    return {
      en: {
        claimpixeosrewards: {
          summary: "<0>{{account}}</0> claimed their balance of <1>{{amountEOS}}</1>"
        }
      },
      zh: {
        claimpixeosrewards: {
          summary: "<0>{{account}}</0> 取出了余额中的 <1>{{amountEOS}}</1>"
        }
      }
    }
  }

  renderContent = () => {
    const { action } = this.props

    return (
      <Box fontSize={[1]} mx={[2]} minWidth="10px" minHeight="26px" alignItems="center">
        <FormattedText
          fields={getPixeosClaimLevel1Fields(action, this.props.traceInfo)}
          i18nKey="pillTemplates.claimpixeosrewards.summary"
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
