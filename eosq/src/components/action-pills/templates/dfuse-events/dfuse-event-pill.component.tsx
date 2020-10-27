import * as React from "react"
import { Box, Pill, PillLogoProps, CellValue, PillClickable, MonospaceText } from "@dfuse/explorer"
import { theme } from "../../../../theme"
import { GenericPillComponent, PillRenderingContext } from "../generic-pill.component"
import { getDfuseEventLevel1Fields } from "../pill-template.helpers"
import { FormattedText } from "../../../formatted-text/formatted-text"

import { Cell, Grid } from "../../../../atoms/ui-grid/ui-grid.component"
import { Text } from "../../../../atoms/text/text.component"
import { t } from "../../../../i18n"

export class DfuseEventPillComponent extends GenericPillComponent {
  get logoParams(): PillLogoProps | undefined {
    return {
      path: "/images/pill-logos/logo-contract-dfuse-01.svg",
      website: "https://docs.dfuse.io/#dfuse-events"
    }
  }

  static requireFields: string[] = ["auth_key", "data"]

  static contextForRendering = (): PillRenderingContext => {
    return {
      networks: ["eos-mainnet"],
      validActions: [{ contract: "dfuseiohooks", action: "event" }]
    }
  }

  static i18n() {
    return {
      en: {
        dfuseevents: {
          summary: "<0>Indexed fields: </0> {{fields}}",
          authKey: "Auth Key: ",
          indexedFields: "Indexed Fields: "
        }
      },
      zh: {
        dfuseevents: {
          summary: "<0>索引字段：</0> {{fields}}",
          authKey: "Auth Key: ",
          indexedFields: "索引字段："
        }
      }
    }
  }

  renderContent = () => {
    const { action } = this.props

    return (
      <Box fontSize={[1]} mx={[2]} minWidth="10px" minHeight="26px" alignItems="center">
        <FormattedText
          fields={getDfuseEventLevel1Fields(action)}
          i18nKey="pillTemplates.dfuseevents.summary"
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
    const fields = this.props.action.data.data.split("&")

    return (
      <Grid
        fontSize={[1]}
        gridRowGap={[3]}
        mx={[2]}
        minWidth="10px"
        minHeight="26px"
        alignItems="center"
        gridTemplateRows={["1fr auto"]}
      >
        {this.renderDetailLine(
          t("pillTemplates.dfuseevents.authKey"),
          this.props.action.data.auth_key
        )}
        {this.renderDetailLine(
          t("pillTemplates.dfuseevents.indexedFields"),
          fields.map((field: string, index: number) => {
            const fieldValues = field.split("=")
            if (fieldValues.length < 2) {
              return null
            }

            return (
              <Cell key={index} pb={[1]}>
                <Text fontWeight="bold" fontSize={[2]} display="inline-block">
                  {fieldValues[0]} ={" "}
                </Text>{" "}
                {fieldValues[1]}
              </Cell>
            )
          })
        )}
      </Grid>
    )
  }

  renderPill2 = () => {
    if (!this.props.headerAndTitleOptions.title) {
      return (
        <Box px="2px" bg={this.props.pill2Color || theme.colors.bleu11}>
          &nbsp;
        </Box>
      )
    }

    const WrapperComponent = this.props.disabled ? Box : PillClickable

    return (
      <WrapperComponent bg={this.props.pill2Color || theme.colors.bleu11}>
        <MonospaceText alignSelf="center" px={[2]} color="text" fontSize={[1]}>
          {this.props.headerAndTitleOptions.title}
        </MonospaceText>
      </WrapperComponent>
    )
  }

  render(): JSX.Element {
    return (
      <Pill
        pill2={this.renderPill2()}
        logo={this.logo}
        highlighted={this.props.highlighted}
        headerBgColor={theme.colors.traceAccountGenericBackground}
        expandButtonBgColor={theme.colors.traceAccountGenericBackground}
        expandButtonColor={theme.colors.traceAccountText}
        headerHoverTitle="dfuseiohooks"
        disabled={this.props.disabled}
        headerText="event"
        renderExpandedContent={this.renderExpandedContent}
        content={this.croppedData ? this.renderContent() : <span />}
        renderInfo={this.renderLevel2Template}
      />
    )
  }
}
