import { t } from "i18next"
import { observer } from "mobx-react"
import * as React from "react"
import { StatusBar } from "../../../atoms/status-bar/status-bar"
import { Text } from "../../../atoms/text/text.component"
import { Cell, Grid } from "../../../atoms/ui-grid/ui-grid.component"
import {
    extractValueWithUnits,
    formatBytes,
//ultra-andrey-bezrukov --- BLOCK-80 Integrate ultra power into dfuse and remove rex related tables
//    formatMicroseconds,
    INFINITY, microSecondsToSeconds,
    NBSP
} from "@dfuse/explorer"
import { Account } from "../../../models/account"
import numeral from "numeral"
import { StatusWidget } from "../../../atoms/status-widget/status-widget"
import { theme, styled } from "../../../theme"
import { SearchShortcut } from "../../../components/search-shortcut/search-shortcut"
import { Config } from '../../../models/config'

const AccountStatusBarsContainer: React.ComponentType<any> = styled.div`
  margin-top: 15px;
`

interface Props {
  account: Account
}

//ultra-andrey-bezrukov --- BLOCK-80 Integrate ultra power into dfuse and remove rex related tables
  function formatMicroseconds(micro: number) {
  if(micro == -1) {
    // special case. Treat it as unlimited
    return "unlimited";
  }
  let unit: string = "us";

  if( micro > 1000000*60*60 ) {
      micro /= 1000000*60*60;
      unit = "hr";
  }
  else if( micro > 1000000*60 ) {
      micro /= 1000000*60;
      unit = "min";
  }
  else if( micro > 1000000 ) {
      micro /= 1000000;
      unit = "sec";
  }
  else if( micro > 1000 ) {
      micro /= 1000;
      unit = "ms";
  }
  return micro.toFixed(1)+unit;
}

@observer
export class AccountStatusBars extends React.Component<Props> {
  isInfiniteResources(numerator: number, denominator: number) {
    return Math.round(numerator) === -1 || Math.round(denominator) === -1
  }

  renderRatioText(formattedText: string, infinite: boolean) {
    return (
      <Text fontWeight="bold" display="inline-block" fontSize={[2]}>
        {infinite ? INFINITY : formattedText}
      </Text>
    )
  }

  renderByteRatio(numerator: number, denominator: number) {
    const infinite = this.isInfiniteResources(numerator, denominator)
    return (
      <Cell>
        {this.renderRatioText(formatBytes(numerator), infinite)} {this.renderUsed()}/{NBSP}
        {this.renderRatioText(formatBytes(denominator), infinite)}
      </Cell>
    )
  }

  renderUsed() {
    return (
      <Text fontFamily="Roboto" fontWeight="400" color={theme.colors.grey5} display="inline-block">
        {" "}
        {t("account.status.used")}
        {NBSP}
      </Text>
    )
  }

  renderTimeRatio(numerator: number, denominator: number) {
    const infinite = this.isInfiniteResources(numerator, denominator)
    return (
      <Cell>
        {this.renderRatioText(formatMicroseconds(numerator), infinite)} {this.renderUsed()}/{NBSP}
        {this.renderRatioText(formatMicroseconds(denominator), infinite)}
      </Cell>
    )
  }

  renderStakeDetails(
    title: string,
    total: string,
    staked: string,
    other: number,
  ): JSX.Element {
    return (
      <Grid mt={[3]} gridTemplateColumns={["1fr"]}>
        <Cell mb={[1]} borderBottom={`1px dotted ${theme.colors.grey4}`}>
          <Text fontSize={[2]} color={theme.colors.grey5}>
            {t("account.summary.staked_by")}
          </Text>
        </Cell>

        <Grid mb={[1]} gridTemplateColumns={["auto auto"]}>
          <Cell alignSelf="left" justifySelf="left">
            {t("account.summary.self")}
          </Cell>
          <Cell alignSelf="right" justifySelf="right">
            {staked} {Config.chain_core_symbol_code}
          </Cell>
        </Grid>
        {other === 0.0 ? null : (
          <Grid mb={[1]} gridTemplateColumns={["auto auto"]}>
            <Cell alignSelf="left" justifySelf="left">
              {t("account.summary.tooltip.other")}
            </Cell>
            <Cell alignSelf="right" justifySelf="right">
              {numeral(other).format(Config.chain_core_asset_format)} {Config.chain_core_symbol_code}
            </Cell>
          </Grid>
        )}
        <Grid pt={[0]} gridTemplateColumns={["auto auto"]} gridColumnGap="40px">
          <Cell alignSelf="left" justifySelf="left">
            {title}
          </Cell>
          <Cell alignSelf="right" justifySelf="right">
            {numeral(total).format(Config.chain_core_asset_format)} {Config.chain_core_symbol_code}
          </Cell>
        </Grid>
      </Grid>
    )
  }

  renderRam(memoryContent: number[], memoryTotal: number) {
    let amount = memoryTotal - memoryContent[0]
    if (amount < 0) {
      amount = 0
    }
    return (
      <Cell
        pb={[3, 0, 0]}
        mb={[3, 0, 0]}
        borderBottom={[`1px solid ${theme.colors.grey4}`, "0px solid"]}
      >
        <StatusWidget
          title={t("account.status_bar.titles.memory")}
          description={t("account.status_bar.titles.available")}
          amount={
            <SearchShortcut
              position="left"
              query={`(ram.consumed:${this.props.account.account_name} OR ram.released:${this.props.account.account_name})`}
            >
              <Text fontSize={[4]}>{formatBytes(amount)}</Text>
            </SearchShortcut>
          }
        />
        <Grid gridTemplateColumns={["5fr 3fr", "2fr 1fr", "5fr 3fr"]}>
          {this.renderByteRatio(memoryContent[0], memoryTotal)}
          <Cell pl={[2]}>
            <StatusBar content={memoryContent} total={memoryTotal} />
          </Cell>
        </Grid>
      </Cell>
    )
  }

//ultra-andrey-bezrukov --- BLOCK-80 Integrate ultra power into dfuse and remove rex related tables
//  renderCPU(
//    cpuBandwidthContent: number[],
//    cpuBandwidthTotal: number,
//    totalCpu: string,
//    stakedCpu: string,
//    delegatedCpu: number,
//  ) {
//    const cpuBandwidthTitle = t("account.summary.tooltip.cpuTitle")
//    let amount = cpuBandwidthTotal - cpuBandwidthContent[0]
//    if (amount < 0) {
//      amount = 0
//    }
//    return (
//      <Cell
//        pb={[3, 0, 0]}
//        mb={[3, 0, 0]}
//        borderBottom={[`1px solid ${theme.colors.grey4}`, "0px solid"]}
//      >
//        <StatusWidget
//          title={t("account.status_bar.titles.cpu_bandwidth")}
//          description={t("account.status_bar.titles.available")}
//          amount={
//            <SearchShortcut
//              position="left"
//              query={`receiver:eosio (action:delegatebw OR action:undelegatebw) data.receiver:${this.props.account.account_name}`}
//            >
//              <Text fontSize={[4]}>{formatMicroseconds(amount)}</Text>
//            </SearchShortcut>
//          }
//        />
//        <Grid gridTemplateColumns={["5fr 3fr", "2fr 1fr", "5fr 3fr"]}>
//          {this.renderTimeRatio(cpuBandwidthContent[0], cpuBandwidthTotal)}
//          <Cell pl={[2]}>
//            <StatusBar content={cpuBandwidthContent} total={cpuBandwidthTotal} />
//          </Cell>
//        </Grid>
//        {this.renderStakeDetails(cpuBandwidthTitle, totalCpu, stakedCpu, delegatedCpu)}
//      </Cell>
//    )
//  }
//
//  renderNetwork(
//    networkBandwidthContent: number[],
//    networkBandwidthTotal: number,
//    totalNetwork: string,
//    stakedNetwork: string,
//    delegatedNetwork: number,
//  ) {
//    const networkBandwidthTitle = t("account.summary.tooltip.networkTitle")
//    let amount = networkBandwidthTotal - networkBandwidthContent[0]
//    if (amount < 0) {
//      amount = 0
//    }
//
//    return (
//      <Cell
//        pb={[3, 0, 0]}
//        mb={[3, 0, 0]}
//        borderBottom={[`1px solid ${theme.colors.grey4}`, "0px solid"]}
//      >
//        <StatusWidget
//          title={t("account.status_bar.titles.network_bandwidth")}
//          description={t("account.status_bar.titles.available")}
//          amount={
//            <SearchShortcut
//              position="left"
//              query={`receiver:eosio (action:delegatebw OR action:undelegatebw) data.receiver:${this.props.account.account_name}`}
//            >
//              <Text fontSize={[4]}>{formatBytes(amount)}</Text>
//            </SearchShortcut>
//          }
//        />
//        <Grid gridTemplateColumns={["5fr 3fr", "2fr 1fr", "5fr 3fr"]}>
//          {this.renderByteRatio(networkBandwidthContent[0], networkBandwidthTotal)}
//          <Cell pl={[2]}>
//            <StatusBar content={networkBandwidthContent} total={networkBandwidthTotal} />
//          </Cell>
//        </Grid>
//        {this.renderStakeDetails(
//          networkBandwidthTitle,
//          totalNetwork,
//          stakedNetwork,
//          delegatedNetwork,
//        )}
//      </Cell>
//    )
//  }

  renderPower(
    cpuBandwidthContent: number[],
    cpuBandwidthTotal: number,
    networkBandwidthContent: number[],
    networkBandwidthTotal: number,
    totalPower: string,
    stakedPower: string,
    delegatedPower: number,
  ) {
    const powerBandwidthTitle = t("account.summary.tooltip.powerTitle")
    let amountCpu = cpuBandwidthTotal - cpuBandwidthContent[0]
    if (amountCpu < 0) {
        amountCpu = 0
    }
    let amountNetwork = networkBandwidthTotal - networkBandwidthContent[0]
    if (amountNetwork < 0) {
        amountNetwork = 0
    }
    return (
      <Cell
        pb={[3, 0, 0]}
        mb={[3, 0, 0]}
      >
      <Grid gridTemplateColumns={"1fr 1fr"} gridColumnGap={[0, 8, 5]}>
        <Cell>
          <StatusWidget
            title={t("account.status_bar.titles.cpu_bandwidth")}
            description={t("account.status_bar.titles.available")}
            amount={
              <SearchShortcut
                position="left"
                query={`receiver:eosio (action:delegatebw OR action:undelegatebw) data.receiver:${this.props.account.account_name}`}
              >
                <Text fontSize={[4]}>{formatMicroseconds(amountCpu)}</Text>
              </SearchShortcut>
            }
          />
          <Grid gridTemplateColumns={["5fr 3fr", "2fr 1fr", "5fr 3fr"]}>
            {this.renderTimeRatio(cpuBandwidthContent[0], cpuBandwidthTotal)}
            <Cell pl={[2]}>
              <StatusBar content={cpuBandwidthContent} total={cpuBandwidthTotal} />
            </Cell>
          </Grid>
        </Cell>
        <Cell>
          <StatusWidget
            title={t("account.status_bar.titles.network_bandwidth")}
            description={t("account.status_bar.titles.available")}
            amount={
              <SearchShortcut
                position="left"
                query={`receiver:eosio (action:delegatebw OR action:undelegatebw) data.receiver:${this.props.account.account_name}`}
              >
                <Text fontSize={[4]}>{formatBytes(amountNetwork)}</Text>
              </SearchShortcut>
            }
          />
          <Grid gridTemplateColumns={["5fr 3fr", "2fr 1fr", "5fr 3fr"]}>
            {this.renderByteRatio(networkBandwidthContent[0], networkBandwidthTotal)}
            <Cell pl={[2]}>
              <StatusBar content={networkBandwidthContent} total={networkBandwidthTotal} />
            </Cell>
          </Grid>
        </Cell>
      </Grid>
        <Cell>
          {this.renderStakeDetails(powerBandwidthTitle, totalPower, stakedPower, delegatedPower)}
        </Cell>
      </Cell>
    )
  }

  render() {
    const { account } = this.props
    const memoryContent = [account.ram_usage]
    const selfDelegatedBandwidth = account.self_delegated_bandwidth

    const cpuBandwidthContent = [account.cpu_limit.used, account.cpu_limit.available]
    const networkBandwidthContent = [account.net_limit.used, account.net_limit.available]

//ultra-andrey-bezrukov --- BLOCK-80 Integrate ultra power into dfuse and remove rex related tables
//    const totalNetwork = extractValueWithUnits(account.total_resources.net_weight)[0]
//    const totalCpu = extractValueWithUnits(account.total_resources.cpu_weight)[0]
//    const stakedCpu = extractValueWithUnits(selfDelegatedBandwidth.cpu_weight)[0]
//    const stakedNetwork = extractValueWithUnits(selfDelegatedBandwidth.net_weight)[0]
//    const delegatedNetwork = parseFloat(totalNetwork) - parseFloat(stakedNetwork)
//    const delegatedCpu = parseFloat(totalCpu) - parseFloat(stakedCpu)

    const totalPower = extractValueWithUnits(account.total_resources.power_weight)[0]
    const stakedPower= extractValueWithUnits(selfDelegatedBandwidth.power_weight)[0]
    const delegatedPower = parseFloat(totalPower) - parseFloat(stakedPower)

//ultra-andrey-bezrukov --- BLOCK-80 Integrate ultra power into dfuse and remove rex related tables
//    return (
//      <AccountStatusBarsContainer>
//        <Grid gridTemplateColumns={["1fr", "1fr 1fr 1fr"]} gridColumnGap={[0, 4, 5]}>
//          {this.renderRam(memoryContent, account.ram_quota)}
//          {this.renderCPU(
//            cpuBandwidthContent,
//            account.cpu_limit.max,
//            totalCpu,
//            stakedCpu,
//            delegatedCpu,
//          )}
//          {this.renderNetwork(
//            networkBandwidthContent,
//            account.net_limit.max,
//            totalNetwork,
//            stakedNetwork,
//            delegatedNetwork,
//          )}
//        </Grid>
//      </AccountStatusBarsContainer>
//    )
    return (
      <AccountStatusBarsContainer>
        <Grid gridTemplateColumns={["1fr", "1fr 2fr"]} gridColumnGap={[0, 4, 5]}>
          {this.renderRam(memoryContent, account.ram_quota)}
          {this.renderPower(
            cpuBandwidthContent,
            account.cpu_limit.max,
            networkBandwidthContent,
            account.net_limit.max,
            totalPower,
            stakedPower,
            delegatedPower,
          )}
        </Grid>
      </AccountStatusBarsContainer>
    )
  }
}
