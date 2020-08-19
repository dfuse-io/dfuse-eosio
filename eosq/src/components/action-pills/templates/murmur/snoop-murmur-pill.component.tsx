import * as React from "react"
import { GenericPillComponent, PillRenderingContext } from "../generic-pill.component"
import { Box, PillLogoProps, CellValue } from "@dfuse/explorer"
import { KeyValueFormatEllipsis, Text } from "../../../../atoms/text/text.component"

import { Grid } from "../../../../atoms/ui-grid/ui-grid.component"

export class SnoopMurmurPillComponent extends GenericPillComponent {
  get logoParams(): PillLogoProps | undefined {
    return {
      path: "/images/pill-logos/logo-contract-murmur-01.svg",
      website: "https://murmurdapp.com"
    }
  }

  static requireFields: string[] = ["from", "murmur_id", "snoop_type"]

  static contextForRendering = (): PillRenderingContext => {
    return {
      networks: ["eos-mainnet"],
      validActions: [{ contract: "murmurdappco", action: "snoopmurmur" }]
    }
  }

  renderContent = (): JSX.Element => {
    const { action } = this.props

    return (
      <Box minWidth="10px" fontSize={[1]} mx={[2]} alignItems="center">
        <KeyValueFormatEllipsis
          content={`from: ${action.data.from} snoop_type: ${action.data.snoop_type}`}
        />
      </Box>
    )
  }

  renderDetailLine(title: string, children: JSX.Element | JSX.Element[]) {
    const templateColumns = ["1fr", "150px 1fr"]

    return (
      <Grid gridTemplateColumns={templateColumns}>
        <Text color="text" fontWeight="bold">
          {title}&nbsp;
        </Text>
        <CellValue>{children}</CellValue>
      </Grid>
    )
  }

  renderLevel2Template = () => {
    return (
      <Box fontSize={[1]} mx={[2]} minWidth="10px" minHeight="26px" alignItems="center">
        {this.renderDetailLine("Murmur ID: ", this.props.action.data.murmur_id)}
      </Box>
    )
  }
}
