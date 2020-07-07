import * as React from "react"
import { GenericPillComponent, PillRenderingContext } from "../generic-pill.component"
import { Box, PillLogoProps } from "@dfuse/explorer"
import { getKarmaClaimPostLevel1Fields } from "../pill-template.helpers"
import { FormattedText } from "../../../formatted-text/formatted-text"
import { ExternalTextLink } from "../../../../atoms/text/text.component"

export class KarmaClaimPostPillComponent extends GenericPillComponent {
  get logoParams(): PillLogoProps | undefined {
    return {
      path: "/images/pill-logos/logo-contract-karma-01.svg",
      website: "https://karmaapp.io"
    }
  }

  static requireFields: string[] = ["author"]

  static contextForRendering = (): PillRenderingContext => {
    return {
      networks: ["eos-mainnet"],
      validActions: [{ contract: "thekarmadapp", action: "claim" }]
    }
  }

  static i18n() {
    return {
      en: {
        karmaclaimpost: {
          summary: "<0>{{account}}</0> claimed their Post rewards"
        }
      },
      zh: {
        karmaclaimpost: {
          summary: "<0>{{account}}</0> 领取 Post 奖励"
        }
      }
    }
  }

  renderContent = () => {
    const { action } = this.props

    return (
      <Box fontSize={[1]} mx={[2]} minWidth="10px" minHeight="26px" alignItems="center">
        <FormattedText
          fields={getKarmaClaimPostLevel1Fields(action)}
          i18nKey="pillTemplates.karmaclaimpost.summary"
          fontSize={[1]}
        />
      </Box>
    )
  }

  renderLevel2Template = () => {
    return (
      <Box fontSize={[1]} mx={[2]} minWidth="10px" minHeight="26px" alignItems="center">
        <ExternalTextLink to="https://karmaapp.io">https://karmaapp.io</ExternalTextLink>
      </Box>
    )
  }
}
