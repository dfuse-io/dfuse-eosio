import { observer } from "mobx-react"
import * as React from "react"
import { Redirect, RouteComponentProps } from "react-router-dom"
import { Account } from "../../models/account"
import { Links } from "../../routes"

import { ContentLoaderComponent } from "../../components/content-loader/content-loader.component"
import { PageContainer } from "../../components/page-container/page-container"
import { ErrorData, Stream } from "@dfuse/client"

// temp ignore for dev

import { DataLoading, BULLET } from "@dfuse/explorer"
import { CustomTitleBanner } from "../../atoms/panel/custom-title-banner"
import { AccountSummary } from "./summary/account-summary"
import { AccountTitle } from "./summary/account-title"
import { metricsStore, voteStore } from "../../stores"
import { AccountContents } from "./account-contents"
import { Cell } from "../../atoms/ui-grid/ui-grid.component"
import { NavigationButtons } from "../../atoms/navigation-buttons/navigation-buttons"
import { Vote } from "../../models/vote"
import { getRankInfo } from "../../helpers/account.helpers"
import { registerAccountDetailsListeners } from "../../streams/account-listeners"

import { FormattedError } from "../../components/formatted-error/formatted-error"

interface Props extends RouteComponentProps<any> {}

interface State {
  account?: Account
  fetchError?: ErrorData
}

@observer
export class AccountDetail extends ContentLoaderComponent<Props, State> {
  state: State = {}

  accountStream?: Stream
  voteStream?: Stream

  componentDidMount = async () => {
    await this.registerStreams()
    this.changeDocumentTitle()
  }

  changeDocumentTitle() {
    document.title = `${this.props.match.params.id} ${BULLET} account ${BULLET} eosq`
  }

  componentDidUpdate = async (prevProps: Props) => {
    if (prevProps.match.params.id !== this.props.match.params.id) {
      await this.unregisterStreams()
      await this.registerStreams()

      this.changeDocumentTitle()
    }
  }

  componentWillUnmount = async () => {
    await this.unregisterStreams()
  }

  registerStreams = async () => {
    const streams = await registerAccountDetailsListeners(
      this.props.match.params.id,
      metricsStore.headBlockNum,
      (account: Account) => {
        this.setState({ account })
      },
      (error: ErrorData) => {
        this.setState({ fetchError: error })
      }
    )

    this.voteStream = streams.voteStream
    this.accountStream = streams.voteStream
  }

  unregisterStreams = async () => {
    if (this.accountStream !== undefined) {
      await this.accountStream.close()
      this.accountStream = undefined
    }

    if (this.voteStream !== undefined) {
      await this.voteStream.close()
      this.voteStream = undefined
    }
  }

  renderNotFound = () => {
    return <Redirect to={Links.notFound()} />
  }

  renderBannerLeft(account: Account) {
    return <AccountTitle account={account} />
  }

  getNextProducer() {
    if (this.state.account && voteStore.votes) {
      const { rank } = getRankInfo(this.state.account!, voteStore.votes)

      return voteStore.votes.find((vote: Vote, index: number) => {
        return rank + 1 === index + 1
      })
    }

    return undefined
  }

  getPrevProducer(): Vote | undefined {
    if (this.state.account && voteStore.votes) {
      const { rank } = getRankInfo(this.state.account!, voteStore.votes)
      return voteStore.votes.find((vote: Vote, index: number) => {
        return rank - 1 === index + 1
      })
    }
    return undefined
  }

  onNext = () => {
    window.location.href = Links.viewAccount({ id: this.getNextProducer()!.producer })
  }

  onPrev = () => {
    window.location.href = Links.viewAccount({ id: this.getPrevProducer()!.producer })
  }

  renderBannerRight() {
    if (this.state.account && getRankInfo(this.state.account, voteStore.votes).rank > 0) {
      return (
        <NavigationButtons
          onNext={this.onNext}
          onPrev={this.onPrev}
          showNext={!!this.getNextProducer()}
          showPrev={!!this.getPrevProducer()}
          showFirst={false}
        />
      )
    }
    return <span />
  }

  render() {
    if (this.state.fetchError) {
      return (
        <PageContainer>
          <FormattedError error={this.state.fetchError} title="Error fetching account" />
        </PageContainer>
      )
    }

    if (!this.state.account) {
      return <DataLoading />
    }

    return (
      <Cell>
        <PageContainer>
          <CustomTitleBanner
            contentLeft={this.renderBannerLeft(this.state.account)}
            contentRight={this.renderBannerRight()}
          />
          <AccountSummary account={this.state.account} />
        </PageContainer>
        <PageContainer>
          <AccountContents
            history={this.props.history}
            location={this.props.location}
            match={this.props.match}
            currentTab={this.props.match.params.currentTab}
            account={this.state.account}
          />
        </PageContainer>
      </Cell>
    )
  }
}
