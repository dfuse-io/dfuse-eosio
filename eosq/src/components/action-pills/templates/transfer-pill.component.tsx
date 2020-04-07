import * as React from "react"
import { Pill } from "../../../atoms/pills/pill"
import { TransferBox } from "../../../atoms/pills/pill-transfer-box"
import { getMemoText } from "../../../helpers/action.helpers"
import { GenericPillComponent, PillRenderingContext } from "./generic-pill.component"
import { getNewAccountFromNameServiceFields, getNewAccountInTraces } from "./pill-template.helpers"
import { Grid } from "../../../atoms/ui-grid/ui-grid.component"
import { Text } from "../../../atoms/text/text.component"
import { CellValue } from "../../../atoms/pills/detail-line"
import { FormattedText } from "../../formatted-text/formatted-text"

export class TransferPillComponent extends GenericPillComponent {
  static requireFields: string[] = ["from", "to", "quantity"]

  static contextForRendering = (): PillRenderingContext => {
    return {
      networks: ["all"],
      validActions: [{ action: "transfer" }]
    }
  }
  static i18n = () => {
    return {
      en: {
        pillTemplates: {
          newAccountFromNameService:
            "The account <0>{{account}}</0> was created using <1>{{link}}</1>"
        }
      },
      zh: {
        pillTemplates: {
          newAccountFromNameService: "账户 <0>{{account}}</0> 通过 <1>{{link}}</1> 创建"
        }
      }
    }
  }

  isReceiveTransfer() {
    return (
      this.props.pageContext && this.props.pageContext.accountName === this.props.action.data.to
    )
  }

  renderContent = () => {
    const { action } = this.props

    return (
      <TransferBox
        context={this.props.pageContext ? this.props.pageContext.accountName : undefined}
        from={action.data.from}
        to={action.data.to}
        amount={action.data.quantity}
        memo={action.data.memo}
      />
    )
  }

  renderLevel2Template = () => {
    const { action } = this.props

    let newAccount
    if (action.data.to === "ens.xyz" && this.hasInlineTraces()) {
      newAccount = getNewAccountInTraces(this.props.traceInfo)
    }

    const memoText = getMemoText(this.props.action)
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
        {memoText ? this.renderDetailLine("Memo: ", memoText) : null}
        {newAccount ? (
          <FormattedText
            fontSize={[2]}
            i18nKey="pillTemplates.newAccountFromNameService"
            fields={getNewAccountFromNameServiceFields(newAccount)}
          />
        ) : null}
      </Grid>
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

  render() {
    const colorVariant = this.isReceiveTransfer()
      ? "traceActionReceiveBackground"
      : "traceActionSendBackground"

    return (
      <Pill
        logo={this.logo}
        highlighted={this.props.highlighted}
        headerHoverTitle={this.props.headerAndTitleOptions.header.hoverTitle}
        disabled={this.props.disabled}
        colorVariant={colorVariant}
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
