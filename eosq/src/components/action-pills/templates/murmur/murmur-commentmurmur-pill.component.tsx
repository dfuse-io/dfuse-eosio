import * as React from "react"
import { GenericPillComponent, PillRenderingContext } from "../generic-pill.component"
import { Box } from "@dfuse/explorer"
import { KeyValueFormatEllipsis, Text } from "../../../../atoms/text/text.component"
import { PillLogoProps } from "../../../../atoms/pills/pill"
import { Grid } from "../../../../atoms/ui-grid/ui-grid.component"
import { CellValue } from "../../../../atoms/pills/detail-line"

export class MurmurCommentMurmurPillComponent extends GenericPillComponent {
  get logoParams(): PillLogoProps | undefined {
    return {
      path: "/images/pill-logos/logo-contract-murmur-01.svg",
      website: "https://murmurdapp.com"
    }
  }

  static requireFields: string[] = ["comment", "from", "murmur_id"]

  static contextForRendering = (): PillRenderingContext => {
    return {
      networks: ["eos-mainnet"],
      validActions: [{ contract: "murmurdappco", action: "comentmurmur" }]
    }
  }

  renderContent = (): JSX.Element => {
    const { action } = this.props

    return (
      <Box minWidth="10px" fontSize={[1]} mx={[2]} alignItems="center">
        <KeyValueFormatEllipsis
          content={`from: ${action.data.from} comment: ${action.data.comment}`}
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
