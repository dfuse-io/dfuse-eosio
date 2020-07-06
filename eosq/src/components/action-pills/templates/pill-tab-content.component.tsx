import { Cell } from "../../../atoms/ui-grid/ui-grid.component"
import { SearchShortcut } from "../../search-shortcut/search-shortcut"
import { AutorizationBox } from "../../authorization-box/authorization-box.component"
import { DetailLine } from "../../../atoms/pills/detail-line"
import { t } from "i18next"
import { LinkStyledText, Text } from "../../../atoms/text/text.component"
import { theme, styled } from "../../../theme"
import { JsonWrapper } from "@dfuse/explorer"
import { RamUsage } from "../../ram-usage/ram-usage.component"
import { DBOperations } from "../../db-operations/db-operations.component"
import * as React from "react"
import { DbOp, RAMOp, TableOp, Action, Authorization } from "@dfuse/client"
import { MonospaceTextLink } from "../../../atoms/text-elements/misc"
import { Links } from "../../../routes"
import { VerticalTabs } from "../../../atoms/vertical-tabs/vertical-tabs"
import { decodeDBOps } from "../../../services/dbops"
import { TraceInfo } from "../../../models/pill-templates"

const ContentWrapper: React.ComponentType<any> = styled(Cell)`
  padding: 24px 24px 24px 40px;
`

const RawWrapper = styled(Cell)`
  margin-bottom: 10px;
`

const TabContentWrapper: React.ComponentType<any> = styled(Cell)`
  overflow-y: scroll;
  max-height: 500px;
`

export const PILL_TAB_VALUES = {
  DBOPS: "dbops",
  RAMOPS: "ramops",
  GENERAL: "general",
  CONSOLE: "console",
  JSON_DATA: "jsonData",
  HEX_DATA: "hexData"
}

export interface Props {
  console?: string
  dbops?: DbOp[]
  ramops?: RAMOp[]
  tableops?: TableOp[]
  action: Action<any>
  traceInfo?: TraceInfo
  data: any
  displayFullContentButton: boolean
  onDisplayFullContent: () => void
  blockNum?: number
}

interface State {
  currentTab: string
  decodedDBOps: DbOp[]
  isDecodedDBOps: boolean
}

export class PillTabContentComponent extends React.Component<Props, State> {
  PILL_TABS = [{ label: t("transaction.pill.general"), value: PILL_TAB_VALUES.GENERAL }]

  constructor(props: Props) {
    super(props)

    this.state = {
      currentTab: PILL_TAB_VALUES.GENERAL,
      isDecodedDBOps: false,
      decodedDBOps: []
    }
  }

  get displayedDBOps(): DbOp[] {
    if (this.state.decodedDBOps.length > 0) {
      return this.state.decodedDBOps
    }

    if (this.props.dbops) {
      return this.props.dbops
    }

    return []
  }

  hasDBOpsToDecode() {
    return !this.state.isDecodedDBOps && this.props.dbops
  }

  onChangeContent = (currentTab: string) => {
    this.setState({ currentTab }, () => {
      if (
        this.state.currentTab === PILL_TAB_VALUES.DBOPS &&
        this.hasDBOpsToDecode() &&
        this.props.blockNum
      ) {
        decodeDBOps(this.props.dbops!, this.props.blockNum, (decodedDBOps: DbOp[]) => {
          this.setState((prevState) => ({
            currentTab: prevState.currentTab,
            decodedDBOps,
            isDecodedDBOps: true
          }))
        })
      }
    })
  }

  renderReceiverInfo() {
    if (this.props.traceInfo) {
      return (
        <DetailLine compact={true} label={t("transaction.pill.receiver")}>
          <SearchShortcut query={`receiver:${this.props.traceInfo.receiver}`}>
            <MonospaceTextLink to={Links.viewAccount({ id: this.props.traceInfo.receiver })}>
              {this.props.traceInfo.receiver}
            </MonospaceTextLink>
          </SearchShortcut>
        </DetailLine>
      )
    }
    return null
  }

  renderAccountLink() {
    let query = `account:${this.props.action.account}`
    if (this.props.traceInfo) {
      query = `${query} receiver:${this.props.traceInfo.receiver}`
    }
    return (
      <DetailLine compact={true} label={t("transaction.pill.account")}>
        <SearchShortcut query={query}>
          <MonospaceTextLink to={Links.viewAccount({ id: this.props.action.account })}>
            {this.props.action.account}
          </MonospaceTextLink>{" "}
        </SearchShortcut>
      </DetailLine>
    )
  }

  renderActionName() {
    let query = `action:${this.props.action.name} account:${this.props.action.account}`
    if (this.props.traceInfo) {
      query = `${query} receiver:${this.props.traceInfo.receiver}`
    }
    return (
      <DetailLine compact={true} label={t("transaction.pill.action_name")}>
        <SearchShortcut query={query}>
          <Text>{this.props.action.name}</Text>
        </SearchShortcut>
      </DetailLine>
    )
  }

  renderAuthorizations() {
    const authorizations = (this.props.action.authorization || []).map(
      (entry: Authorization, index: number) => {
        return (
          <Cell key={index}>
            <SearchShortcut query={`auth:${entry.actor}@${entry.permission}`}>
              <AutorizationBox authorization={entry} />
            </SearchShortcut>
          </Cell>
        )
      }
    )

    return (
      <DetailLine compact={true} label={t("transaction.pill.authorization")}>
        <Text>{authorizations}</Text>
      </DetailLine>
    )
  }

  renderDisplayFullContentButton() {
    return this.props.displayFullContentButton ? (
      <Cell float="right">
        <LinkStyledText color={theme.colors.link} onClick={() => this.props.onDisplayFullContent()}>
          Show Full Content
        </LinkStyledText>
      </Cell>
    ) : null
  }

  renderTabContent() {
    if (this.state.currentTab === PILL_TAB_VALUES.GENERAL) {
      return (
        <ContentWrapper>
          {this.renderReceiverInfo()}
          {this.renderAccountLink()}
          {this.renderActionName()}
          {this.renderAuthorizations()}
        </ContentWrapper>
      )
    }

    if (this.state.currentTab === PILL_TAB_VALUES.JSON_DATA) {
      return (
        <RawWrapper px="24px" py="10px">
          {this.renderDisplayFullContentButton()}
          <JsonWrapper>{JSON.stringify(this.props.data, null, "   ")}</JsonWrapper>
        </RawWrapper>
      )
    }

    if (this.state.currentTab === PILL_TAB_VALUES.HEX_DATA) {
      return (
        <RawWrapper px="24px" py="10px">
          <JsonWrapper>{this.props.action.hex_data}</JsonWrapper>
        </RawWrapper>
      )
    }

    if (this.state.currentTab === PILL_TAB_VALUES.RAMOPS) {
      return (
        <ContentWrapper>
          <RamUsage type="detailed" ramops={this.props.ramops || []} />
        </ContentWrapper>
      )
    }

    if (this.state.currentTab === PILL_TAB_VALUES.DBOPS) {
      return (
        <ContentWrapper>
          <DBOperations tableops={this.props.tableops || []} dbops={this.displayedDBOps || []} />
        </ContentWrapper>
      )
    }

    if (this.state.currentTab === PILL_TAB_VALUES.CONSOLE) {
      return (
        <ContentWrapper>
          <JsonWrapper>{this.props.console!.replace(/\\r/g, "")}</JsonWrapper>
        </ContentWrapper>
      )
    }

    return null
  }

  render() {
    const tabs = [...this.PILL_TABS]

    if (this.props.action.data) {
      tabs.push({ label: t("transaction.pill.jsonData"), value: PILL_TAB_VALUES.JSON_DATA })
    } else if (this.props.action.hex_data) {
      tabs.push({ label: t("transaction.pill.hexData"), value: PILL_TAB_VALUES.HEX_DATA })
    }

    if (this.props.dbops && this.props.dbops.length > 0) {
      tabs.push({ label: t("transaction.pill.dbOps"), value: PILL_TAB_VALUES.DBOPS })
    }

    if (this.props.ramops && this.props.ramops.length > 0) {
      tabs.push({ label: t("transaction.pill.ramOps"), value: PILL_TAB_VALUES.RAMOPS })
    }

    if (this.props.console && this.props.console.length > 0) {
      tabs.push({ label: t("transaction.pill.console"), value: PILL_TAB_VALUES.CONSOLE })
    }

    return [
      <VerticalTabs key={1} tabData={tabs} onSelectTab={this.onChangeContent} />,
      <TabContentWrapper key={2}>{this.renderTabContent()}</TabContentWrapper>
    ]
  }
}
