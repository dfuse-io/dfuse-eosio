import * as React from "react"
import { GenericPillComponent, PillRenderingContext } from "../generic-pill.component"
import { Box, PillLogoProps, CellValue } from "@dfuse/explorer"
import { FormattedText } from "../../../formatted-text/formatted-text"
import { Text } from "../../../../atoms/text/text.component"

import { getInfiniverseRegisterlandLevel1Fields } from "../pill-template.helpers"
import { Grid } from "../../../../atoms/ui-grid/ui-grid.component"

import { t } from "../../../../i18n"

export class InfiniverseRegisterlandPillComponent extends GenericPillComponent {
  get logoParams(): PillLogoProps | undefined {
    return {
      path: "/images/pill-logos/logo-contract-infiniverse-01.svg",
      website: "https://infiniverse.net"
    }
  }

  static requireFields: string[] = [
    "lat_north_edge",
    "lat_south_edge",
    "long_east_edge",
    "long_west_edge",
    "owner"
  ]

  static contextForRendering = (): PillRenderingContext => {
    return {
      networks: ["eos-mainnet"],
      validActions: [{ contract: "infiniverse1", action: "registerland" }]
    }
  }

  static i18n() {
    return {
      en: {
        infiniverseregisterland: {
          summary: "<0>{{owner}}</0> registered their land",
          southEdge: "South Edge:",
          northEdge: "North Edge:",
          eastEdge: "East Edge:",
          westEdge: "West Edge:"
        }
      },
      zh: {
        infiniverseregisterland: {
          summary: "<0>{{owner}}</0> 注册了土地",
          southEdge: "South Edge:",
          northEdge: "North Edge:",
          eastEdge: "East Edge:",
          westEdge: "West Edge:"
        }
      }
    }
  }

  renderContent = () => {
    const { action } = this.props

    return (
      <Box fontSize={[1]} mx={[2]} minWidth="10px" minHeight="26px" alignItems="center">
        <FormattedText
          fields={getInfiniverseRegisterlandLevel1Fields(action)}
          i18nKey="pillTemplates.infiniverseregisterland.summary"
          fontSize={[1]}
        />
      </Box>
    )
  }

  renderDetailLine(title: string, children: JSX.Element | JSX.Element[] | string) {
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
        gridTemplateRows={["1fr 1fr 1fr 1fr"]}
      >
        {this.renderDetailLine(
          t("pillTemplates.infiniverseregisterland.northEdge"),
          this.props.action.data.lat_north_edge
        )}
        {this.renderDetailLine(
          t("pillTemplates.infiniverseregisterland.southEdge"),
          this.props.action.data.lat_south_edge
        )}
        {this.renderDetailLine(
          t("pillTemplates.infiniverseregisterland.eastEdge"),
          this.props.action.data.long_east_edge
        )}
        {this.renderDetailLine(
          t("pillTemplates.infiniverseregisterland.westEdge"),
          this.props.action.data.long_west_edge
        )}
      </Grid>
    )
  }
}
