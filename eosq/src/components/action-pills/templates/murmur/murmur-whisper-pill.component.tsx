import * as React from "react"
import { GenericPillComponent, PillRenderingContext } from "../generic-pill.component"
import { Box } from "@dfuse/explorer"
import { KeyValueFormatEllipsis, Text } from "../../../../atoms/text/text.component"
import { PillLogoProps } from "../../../../atoms/pills/pill"
import { Grid } from "../../../../atoms/ui-grid/ui-grid.component"
import { CellValue } from "../../../../atoms/pills/detail-line"

export class MurmurWhisperPillComponent extends GenericPillComponent {
  get logoParams(): PillLogoProps | undefined {
    return {
      path: "/images/pill-logos/logo-contract-murmur-01.svg",
      website: "https://murmurdapp.com"
    }
  }

  static requireFields: string[] = [
    "encrypted_message",
    "from",
    "from_public_key",
    "to",
    "to_public_key"
  ]

  static contextForRendering = (): PillRenderingContext => {
    return {
      networks: ["eos-mainnet"],
      validActions: [{ contract: "murmurdappco", action: "whisper" }]
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
      <Grid
        fontSize={[1]}
        gridRowGap={[3]}
        mx={[2]}
        minWidth="10px"
        minHeight="26px"
        alignItems="center"
        gridTemplateRows={["1fr 1fr"]}
      >
        {this.renderDetailLine("From Key: ", this.props.action.data.from_public_key)}
        {this.renderDetailLine("To Key: ", this.props.action.data.to_public_key)}
      </Grid>
    )
  }
}
