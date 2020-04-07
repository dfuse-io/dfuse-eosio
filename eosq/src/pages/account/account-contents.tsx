import { observer } from "mobx-react"
import * as React from "react"
import { styled } from "../../theme"
import { Cell } from "../../atoms/ui-grid/ui-grid.component"
import { ContentLoaderComponent } from "../../components/content-loader/content-loader.component"
import { Account } from "../../models/account"
import { TabbedPanel } from "../../atoms/tabbed-panel/tabbed-panel"
import { AccountTransactions } from "./transactions/account-transactions"
import { AccountVotes } from "./votes/account-votes"
import { Links } from "../../routes"
import { fetchContractAbi } from "../../services/contract-table"
import { AccountTables } from "./tables/account-tables"
import { t } from "i18next"
import { RouteComponentProps } from "react-router"
import { AccountAbi } from "./abi/account-abi"
import { Abi } from "@dfuse/client"

const PanelContentWrapper: React.ComponentType<any> = styled(Cell)`
  width: 100%;
`

interface Props extends RouteComponentProps<any> {
  account: Account
  currentTab: string
}

type State = { currentTab: string; isContract: boolean; abi: Abi | null }

@observer
export class AccountContents extends ContentLoaderComponent<Props, State> {
  limit = 50

  constructor(props: Props) {
    super(props)

    this.state = {
      currentTab: props.currentTab || "transactions",
      isContract: props.currentTab === "tables" || false,
      abi: null
    }
  }

  componentDidMount() {
    fetchContractAbi(this.props.account.account_name).then((data: { abi: Abi } | undefined) => {
      if (data && data.abi) {
        this.setState({ isContract: true, abi: data.abi })
      }
    })
  }

  componentDidUpdate(prevProps: Readonly<Props>) {
    if (this.props.account.account_name !== prevProps.account.account_name) {
      fetchContractAbi(this.props.account.account_name).then((data: { abi: Abi } | undefined) => {
        if (data && data.abi) {
          this.setState({ isContract: true, abi: data.abi })
        }
      })
    }
  }

  renderTabContent = () => {
    switch (this.state.currentTab) {
      case "transactions":
        return (
          <AccountTransactions
            account={this.props.account}
            location={this.props.location}
            history={this.props.history}
            match={this.props.match}
          />
        )
      case "tables":
        return (
          <AccountTables
            abi={this.state.abi}
            accountName={this.props.account.account_name}
            location={this.props.location}
            history={this.props.history}
          />
        )
      case "votes":
        return <AccountVotes account={this.props.account} />
      case "abi":
        return <AccountAbi abi={this.state.abi} />
      default:
        return (
          <AccountTransactions
            account={this.props.account}
            location={this.props.location}
            history={this.props.history}
            match={this.props.match}
          />
        )
    }
  }

  onSelectTab = (currentTab: string) => {
    this.setState({ currentTab })
    this.props.history.replace(
      Links.viewAccountTabs({ id: this.props.account.account_name, currentTab })
    )
  }

  render() {
    let tabData = [
      { label: "transactions", value: t("account.tabs.transactions") },
      { label: "votes", value: t("account.tabs.vote_title") }
    ]

    if (this.state.isContract) {
      tabData = tabData.concat([
        { label: "tables", value: t("account.tabs.tables") },
        { label: "abi", value: "ABI" }
      ])
    }

    return (
      <Cell mt={[3]}>
        <PanelContentWrapper>
          <TabbedPanel
            selected={this.state.currentTab}
            fontSize={[1, 2, 3]}
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
