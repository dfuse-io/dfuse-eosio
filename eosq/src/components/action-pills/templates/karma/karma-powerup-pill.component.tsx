import * as React from "react"
import { GenericPillComponent, PillRenderingContext } from "../generic-pill.component"
import { Box, PillLogoProps } from "@dfuse/explorer"
import { getKarmaPowerUpLevel1Fields } from "../pill-template.helpers"
import { FormattedText } from "../../../formatted-text/formatted-text"
import { ExternalTextLink } from "../../../../atoms/text/text.component"

export class KarmaPowerupPillComponent extends GenericPillComponent {
  get logoParams(): PillLogoProps | undefined {
    return {
      path: "/images/pill-logos/logo-contract-karma-01.svg",
      website: "https://karmaapp.io"
    }
  }

  static requireFields: string[] = ["owner", "quantity"]

  static contextForRendering = (): PillRenderingContext => {
    return {
      networks: ["eos-mainnet"],
      validActions: [{ contract: "therealkarma", action: "powerup" }]
    }
  }

  static i18n() {
    return {
      en: {
        karmapowerup: {
          summary: "<0>{{account}}</0> used <1>{{amountKarma}}</1> to Power Up"
        }
      },
      zh: {
        karmapowerup: {
          summary: "<0>{{account}}</0> 使用了<1>{{amountKarma}}</1>Power Up"
        }
      }
    }
  }

  renderContent = () => {
    const { action } = this.props

    return (
      <Box fontSize={[1]} mx={[2]} minWidth="10px" minHeight="26px" alignItems="center">
        <FormattedText
          fields={getKarmaPowerUpLevel1Fields(action)}
          i18nKey="pillTemplates.karmapowerup.summary"
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
