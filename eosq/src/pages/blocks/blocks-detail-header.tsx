import { t } from "i18next"
import { observer } from "mobx-react"
import * as React from "react"
import { MonospaceText, MonospaceTextLink } from "../../atoms/text-elements/misc"
import { DetailLine, BULLET, formatNumber, formatPercentage, truncateString } from "@dfuse/explorer"

import { Links } from "../../routes"
import { Text } from "../../atoms/text/text.component"
import { Cell, Grid } from "../../atoms/ui-grid/ui-grid.component"
import { Age } from "../../atoms/age/age.component"
import { fetchBlock } from "../../services/block"
import { metricsStore } from "../../stores"
import { RouteComponentProps } from "react-router-dom"
import { BlockProgressPie } from "./block-progress-pie"
import { computeTransactionTrustPercentage } from "../../models/transaction"
import { ContentLoaderComponent } from "../../components/content-loader/content-loader.component"
import { Panel } from "../../atoms/panel/panel.component"
import { PanelTitleBanner } from "../../atoms/panel/panel-title-banner"
import { UiToolTip } from "../../atoms/ui-tooltip/ui-tooltip"
import { NavigationButtons } from "../../atoms/navigation-buttons/navigation-buttons"
import { PageContainer } from "../../components/page-container/page-container"
import { theme, styled } from "../../theme"
import { BlockSummary, ProducerKey } from "../../models/block"

interface Props extends RouteComponentProps<any> {}

const ScheduleUl: React.ComponentType<any> = styled.ul`
  list-style: none;
  text-align: left;
  -webkit-columns: 3;
  -moz-columns: 3;
  columns: 3;
  list-style-position: inside;
  padding-left: 0px;

  @media (max-width: 450px) {
    columns: 2;
    -webkit-columns: 2;
    -moz-columns: 2;
  }
`
const PanelContentWrapper: React.ComponentType<any> = styled(Cell)`
  width: 100%;
  // min-width: 1000px;
`

@observer
export class BlockHeader extends ContentLoaderComponent<Props, any> {
  componentDidMount() {
    this.changeDocumentTitle()
    fetchBlock(this.props.match.params.id)
  }

  componentDidUpdate(prevProps: Props) {
    this.changeDocumentTitle()
    if (prevProps.match.params.id !== this.props.match.params.id) {
      fetchBlock(this.props.match.params.id)
    }
  }

  changeDocumentTitle() {
    document.title = `${truncateString(this.props.match.params.id, 8).join(
      ""
    )} ${BULLET} block ${BULLET} eosq`
  }

  hasRecentMetrics(block: BlockSummary) {
    return metricsStore.lastIrreversibleBlockNum > 0 && metricsStore.headBlockNum > block.block_num
  }

  renderProducerValue = (producer: string): React.ReactChild => {
    if (!producer || producer === "") {
      return <MonospaceText>{t("transaction.detailPanel.producer.unknown")}</MonospaceText>
    }

    return (
      <MonospaceTextLink to={Links.viewAccount({ id: producer })}>{producer}</MonospaceTextLink>
    )
  }

  renderAge = (timestamp: string): React.ReactChild => {
    return <Age date={timestamp} />
  }

  renderText = (text: string) => {
    return <Text>{text}</Text>
  }

  renderMonospaceText = (text: string) => {
    return <Text>{text}</Text>
  }

  renderProducerSchedule = (block: BlockSummary): JSX.Element[] => {
    return (block.active_schedule.producers || []).map(
      (scheduleItem: ProducerKey, index: number) => {
        return (
          <li key={index}>
            {index + 1}: {scheduleItem.producer_name}
          </li>
        )
      }
    )
  }

  blockInactive = (block: BlockSummary): boolean => {
    return metricsStore.lastIrreversibleBlockNum >= block.block_num && !block.irreversible
  }

  renderStatus = (block: BlockSummary): JSX.Element | string => {
    const percentage = computeTransactionTrustPercentage(
      block.block_num,
      metricsStore.headBlockNum,
      metricsStore.lastIrreversibleBlockNum
    )

    if (!this.hasRecentMetrics(block)) {
      return "-"
    }

    if (this.blockInactive(block)) {
      return (
        <Text fontWeight="bold" color="secondHighlight">
          {t("block.stale")}
        </Text>
      )
    }

    if (percentage >= 1) {
      return (
        <Text fontWeight="bold" color="ternary">
          {t("block.irreversible")}
        </Text>
      )
    }

    return <Text>{formatPercentage(percentage)}</Text>
  }

  getNextBlock(refBlock: BlockSummary) {
    const candidates = (refBlock.sibling_blocks || []).filter((block: BlockSummary) => {
      return block.header.previous === refBlock.id
    })

    let winner: any
    if (candidates.length > 1) {
      winner = candidates.find((candidate: BlockSummary) => {
        return candidate.irreversible
      })
    }

    return winner || candidates[0]
  }

  getPreviousBlock(refBlock: BlockSummary) {
    return (refBlock.sibling_blocks || []).find((block: BlockSummary) => {
      return block.id === refBlock.header.previous
    })
  }

  goToBlock = (block: BlockSummary) => {
    this.props.history.push(Links.viewBlock({ id: block.id }))
  }

  renderDetail = (block: BlockSummary): JSX.Element => {
    return (
      <Cell wordBreak="break-all" pt={[2]}>
        <DetailLine color="text" variant="compact" label={t("transaction.blockPanel.block")}>
          {this.renderMonospaceText(formatNumber(block.block_num))}
        </DetailLine>
        <DetailLine variant="compact" label={t("transaction.blockPanel.age")}>
          {this.renderAge(block.header.timestamp)}
        </DetailLine>
        <DetailLine variant="compact" label={t("transaction.blockPanel.blockId")}>
          {this.renderMonospaceText(block.id)}
        </DetailLine>
        <DetailLine variant="compact" label={t("transaction.blockPanel.status")}>
          {this.renderStatus(block)}
        </DetailLine>
        <DetailLine variant="compact" label={t("transaction.blockPanel.producer")}>
          {this.renderProducerValue(block.header.producer)}
        </DetailLine>
        <DetailLine variant="compact" label={t("block.transactionCount")}>
          {this.renderText(formatNumber(block.transaction_count))}
        </DetailLine>
        <DetailLine variant="compact" label={t("block.scheduleVersion")}>
          <UiToolTip>
            <Text color="text" borderBottom={`2px dotted ${theme.colors.grey5}`}>
              {block.header.schedule_version}
            </Text>
            <Cell p={[3]}>
              <Cell>
                <Text color="primary">
                  {t("block.producerSchedule.title")} #{block.header.schedule_version}
                </Text>
                <ScheduleUl>{this.renderProducerSchedule(block)}</ScheduleUl>
              </Cell>
            </Cell>
          </UiToolTip>
        </DetailLine>
        {block.dpos_lib_num ? (
          <DetailLine variant="compact" label={t("block.dpos_lib_num")}>
            {" "}
            {this.renderMonospaceText(formatNumber(block.dpos_lib_num))}{" "}
          </DetailLine>
        ) : null}
      </Cell>
    )
  }

  renderContent = (block: BlockSummary) => {
    if (!block) {
      return <div />
    }

    const next = this.getNextBlock(block)

    const prev = this.getPreviousBlock(block)

    return (
      <PageContainer>
        <PanelTitleBanner title="Block #" content={formatNumber(block.block_num)}>
          <NavigationButtons
            onNext={() => this.goToBlock(next as BlockSummary)}
            onPrev={() => this.goToBlock(prev as BlockSummary)}
            showNext={!!next}
            showPrev={!!prev}
            showFirst={false}
          />
        </PanelTitleBanner>
        <Panel>
          <PanelContentWrapper pt={[0]}>
            <Grid gridTemplateColumns={["1fr", "4fr 200px"]}>
              <Grid px={[3, 4]} pt={[3, 2]} pb={[1, 2]} gridRow={[2, 1]}>
                <Cell>{this.renderDetail(block)}</Cell>
              </Grid>
              <Grid
                maxWidth={["200px", "none"]}
                mx={["auto", 0]}
                px={[1, 4]}
                pb={[1, 2]}
                pt={[4, 0]}
                gridRow={[1, 1]}
              >
                {this.blockInactive(block) || !this.hasRecentMetrics(block) ? null : (
                  <BlockProgressPie
                    headBlockNum={metricsStore.headBlockNum}
                    blockNum={block.block_num}
                    lastIrreversibleBlockNum={metricsStore.lastIrreversibleBlockNum}
                  />
                )}
              </Grid>
            </Grid>
          </PanelContentWrapper>
        </Panel>
      </PageContainer>
    )
  }

  render() {
    return this.handleRender(fetchBlock, "Loading block")
  }
}
