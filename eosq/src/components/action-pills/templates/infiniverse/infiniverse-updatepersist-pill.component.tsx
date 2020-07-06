import * as React from "react"
import { GenericPillComponent, PillRenderingContext } from "../generic-pill.component"
import { Box } from "@dfuse/explorer"
import { FormattedText } from "../../../formatted-text/formatted-text"
import { ExternalTextLink, Text } from "../../../../atoms/text/text.component"
import { PillLogoProps } from "../../../../atoms/pills/pill"
import { getInfiniverseUpdatePersistLevel1Fields } from "../pill-template.helpers"
import { Grid } from "../../../../atoms/ui-grid/ui-grid.component"
import { CellValue } from "../../../../atoms/pills/detail-line"

export class InfiniverseUpdatePersistPillComponent extends GenericPillComponent {
  get logoParams(): PillLogoProps | undefined {
    return {
      path: "/images/pill-logos/logo-contract-infiniverse-01.svg",
      website: "https://infiniverse.net"
    }
  }

  static requireFields: string[] = ["land_id", "orientation", "persistent_id", "position", "scale"]

  static contextForRendering = (): PillRenderingContext => {
    return {
      networks: ["eos-mainnet"],
      validActions: [{ contract: "infiniverse1", action: "updatepersis" }]
    }
  }

  static i18n() {
    return {
      en: {
        infiniverseupdatepersist: {
          summary: "<0>Land ID:</0> <1>{{land_id}}</1>, <2>Persistent ID:</2>  <3>{{poly_id}}</3>",
          orientation: "Orientation:",
          position: "Position:",
          scale: "Scale:"
        }
      },
      zh: {
        infiniverseupdatepersist: {
          summary: "<0>Land ID:</0> <1>{{land_id}}</1>, <2>Persistent ID:</2>  <3>{{poly_id}}</3>",
          orientation: "方向：",
          position: "位置：",
          scale: "大小："
        }
      }
    }
  }

  renderContent = () => {
    const { action } = this.props

    return (
      <Box fontSize={[1]} mx={[2]} minWidth="10px" minHeight="26px" alignItems="center">
        <FormattedText
          fields={getInfiniverseUpdatePersistLevel1Fields(action)}
          i18nKey="pillTemplates.infiniverseupdatepersist.summary"
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
      <Box fontSize={[1]} mx={[2]} minWidth="10px" minHeight="26px" alignItems="center">
        <ExternalTextLink to="https://infiniverse.net">https://infiniverse.net</ExternalTextLink>
      </Box>
    )
  }
}
