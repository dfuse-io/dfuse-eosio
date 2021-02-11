import * as React from "react"
import {
  Box,
  explodeJson,
  DetailLineAuto,
  Pill,
  PillLogoProps,
  PillClickable,
  MonospaceText,
} from "@dfuse/explorer"

import { Cell } from "../../../atoms/ui-grid/ui-grid.component"

import { KeyValueFormatEllipsis, Text } from "../../../atoms/text/text.component"
import { theme } from "../../../theme"
import { t } from "i18next"
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome"
import { faBell } from "@fortawesome/free-solid-svg-icons"
import { PillTabContentComponent } from "./pill-tab-content.component"
import { truncateJsonString, truncateStringPlus } from "./pill-template.helpers"
import { GenericPillParams, GenericPillState } from "../../../models/pill-templates"
import { Config } from "../../../models/config"
import { PILL_CONTRACT_LOGOS } from "./all-logos"

export interface GenericPillComponentInterface {
  renderContent(): JSX.Element | null
  renderLevel2Template(): JSX.Element | null
  renderExpandedContent(): JSX.Element | null
  render(): JSX.Element | null
}

export interface PillTargetAction {
  contract?: string
  action: string
}

export interface PillRenderingContext {
  validActions: PillTargetAction[]
  networks: string[]
  needsTranslate?: boolean
}

export interface PillComponentClass<P = any, S = any> extends React.ComponentClass<P, S> {
  requireFields: string[]
  contextForRendering(): PillRenderingContext
  i18n?(): { [key: string]: any }
}

export class GenericPillComponent
  extends React.Component<GenericPillParams, GenericPillState>
  implements GenericPillComponentInterface {
  jsonData: any = {}
  croppedData: any = {}
  hasCroppedData = false
  dataCutOff = 200

  get logoParams(): PillLogoProps | undefined {
    const availableLogos = PILL_CONTRACT_LOGOS[Config.network_id] || []
    const logoParams = availableLogos.find((ref: any) => {
      if (ref.action) {
        return ref.contract === this.props.action.account && ref.action === this.props.action.name
      }

      return ref.contract === this.props.action.account
    })

    if (logoParams) {
      return {
        path: logoParams.path,
        website: logoParams.website,
      }
    }

    return undefined
  }

  get logo(): PillLogoProps | undefined {
    let { logoParams } = this
    if (this.props.traceInfo && this.props.action.account !== this.props.traceInfo.receiver) {
      logoParams = undefined
    }

    return logoParams
  }

  constructor(props: GenericPillParams) {
    super(props)

    this.rebuildData()
    this.state = { fullContent: false }
  }

  componentDidUpdate(prevProps: Readonly<GenericPillParams>): void {
    if (prevProps.action !== this.props.action) {
      this.rebuildData()
      this.forceUpdate()
    }
  }

  rebuildData() {
    if (this.props.action.data == null) {
      if (this.props.action.hex_data) {
        this.croppedData = truncateStringPlus(this.props.action.hex_data, this.dataCutOff)
        return
      }

      this.jsonData = ""
      return
    }

    if (typeof this.props.action.data === "string") {
      this.croppedData = truncateStringPlus(this.props.action.data, this.dataCutOff)
      return
    }

    const dataString = JSON.stringify(this.props.action.data)
    this.jsonData = JSON.parse(dataString)

    this.croppedData = truncateJsonString(dataString, this.dataCutOff, () => {
      this.hasCroppedData = true
    })
  }

  showFullContent = () => {
    this.setState({ fullContent: true })
  }

  blockNum() {
    return this.props.pageContext && this.props.pageContext.blockNum
      ? this.props.pageContext.blockNum
      : undefined
  }

  hasInlineTraces() {
    return (
      this.props.traceInfo &&
      this.props.traceInfo.inline_traces &&
      this.props.traceInfo.inline_traces.length > 0
    )
  }

  renderLevel2Template = (): JSX.Element | null => {
    const { data } = this.props.action

    if (data && data.memo) {
      return <DetailLineAuto label={t("transaction.pill.memo")}>{data.memo}</DetailLineAuto>
    }

    return null
  }

  renderPill2 = () => {
    if (!this.props.headerAndTitleOptions.title) {
      return (
        <Box px="2px" bg={this.props.pill2Color || theme.colors.traceActionGenericBackground}>
          &nbsp;
        </Box>
      )
    }

    const WrapperComponent = this.props.disabled ? Box : PillClickable

    return (
      <WrapperComponent bg={this.props.pill2Color || theme.colors.traceActionGenericBackground}>
        <MonospaceText alignSelf="center" px={[2]} color="text" fontSize={[1]}>
          {this.props.headerAndTitleOptions.title}
        </MonospaceText>
      </WrapperComponent>
    )
  }

  renderExpandedContent = (): JSX.Element => {
    const displayFullContentButton = !this.state.fullContent && this.hasCroppedData
    return (
      <PillTabContentComponent
        ramops={this.props.ramops}
        tableops={this.props.tableops}
        traceInfo={this.props.traceInfo}
        dbops={this.props.dbops}
        data={this.state.fullContent ? this.jsonData : this.croppedData}
        displayFullContentButton={displayFullContentButton}
        console={this.props.console}
        action={this.props.action}
        onDisplayFullContent={this.showFullContent}
        blockNum={this.blockNum()}
      />
    )
  }

  renderDefaultContent() {
    return (
      <Box minWidth="10px" fontSize={[1]} mx={[2]} alignItems="center">
        <KeyValueFormatEllipsis content={explodeJson(this.croppedData)} />
      </Box>
    )
  }

  renderContent = (): JSX.Element => {
    return this.renderDefaultContent()
  }

  renderTextWrapper(content: JSX.Element | string, padding?: number[]) {
    return (
      <Text
        display="inline-block"
        color={theme.colors.primary}
        fontSize={[1]}
        fontFamily="'Roboto Mono', monospace;"
        pr={padding}
      >
        {content}
      </Text>
    )
  }

  renderHeaderText() {
    const headerText = this.props.headerAndTitleOptions.header.text
    if (headerText.includes("notification:")) {
      return (
        <Cell>
          {this.renderTextWrapper(<FontAwesomeIcon icon={faBell as any} />, [2])}
          {this.renderTextWrapper(headerText.replace("notification:", ""))}
        </Cell>
      )
    }

    return headerText
  }

  render(): JSX.Element {
    return (
      <Pill
        pill2={this.renderPill2()}
        logo={this.logo}
        highlighted={this.props.highlighted}
        headerHoverTitle={this.props.headerAndTitleOptions.header.hoverTitle}
        disabled={this.props.disabled}
        headerBgColor={theme.colors.traceAccountGenericBackground}
        expandButtonBgColor={theme.colors.traceAccountGenericBackground}
        expandButtonColor={theme.colors.traceAccountText}
        headerText={this.renderHeaderText()}
        renderExpandedContent={this.renderExpandedContent}
        content={this.croppedData ? this.renderContent() : <span />}
        renderInfo={this.renderLevel2Template}
      />
    )
  }
}
