import * as React from "react"
import { Box } from "@dfuse/explorer"
import { getMemoText } from "../../../../helpers/action.helpers"
import { GenericPillComponent, PillRenderingContext } from "../generic-pill.component"
import { getCarbonBurnLevel1Fields } from "../pill-template.helpers"
import { FormattedText } from "../../../formatted-text/formatted-text"
import { Pill, PillLogoProps } from "../../../../atoms/pills/pill"

export class CarbonBurnPillComponent extends GenericPillComponent {
  get logoParams(): PillLogoProps | undefined {
    return {
      path: "/images/pill-logos/logo-contract-carbon-01.svg",
      website: "https://carbon.money"
    }
  }

  static requireFields: string[] = ["quantity", "from"]

  static contextForRendering = (): PillRenderingContext => {
    return {
      networks: ["eos-mainnet"],

      validActions: [{ contract: "stablecarbon", action: "burn" }]
    }
  }

  static i18n() {
    return {
      en: {
        carbonburn: {
          summary: "Carbon Fiber burned <0>{{amountCUSD}}</0> from <1>{{from}}</1>"
        }
      },
      zh: {
        carbonburn: {
          summary: "Carbon Fiber 从<1>{{from}}</1> 烧毁了<0>{{amountCUSD}}</0> "
        }
      }
    }
  }

  renderContent = () => {
    const { action } = this.props

    return (
      <Box fontSize={[1]} mx={[2]} minWidth="10px" minHeight="26px" alignItems="center">
        <FormattedText
          fields={getCarbonBurnLevel1Fields(action)}
          i18nKey="pillTemplates.carbonburn.summary"
          fontSize={[1]}
        />
      </Box>
    )
  }

  render() {
    const memoText = getMemoText(this.props.action)

    return (
      <Pill
        logo={this.logo}
        highlighted={this.props.highlighted}
        headerHoverTitle={this.props.headerAndTitleOptions.header.hoverTitle}
        disabled={this.props.disabled}
        info={memoText}
        colorVariant="traceActionGenericBackground"
        colorVariantHeader={this.props.headerAndTitleOptions.header.color}
        headerText={this.renderHeaderText()}
        renderExpandedContent={() => {
          return this.renderExpandedContent()
        }}
        renderInfo={this.renderLevel2Template}
        content={this.renderContent()}
        title={this.props.headerAndTitleOptions.title}
      />
    )
  }
}
