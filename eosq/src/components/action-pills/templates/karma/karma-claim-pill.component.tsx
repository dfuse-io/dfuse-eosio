import * as React from "react"
import { GenericPillComponent, PillRenderingContext } from "../generic-pill.component"
import Box from "../../../../atoms/ui-box/ui-box.component"
import { getKarmaClaimLevel1Fields } from "../pill-template.helpers"
import { FormattedText } from "../../../formatted-text/formatted-text"
import { ExternalTextLink } from "../../../../atoms/text/text.component"
import { PillLogoProps } from "../../../../atoms/pills/pill"

export class KarmaClaimPillComponent extends GenericPillComponent {
  get logoParams(): PillLogoProps | undefined {
    return {
      path: "/images/pill-logos/logo-contract-karma-01.svg",
      website: "https://karmaapp.io"
    }
  }

  static requireFields: string[] = ["owner"]

  static contextForRendering = (): PillRenderingContext => {
    return {
      networks: ["eos-mainnet"],
      validActions: [{ contract: "therealkarma", action: "claim" }]
    }
  }

  static i18n() {
    return {
      en: {
        karmaclaim: {
          summary: "<0>{{account}}</0> claimed their Power Up rewards"
        }
      },
      zh: {
        karmaclaim: {
          summary: "<0>{{account}}</0> 领取 Power Up 奖励"
        }
      }
    }
  }

  renderContent = () => {
    const { action } = this.props

    return (
      <Box fontSize={[1]} mx={[2]} minWidth="10px" minHeight="26px" alignItems="center">
        <FormattedText
          fields={getKarmaClaimLevel1Fields(action)}
          i18nKey="pillTemplates.karmaclaim.summary"
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
