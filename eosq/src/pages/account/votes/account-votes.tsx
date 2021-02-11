import * as React from "react"
import { Account } from "../../../models/account"
import { addIndex, map } from "ramda"
import { MonospaceTextLink } from "../../../atoms/text-elements/misc"
import { Links } from "../../../routes"
import { styled } from "../../../theme"
import { Cell, Grid } from "../../../atoms/ui-grid/ui-grid.component"
import { StatusBar } from "../../../atoms/status-bar/status-bar"
import { Text, TextLink } from "../../../atoms/text/text.component"
import { t } from "i18next"
import { formatPercentage, NBSP, DetailLine } from "@dfuse/explorer"
import numeral from "numeral"

import { calculateVoteStrength } from "./vote.helpers"
import { Stream } from "@dfuse/client"
import { registerAccountDetailsListeners } from "../../../streams/account-listeners"
import { metricsStore } from "../../../stores"
import { Config } from "../../../models/config"

interface Props {
  account: Account
}

const VotedAccount: React.ComponentType<any> = styled(MonospaceTextLink)`
  padding: 10px;
  background-color: ${(props) => props.theme.colors.accountVoteBtn};
  margin-bottom: 10px;
  display: inline-block;
  margin-bottom: 17px;
  text-align: center;

  &:hover {
    background-color: ${(props) => props.theme.colors.accountVoteBtnOver};
    color: ${(props) => props.theme.colors.accountVoteBtnOverLink};
  }
`

const VoteBarText: React.ComponentType<any> = styled(Text)`
  color: #ffffff !important;
  position: absolute;
  width: max-content;
`

interface State {
  voteStrength: number
  proxyAccount?: Account
}

export class AccountVotes extends React.Component<Props, State> {
  state: State = { voteStrength: 0, proxyAccount: undefined }

  accountStream?: Stream

  registerStreams = async (accountName: string) => {
    const streams = await registerAccountDetailsListeners(
      accountName,
      metricsStore.headBlockNum,
      (account: Account) => {
        this.setState({
          voteStrength: calculateVoteStrength(
            this.props.account,
            this.props.account.voter_info.staked
          ),
          proxyAccount: account,
        })
      },
      () => {
        this.setState({
          voteStrength: 0,
          proxyAccount: undefined,
        })
      }
    )

    this.accountStream = streams.accountStream
  }

  componentDidMount = async () => {
    const { voter_info: voterInfo } = this.props.account
    if (voterInfo.proxy && voterInfo.proxy.length > 0) {
      await this.registerStreams(voterInfo.proxy)
    } else if (voterInfo.producers && voterInfo.producers.length > 0) {
      this.setState({
        voteStrength: calculateVoteStrength(
          this.props.account,
          this.props.account.voter_info.staked
        ),
      })
    }
  }

  componentDidUpdate = async (prevProps: Readonly<Props>) => {
    if (prevProps.account.account_name !== this.props.account.account_name) {
      const { voter_info: voterInfo } = this.props.account
      if (voterInfo.proxy && voterInfo.proxy.length > 0) {
        await this.unregisterStreams()
        await this.registerStreams(voterInfo.proxy)
      } else if (voterInfo.producers && voterInfo.producers.length > 0) {
        // eslint-disable-next-line react/no-did-update-set-state
        this.setState({
          voteStrength: calculateVoteStrength(
            this.props.account,
            this.props.account.voter_info.staked
          ),
        })
      }
    }
  }

  componentWillUnmount = async () => {
    await this.unregisterStreams()
  }

  unregisterStreams = async () => {
    if (this.accountStream) {
      await this.accountStream.close()
      this.accountStream = undefined
    }
  }

  renderAccounts = (accounts: string[]) => {
    const mapIndexed = addIndex<string>(map)

    return mapIndexed(
      (account: string, index: number) => this.renderAccountName(account, index),
      accounts || []
    )
  }

  renderAccountName = (account: string, index: number) => {
    return (
      <VotedAccount
        mr={[0, 3]}
        width={["100%", "auto", "auto"]}
        key={index}
        fontSize={[2, 3]}
        to={Links.viewAccount({ id: account })}
      >
        {account}
      </VotedAccount>
    )
  }

  renderProxyAccountTitle = () => {
    if (this.props.account.voter_info.is_proxy) {
      return (
        <Cell>
          <Text alignSelf="left" px={[4]} pb={[2]} color="header" fontSize={[25]} fontWeight="500">
            Proxy account
          </Text>
        </Cell>
      )
    }
    return null
  }

  renderStatusBar = () => {
    if (this.props.account.voter_info.is_proxy) {
      return null
    }
    return (
      <Cell mt={[4]}>
        <StatusBar
          large={true}
          content={[this.state.voteStrength]}
          total={1}
          bg="barVoteBg"
          color="barVote"
        >
          <Grid gridTemplateColumns={["auto 1fr", "1fr 1fr"]} height="100%">
            <VoteBarText alignSelf="center" pl={[2, 3]} fontSize={[1, 2]}>
              <Text color="#ffffff" display="inline-block" fontWeight="bold">
                {t("account.summary.voter_info.labels.strength")}
                {NBSP}
              </Text>
              {formatPercentage(this.state.voteStrength)} ,{NBSP}
              {t("account.summary.voter_info.labels.nextDecay")}
            </VoteBarText>
          </Grid>
        </StatusBar>
      </Cell>
    )
  }

  renderVoteWeights() {
    if (this.props.account.voter_info.is_proxy && this.props.account.voter_info.staked === 0) {
      return null
    }
    const voteWeight = this.props.account.voter_info.staked / 10000
    const decayedVoteWeight = voteWeight * this.state.voteStrength

    return (
      <Cell alignSelf="left" justifySelf="left" gridColumn={["1"]} mt={[4]}>
        <DetailLine
          mb={2}
          variant="compact"
          label={t("account.summary.voter_info.labels.vote_weight")}
        >
          {numeral(voteWeight).format(Config.chain_core_asset_format)}{" "}
          {Config.chain_core_symbol_code}
        </DetailLine>
        <DetailLine
          mb={2}
          variant="compact"
          label={t("account.summary.voter_info.labels.decayed_vote_weight")}
        >
          {numeral(decayedVoteWeight).format(Config.chain_core_asset_format)}{" "}
          {Config.chain_core_symbol_code}
        </DetailLine>
      </Cell>
    )
  }

  render() {
    if (this.state.voteStrength === 0) {
      return (
        <Cell p={[4]}>
          <Text color="text" display="inline-block" fontWeight="bold">
            {t("account.summary.voter_info.noVotes")}
          </Text>
        </Cell>
      )
    }

    const account = this.state.proxyAccount || this.props.account
    return (
      <Cell pt={[2]} pb={[4]}>
        {this.renderProxyAccountTitle()}
        <Cell px={[4]} pb={[5]}>
          {this.renderVoteWeights()}
          {this.renderStatusBar()}
          {this.state.proxyAccount ? (
            <Text fonSize={[3]} pt={[5]}>
              <Cell>
                <Text display="inline-block" fontSize={[3]}>
                  Proxied to:{" "}
                </Text>
                {NBSP}
                <TextLink
                  fontSize={[3]}
                  to={Links.viewAccount({
                    id: this.state.proxyAccount.account_name,
                  })}
                >
                  {this.state.proxyAccount.account_name}
                </TextLink>
              </Cell>
            </Text>
          ) : null}
          <Text pt={[4]} fontWeight={["800"]} fontSize={[2]}>
            {this.state.proxyAccount
              ? t("account.summary.voter_info.labels.vote_for_proxy")
              : t("account.summary.voter_info.labels.vote_for_producers")}
          </Text>
          <Cell pt={[2, 3]}>{this.renderAccounts(account.voter_info.producers)}</Cell>
        </Cell>
      </Cell>
    )
  }
}
