import * as React from "react"
import { GenericPillComponent, PillRenderingContext } from "../generic-pill.component"
import { Box } from "@dfuse/explorer"
import { FormattedText } from "../../../formatted-text/formatted-text"
import { ExternalTextLink, Text } from "../../../../atoms/text/text.component"
import { PillLogoProps } from "../../../../atoms/pills/pill"
import { getInfiniverseSetLandPriceLevel1Fields } from "../pill-template.helpers"
import { Grid } from "../../../../atoms/ui-grid/ui-grid.component"
import { CellValue } from "@dfuse/explorer"

export class InfiniverseSetlandpricePillComponent extends GenericPillComponent {
  get logoParams(): PillLogoProps | undefined {
    return {
      path: "/images/pill-logos/logo-contract-infiniverse-01.svg",
      website: "https://infiniverse.net"
    }
  }

  static requireFields: string[] = ["land_id", "price"]

  static contextForRendering = (): PillRenderingContext => {
    return {
      networks: ["eos-mainnet"],
      validActions: [{ contract: "infiniverse1", action: "setlandprice" }]
    }
  }

  static i18n() {
    return {
      en: {
        infiniversesetlandprice: {
          summary:
            "<0>{{authorizer}}</0> set a price of <1>{{quantity}}</1> for Land <2>{{land_id}}</2>"
        }
      },
      zh: {
        infiniversesetlandprice: {
          summary: "<0>{{authorizer}}</0> 为 Land <2>{{land_id}}</2> 定价 <1>{{quantity}}</1> "
        }
      }
    }
  }

  renderContent = () => {
    const { action } = this.props

    return (
      <Box fontSize={[1]} mx={[2]} minWidth="10px" minHeight="26px" alignItems="center">
        <FormattedText
          fields={getInfiniverseSetLandPriceLevel1Fields(action)}
          i18nKey="pillTemplates.infiniversesetlandprice.summary"
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
