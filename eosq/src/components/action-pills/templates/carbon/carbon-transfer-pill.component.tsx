import * as React from "react"
import { Box, Pill, PillLogoProps, PillClickable, MonospaceText } from "@dfuse/explorer"
import { theme } from "../../../../theme"
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

  renderPill2 = () => {
    const colorVariant = this.isReceiveTransfer()
      ? "traceActionReceiveBackground"
      : "traceActionSendBackground"

    if (!this.props.headerAndTitleOptions.title) {
      return (
        <Box px="2px" bg={this.props.pill2Color || theme.colors[colorVariant]}>
          &nbsp;
        </Box>
      )
    }

    const WrapperComponent = this.props.disabled ? Box : PillClickable

    return (
      <WrapperComponent bg={this.props.pill2Color || theme.colors[colorVariant]}>
        <MonospaceText alignSelf="center" px={[2]} color="text" fontSize={[1]}>
          {this.props.headerAndTitleOptions.title}
        </MonospaceText>
      </WrapperComponent>
    )
  }

  render() {
    const memoText = getMemoText(this.props.action)
    let logo: PillLogoProps | undefined = {
      path: "/images/pill-logos/logo-contract-carbon-01.svg",
      website: "https://carbon.money"
    }
    if (this.props.traceInfo && this.props.action.account !== this.props.traceInfo.receiver) {
      logo = undefined
    }

    return (
      <Pill
        pill2={this.renderPill2()}
        logo={logo}
        highlighted={this.props.highlighted}
        headerHoverTitle={this.props.headerAndTitleOptions.header.hoverTitle}
        disabled={this.props.disabled}
        info={memoText}
        headerText={this.renderHeaderText()}
        renderExpandedContent={() => {
          return this.renderExpandedContent()
        }}
        renderInfo={this.renderLevel2Template}
        content={this.renderContent()}
      />
    )
  }
}
