import { t } from "i18next"
import { observer } from "mobx-react"
import * as React from "react"

// temp ignore for dev

import { DataEmpty } from "@dfuse/explorer"
import { Panel } from "../../atoms/panel/panel.component"
import { fetchBlockList } from "../../services/block"
import { ListBlocks } from "../../components/list-blocks/list-blocks.component"
import { ContentLoaderComponent } from "../../components/content-loader/content-loader.component"
import { Cell, Grid } from "../../atoms/ui-grid/ui-grid.component"
import { NavigationButtons } from "../../atoms/navigation-buttons/navigation-buttons"
import { metricsStore } from "../../stores"
import { RouteComponentProps } from "react-router"
import queryString from "query-string"
import { Links } from "../../routes"
import { BlockSummary, BLOCK_NUM_100B } from "../../models/block"

type Props = RouteComponentProps<{}>

@observer
export class PagedBlocks extends ContentLoaderComponent<Props> {
  firstBlockNum = BLOCK_NUM_100B
  lastBlockNum = BLOCK_NUM_100B

  PER_PAGE = 100

  get parsed() {
    return queryString.parse(this.props.location.search)
  }

  componentDidMount() {
    if (this.parsed.lastBlockNum && this.parsed.lastBlockNum.length > 0) {
      this.lastBlockNum = this.parsed.lastBlockNum
    }

    fetchBlockList(this.lastBlockNum)
  }

  renderEmpty() {
    return <DataEmpty text={t("block.list.empty")} />
  }

  renderContent = (blocks: BlockSummary[]) => {
    if (!blocks) {
      return this.renderEmpty()
    }

    this.lastBlockNum = blocks[0].block_num
    this.firstBlockNum = blocks[blocks.length - 1].block_num
    return (
      <Cell>
        <Cell overflowX="auto">
          <ListBlocks blocks={blocks} />
        </Cell>
        <Grid gridTemplateColumns={["1fr"]}>
          <Cell justifySelf="right" alignSelf="right" p={[4]}>
            {this.renderNavigation()}
          </Cell>
        </Grid>
      </Cell>
    )
  }

  onFirst = () => {
    this.props.history.replace(`${Links.blocks()}?lastBlockNum=${metricsStore.headBlockNum - 1}`)

    fetchBlockList(metricsStore.headBlockNum - 1)
  }

  onNext = () => {
    this.props.history.replace(`${Links.blocks()}?lastBlockNum=${this.firstBlockNum - 1}`)

    fetchBlockList(this.firstBlockNum - 1)
  }

  onPrev = () => {
    this.props.history.replace(
      `${Links.blocks()}?lastBlockNum=${this.lastBlockNum + this.PER_PAGE}`
    )

    fetchBlockList(this.lastBlockNum + this.PER_PAGE)
  }

  renderNavigation = () => {
    return (
      <NavigationButtons
        onFirst={this.onFirst}
        onNext={this.onNext}
        onPrev={this.onPrev}
        showNext={true}
        showFirst={this.lastBlockNum < metricsStore.headBlockNum}
        showPrev={this.lastBlockNum < metricsStore.headBlockNum}
        variant="light"
      />
    )
  }

  render() {
    return (
      <Panel title={t("block.list.title")} renderSideTitle={() => this.renderNavigation()}>
        {this.handleRender(fetchBlockList, t("block.list.loading"))}
      </Panel>
    )
  }
}
