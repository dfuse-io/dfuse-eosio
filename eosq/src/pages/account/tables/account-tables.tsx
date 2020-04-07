import H from "history"
import { observer } from "mobx-react"
import * as React from "react"
import { Grid } from "../../../atoms/ui-grid/ui-grid.component"
import { ContentLoaderComponent } from "../../../components/content-loader/content-loader.component"
import { AbiLoader } from "../../../services/abi-loader"
import { AccountTableSearch } from "./account-table-search"
import { AccountTableView } from "./account-table-view"
import { Abi } from "@dfuse/client"

interface Props {
  abi: Abi | null
  accountName: string
  location: H.Location
  history: H.History
}

@observer
export class AccountTables extends ContentLoaderComponent<Props> {
  abiLoader?: AbiLoader

  constructor(props: Props) {
    super(props)

    if (this.props.abi) {
      this.abiLoader = new AbiLoader(this.props.abi)
    }
  }

  componentDidUpdate(prevProps: Props) {
    if (prevProps.abi !== this.props.abi && this.props.abi) {
      this.abiLoader = new AbiLoader(this.props.abi)
      this.forceUpdate()
    }
  }

  renderContent = () => {
    if (!this.abiLoader) {
      return <div />
    }

    return (
      <Grid>
        <AccountTableSearch
          accountName={this.props.accountName}
          location={this.props.location}
          history={this.props.history}
          abiLoader={this.abiLoader}
        />
        {this.renderTableView()}
      </Grid>
    )
  }

  renderTableView = () => {
    if (!this.abiLoader) {
      return <div />
    }

    return <AccountTableView history={this.props.history} location={this.props.location} />
  }

  render() {
    return <Grid>{this.renderContent()}</Grid>
  }
}
