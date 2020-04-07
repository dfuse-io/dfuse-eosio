import { t } from "i18next"
import { observer } from "mobx-react"
import numeral from "numeral"
import * as React from "react"
import { DonutChart } from "../../../atoms/pie-chart/donut-chart"
import { DonutChartLegend } from "../../../atoms/pie-chart/donut-legend"
import { Text } from "../../../atoms/text/text.component"
import { Cell, Grid } from "../../../atoms/ui-grid/ui-grid.component"
import { Account } from "../../../models/account"
import { theme, styled } from "../../../theme"
import { fetchContractTableRowsFromEOSWS } from "../../../services/contract-table"
import {
  AccountResources,
  getAccountResources,
  getPieChartParams,
  PieChartParams,
  StakeDetail
} from "../../../helpers/account.helpers"
import { NBSP } from "../../../helpers/formatters"
import { DataLoading } from "../../../atoms/data-loading/data-loading.component"
import { MonospaceText } from "../../../atoms/text-elements/misc"
import { SearchShortcut } from "../../../components/search-shortcut/search-shortcut"
import { UiToolTip } from "../../../atoms/ui-tooltip/ui-tooltip"

interface Props {
  account: Account
}

const HidableContainer: React.ComponentType<any> = styled.div`
  @media (max-width: 850px) {
    width: 100%;
    text-align: center;
  }
`

interface State {
  stakeDetails: StakeDetail[]
  stakeLoaded: boolean
}

const ToolTipUl: React.ComponentType<any> = styled.ul`
  list-style: none;
  padding-left: 10px;
  margin-bottom: 0;
`

@observer
export class AccountPieChart extends React.Component<Props, State> {
  state: State = { stakeDetails: [], stakeLoaded: false }

  componentDidMount() {
    this.fetchDelband()
  }

  componentDidUpdate(oldProps: Props) {
    if (this.props.account.account_name !== oldProps.account.account_name) {
      // eslint-disable-next-line react/no-did-update-set-state
      this.setState({
        stakeDetails: [],
        stakeLoaded: false
      })

      this.fetchDelband()
    }
  }

  fetchDelband() {
    fetchContractTableRowsFromEOSWS({
      code: "eosio",
      json: true,
      limit: -1,
      scope: this.props.account.account_name,
      table: "delband",
      table_key: ""
    }).then((output: any) => {
      output = output as { more: boolean; rows: StakeDetail[] }
      this.setState({
        stakeDetails: output && output.rows ? output.rows.map((row: any) => row.json) : [],
        stakeLoaded: true
      })
    })
  }

  renderPieChart(pieChartParams: PieChartParams) {
    return (
      <Cell pb={[4, 0, 0]} alignSelf="center">
        <HidableContainer>
          <DonutChart
            id="account-pie-chart"
            centerContent={pieChartParams.pieChartCenter}
            params={{
              data: pieChartParams.pieChartDataForPie,
              colors: pieChartParams.pieChartColorsForPie
            }}
          />
        </HidableContainer>
      </Cell>
    )
  }

  renderToolTip(accountResources: AccountResources, type: string): JSX.Element {
    return (
      <Cell p={[3]} width="100%">
        <Cell pb={[1]}>
          Self Staked: {accountResources[type].selfStaked.toFixed(4)}
          {NBSP}EOS
        </Cell>
        <Cell pb={[1]}>
          Staked From Others: {accountResources[type].stakedFromOthers.toFixed(4)}
          {NBSP}EOS
        </Cell>

        <Cell pb={[1]}>
          Staked To Others: {accountResources[type].stakedToOthers.toFixed(4)}
          {NBSP}EOS
        </Cell>
        <ToolTipUl>
          {accountResources.stakes
            .filter((stake: StakeDetail) => stake[`${type}_weight`] !== "0.0000 EOS")
            .map((stake: StakeDetail, index: number) => {
              if (index < 20) {
                return (
                  <li key={index}>
                    to{" "}
                    <MonospaceText display="inline-block" color={theme.colors.primary}>
                      {stake.to}
                    </MonospaceText>
                    : {stake[`${type}_weight`]}
                  </li>
                )
              }
              return null
            })}
        </ToolTipUl>
        {accountResources.stakes.length > 20 ? (
          <Cell width="100%" mt={[1]} textAlign="right">
            + {accountResources.stakes.length - 20} more
          </Cell>
        ) : null}
      </Cell>
    )
  }

  renderTooltipWrapper = (value: number, unit: string, toolTip?: JSX.Element) => {
    if (toolTip) {
      return (
        <UiToolTip>
          <Cell alignSelf="center" lineHeight="20px">
            <Text
              borderBottom={`2px dotted ${theme.colors.text}`}
              fontSize={[3, 2]}
              alignSelf="center"
              lineHeight="20px"
              fontWeight="bold"
            >
              {numeral(value).format("0,0.0000")} {unit}
            </Text>
          </Cell>
          {toolTip}
        </UiToolTip>
      )
    }
    return (
      <Text fontSize={[3, 2]} alignSelf="center" lineHeight="20px" fontWeight="bold">
        {numeral(value).format("0,0.0000")} {unit}
      </Text>
    )
  }

  renderSearchShortcutWrapper = (contents: JSX.Element, query: string) => {
    return (
      <SearchShortcut position="left" query={query}>
        {contents}
      </SearchShortcut>
    )
  }

  renderWrapper = (accountResources: AccountResources, type: string, value: number) => {
    let contents: JSX.Element
    let query = ""
    const accountName = this.props.account.account_name
    switch (type) {
      case "cpu":
        contents = this.renderTooltipWrapper(
          value,
          accountResources.unit,
          this.renderToolTip(accountResources, type)
        )
        query = `(action:delegatebw OR action:undelegatebw) receiver:eosio data.receiver:${accountName}`
        return this.renderSearchShortcutWrapper(contents, query)
      case "net":
        contents = this.renderTooltipWrapper(
          value,
          accountResources.unit,
          this.renderToolTip(accountResources, type)
        )
        query = `(action:delegatebw OR action:undelegatebw) receiver:eosio data.receiver:${accountName}`
        return this.renderSearchShortcutWrapper(contents, query)
      case "refund":
        contents = this.renderTooltipWrapper(value, accountResources.unit)
        query = `receiver:eosio action:refund data.owner:${accountName}`
        return this.renderSearchShortcutWrapper(contents, query)
      case "REX":
        contents = this.renderTooltipWrapper(value, accountResources.unit)
        query = `account:eosio (action:rentcpu OR action:rentnet OR action:deposit OR action:withdraw OR action:rentram OR action:updaterex OR action:buyrex OR action:sellrex OR action:cnclrexorder) (data.owner:${accountName} OR data.from:${accountName} OR data.receiver:${accountName})`
        return this.renderSearchShortcutWrapper(contents, query)
      case "REX_FUNDS":
        contents = this.renderTooltipWrapper(value, accountResources.unit)
        query = `account:eosio (action:rentcpu OR action:rentnet OR action:deposit OR action:withdraw OR action:rentram OR action:updaterex OR action:buyrex OR action:sellrex OR action:cnclrexorder) (data.owner:${accountName} OR data.from:${accountName} OR data.receiver:${accountName})`
        return this.renderSearchShortcutWrapper(contents, query)
      case "available_funds":
        contents = this.renderTooltipWrapper(value, accountResources.unit)
        query = `receiver:eosio.token action:transfer (data.from:${accountName} OR data.to:${accountName})`
        return this.renderSearchShortcutWrapper(contents, query)
      default:
        throw new Error(`Wrong type: ${type}`)
    }
  }

  render() {
    if (!this.state.stakeLoaded) {
      return (
        <Cell alignSelf="center">
          <DataLoading />
        </Cell>
      )
    }
    const accountResources = getAccountResources(this.props.account, this.state.stakeDetails)
    const pieChartParams = getPieChartParams(accountResources, this.renderWrapper)

    return (
      <Cell pt={[4]}>
        <Grid
          gridTemplateColumns={["1fr", "1fr 1fr"]}
          borderBottom={[`0px solid ${theme.colors.grey4}`, `1px solid ${theme.colors.grey4}`]}
          mb={[1, 2]}
          pb={[1, 1]}
        >
          <Cell alignSelf="end">
            <Text fontSize={[3]} fontWeight="700">
              {t("account.pie_chart.legendTitle")}
            </Text>
          </Cell>
          <Cell
            gridRow={["2", "1"]}
            gridColumn={["1", "2"]}
            alignSelf={["left", "right"]}
            justifySelf={["left", "right"]}
          >
            <Text fontSize={[4, 4]}>
              {numeral(accountResources.totalOwnerShip).format("0,0.0000")} {accountResources.unit}
            </Text>
          </Cell>
        </Grid>

        <Grid
          gridTemplateColumns={["1fr", "auto 1fr"]}
          borderBottom={["0px solid", "0px solid"]}
          borderColor="border"
          pb={[20, 0]}
        >
          {this.renderPieChart(pieChartParams)}
          <Cell pt="14px" alignSelf="center">
            <DonutChartLegend
              id="account-pie-chart-legend"
              params={{ data: pieChartParams.pieChartData, colors: pieChartParams.pieChartColors }}
              units={accountResources.unit}
            />
          </Cell>
        </Grid>
      </Cell>
    )
  }
}
