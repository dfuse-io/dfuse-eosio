import * as React from "react"
import { GenericPillComponent, PillRenderingContext } from "../generic-pill.component"
import Box from "../../../../atoms/ui-box/ui-box.component"
import { FormattedText } from "../../../formatted-text/formatted-text"
import { ExternalTextLink, Text } from "../../../../atoms/text/text.component"
import { PillLogoProps } from "../../../../atoms/pills/pill"
import { getInfiniverseDeletePersistLevel1Fields } from "../pill-template.helpers"
import { Grid } from "../../../../atoms/ui-grid/ui-grid.component"
import { CellValue } from "../../../../atoms/pills/detail-line"

export class InfiniverseDeletePersistPillComponent extends GenericPillComponent {
  get logoParams(): PillLogoProps | undefined {
    return {
      path: "/images/pill-logos/logo-contract-infiniverse-01.svg",
      website: "https://infiniverse.net"
    }
  }

  static requireFields: string[] = ["persistent_id"]

  static contextForRendering = (): PillRenderingContext => {
    return {
      networks: ["eos-mainnet"],
      validActions: [{ contract: "infiniverse1", action: "deletepersis" }]
    }
  }

  static i18n() {
    return {
      en: {
        infiniversedeletepersist: {
          summary: "<0>{{authorizer}}</0> has deleted the peristent data for their land"
        }
      },
      zh: {
        infiniversedeletepersist: {
          summary: "<0>{{authorizer}}</0> 删除了他们土地的持久数据"
        }
      }
    }
  }

  renderContent = () => {
    const { action } = this.props

    return (
      <Box fontSize={[1]} mx={[2]} minWidth="10px" minHeight="26px" alignItems="center">
        <FormattedText
          fields={getInfiniverseDeletePersistLevel1Fields(action)}
          i18nKey="pillTemplates.infiniversedeletepersist.summary"
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
