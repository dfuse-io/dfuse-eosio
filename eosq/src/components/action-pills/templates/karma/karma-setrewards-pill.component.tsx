import * as React from "react"
import { GenericPillComponent, PillRenderingContext } from "../generic-pill.component"
import { Box } from "@dfuse/explorer"
import { ExternalTextLink } from "../../../../atoms/text/text.component"
import { PillLogoProps } from "../../../../atoms/pills/pill"

export class KarmaSetrewardsPillComponent extends GenericPillComponent {
  get logoParams(): PillLogoProps | undefined {
    return {
      path: "/images/pill-logos/logo-contract-karma-01.svg",
      website: "https://karmaapp.io"
    }
  }

  static requireFields: string[] = []

  static contextForRendering = (): PillRenderingContext => {
    return {
      networks: ["eos-mainnet"],
      validActions: [{ contract: "therealkarma", action: "setrewards" }]
    }
  }

  renderLevel2Template = () => {
    return (
      <Box fontSize={[1]} mx={[2]} minWidth="10px" minHeight="26px" alignItems="center">
        <ExternalTextLink to="https://karmaapp.io">https://karmaapp.io</ExternalTextLink>
      </Box>
    )
  }
}
