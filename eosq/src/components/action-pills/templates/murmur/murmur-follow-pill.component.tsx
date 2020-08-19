import * as React from "react"
import { GenericPillComponent, PillRenderingContext } from "../generic-pill.component"
import { Box, PillLogoProps, CellValue } from "@dfuse/explorer"
import { KeyValueFormatEllipsis, Text } from "../../../../atoms/text/text.component"

import { Grid } from "../../../../atoms/ui-grid/ui-grid.component"

export class MurmurFollowPillComponent extends GenericPillComponent {
  get logoParams(): PillLogoProps | undefined {
    return {
      path: "/images/pill-logos/logo-contract-murmur-01.svg",
      website: "https://murmurdapp.com"
    }
  }

  static requireFields: string[] = ["from", "to", "weightage"]

  static contextForRendering = (): PillRenderingContext => {
    return {
      networks: ["eos-mainnet"],
      validActions: [{ contract: "murmurdappco", action: "follow" }]
    }
  }

  renderContent = (): JSX.Element => {
    const { action } = this.props

    return (
      <Box minWidth="10px" fontSize={[1]} mx={[2]} alignItems="center">
        <KeyValueFormatEllipsis content={`from: ${action.data.from} to: ${action.data.to}`} />
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
        {this.renderDetailLine("Weight Age: ", this.props.action.data.weightage)}
      </Box>
    )
  }
}
