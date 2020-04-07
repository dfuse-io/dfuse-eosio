import { observer } from "mobx-react"
import * as React from "react"
import { styled } from "../../theme"
import { Cell, Grid } from "../../atoms/ui-grid/ui-grid.component"
import { TabbedPanel } from "../../atoms/tabbed-panel/tabbed-panel"
import H from "history"
import { t } from "i18next"
import { TransactionActions } from "../../components/transaction/transaction-actions.component"
import { convertDTrxOpsToDeferredOperations } from "../../helpers/legacy.helpers"
import { TransactionRamUsage } from "./summary/transaction-ram-usage"
import { TransactionLifecycleWrap } from "../../services/transaction-lifecycle"

const PanelContentWrapper: React.ComponentType<any> = styled(Grid)`
  width: 100%;
`

interface Props {
  lifecycleWrap: TransactionLifecycleWrap
  currentTab: string
  history: H.History
  location: H.Location
}

type State = { currentTab: string }

@observer
export class TransactionContents extends React.Component<Props, State> {
  constructor(props: Props) {
    super(props)

    this.state = {
      currentTab: props.currentTab || "actions"
    }
  }

  renderActions() {
    if (this.props.lifecycleWrap.hasActions) {
      return (
        <Cell minHeight="250px">
          <TransactionActions
            dbops={this.props.lifecycleWrap.lifecycle.dbops || []}
            ramops={this.props.lifecycleWrap.lifecycle.ramops || []}
            actionTraces={this.props.lifecycleWrap.actionTraces}
            tableops={this.props.lifecycleWrap.lifecycle.tableops}
            actions={this.props.lifecycleWrap.actions}
            deferredOperations={convertDTrxOpsToDeferredOperations(
              this.props.lifecycleWrap.lifecycle.id,
              this.props.lifecycleWrap.lifecycle.dtrxops || []
            )}
            pageContext={{
              blockNum: this.props.lifecycleWrap.blockNum
            }}
            creationTree={this.props.lifecycleWrap.lifecycle.creation_tree}
          />
        </Cell>
      )
    }

    return null
  }

  renderTabContent = () => {
    switch (this.state.currentTab) {
      case "actions":
        return this.renderActions()
      case "ramUsage":
        return <TransactionRamUsage transactionLifeCycle={this.props.lifecycleWrap.lifecycle} />

      default:
        return this.renderActions()
    }
  }

  onSelectTab = (currentTab: string) => {
    this.setState({ currentTab })
  }

  render() {
    let totalCount = this.props.lifecycleWrap.totalActionCount

    if (totalCount === 0) {
      totalCount = this.props.lifecycleWrap.actions.length
    }

    const tabData = [
      { label: "actions", value: `${t("transaction.traces.title")} (${totalCount})` }
    ]

    if (
      this.props.lifecycleWrap.lifecycle.ramops &&
      this.props.lifecycleWrap.lifecycle.ramops.length > 0
    ) {
      tabData.push({ label: "ramUsage", value: t("transaction.ramUsage.title") })
    }

    return (
      <Cell mt={[3]}>
        <PanelContentWrapper>
          <TabbedPanel
            selected={this.state.currentTab}
            fontSize={[2, 3]}
            tabData={tabData}
            onSelect={this.onSelectTab}
          >
            {this.renderTabContent()}
          </TabbedPanel>
        </PanelContentWrapper>
      </Cell>
    )
  }
}
