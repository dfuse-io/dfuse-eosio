import * as React from "react"
import { Pill, PillLogoProps } from "../../../../atoms/pills/pill"
import { TransferBox } from "../../../../atoms/pills/pill-transfer-box"
import { getMemoText } from "../../../../helpers/action.helpers"
import { GenericPillComponent, PillRenderingContext } from "../generic-pill.component"

export class CarbonTransferPillComponent extends GenericPillComponent {
  get logoParams(): PillLogoProps | undefined {
    return {
      path: "/images/pill-logos/logo-contract-carbon-01.svg",
      website: "https://carbon.money"
    }
  }

  static requireFields: string[] = ["from", "to", "quantity"]

  static contextForRendering = (): PillRenderingContext => {
    return {
      networks: ["eos-mainnet"],

      validActions: [{ contract: "stablecarbon", action: "transfer" }]
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

  render() {
    const memoText = getMemoText(this.props.action)
    const colorVariant = this.isReceiveTransfer()
      ? "traceActionReceiveBackground"
      : "traceActionSendBackground"

    let logo: PillLogoProps | undefined = {
      path: "/images/pill-logos/logo-contract-carbon-01.svg",
      website: "https://carbon.money"
    }
    if (this.props.traceInfo && this.props.action.account !== this.props.traceInfo.receiver) {
      logo = undefined
    }

    return (
      <Pill
        logo={logo}
        highlighted={this.props.highlighted}
        headerHoverTitle={this.props.headerAndTitleOptions.header.hoverTitle}
        disabled={this.props.disabled}
        info={memoText}
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
