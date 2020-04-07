import * as React from "react"
import { GenericPillComponent, PillRenderingContext } from "../generic-pill.component"
import Box from "../../../../atoms/ui-box/ui-box.component"
import { FormattedText } from "../../../formatted-text/formatted-text"
import { Text } from "../../../../atoms/text/text.component"
import { PillLogoProps } from "../../../../atoms/pills/pill"
import { getInfiniverseMoveLandLevel1Fields } from "../pill-template.helpers"
import { Grid } from "../../../../atoms/ui-grid/ui-grid.component"
import { CellValue } from "../../../../atoms/pills/detail-line"
import { t } from "../../../../i18n"

export class MoveLandPillComponent extends GenericPillComponent {
  get logoParams(): PillLogoProps | undefined {
    return {
      path: "/images/pill-logos/logo-contract-infiniverse-01.svg",
      website: "https://infiniverse.net"
    }
  }

  static requireFields: string[] = [
    "land_id",
    "lat_north_edge",
    "lat_south_edge",
    "long_east_edge",
    "long_west_edge"
  ]

  static contextForRendering = (): PillRenderingContext => {
    return {
      networks: ["eos-mainnet"],
      validActions: [{ contract: "infiniverse1", action: "moveland" }]
    }
  }

  static i18n() {
    return {
      en: {
        infiniversemoveland: {
          summary: "<0>{{authorizer}}</0> moved their land",
          southEdge: "South Edge:",
          northEdge: "North Edge:",
          eastEdge: "East Edge:",
          westEdge: "West Edge:"
        }
      },
      zh: {
        infiniversemoveland: {
          summary: "<0>{{authorizer}}</0> 移动了土地",
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
          fields={getInfiniverseMoveLandLevel1Fields(action)}
          i18nKey="pillTemplates.infiniversemoveland.summary"
          fontSize={[1]}
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
          t("pillTemplates.infiniversemoveland.northEdge"),
          this.props.action.data.lat_north_edge
        )}
        {this.renderDetailLine(
          t("pillTemplates.infiniversemoveland.southEdge"),
          this.props.action.data.lat_south_edge
        )}
        {this.renderDetailLine(
          t("pillTemplates.infiniversemoveland.eastEdge"),
          this.props.action.data.long_east_edge
        )}
        {this.renderDetailLine(
          t("pillTemplates.infiniversemoveland.westEdge"),
          this.props.action.data.long_west_edge
        )}
      </Grid>
    )
  }
}
