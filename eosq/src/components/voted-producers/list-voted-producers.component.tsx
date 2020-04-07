import { t } from "i18next"
import * as React from "react"
import { formatNumber } from "../../helpers/formatters"
import { Vote } from "../../models/vote"
import { Text } from "../../atoms/text/text.component"
import Box from "../../atoms/ui-box"
import { Links } from "../../routes"
import { MonospaceTextLink } from "../../atoms/text-elements/misc"
import { Cell, Grid } from "../../atoms/ui-grid/ui-grid.component"
import { VotedProducerPagination } from "./voted-producers-pagination.component"
import { theme, styled } from "../../theme"
import { observer } from "mobx-react"
import {
  TableCaptionColor,
  TableCaptionItem,
  UiTable,
  UiTableBody,
  UiTableCell,
  UiTableHead,
  UiTableRow,
  UiTableRowAlternated
} from "../../atoms/ui-table/ui-table.component"
import { Panel } from "../../atoms/panel/panel.component"
import { getRankBgColor } from "../../helpers/account.helpers"
import { Spinner } from "../../atoms/spinner/spinner"
import { Config } from "../../models/config"

const UiTableCellRank: React.ComponentType<any> = styled(UiTableCell)`
  color: white !important;
  background-color: ${(props: any) => props.bg} !important;
  text-align: center !important;
  width: 40px !important;
  padding: 4px 14px 4px 14px !important;
`

const UiTableCellAccount: React.ComponentType<any> = styled(UiTableCell)`
  min-width: 180px !important;
`

const UiTableCellRankHeader: React.ComponentType<any> = styled(UiTableCell)`
  text-align: center !important;
  width: 30px !important;
  padding: 4px 14px 4px 14px !important;
`

const ProducerSpinner: React.ComponentType<any> = styled(Spinner)`
  transform: scale(0.3) translateY(14px);
  position: absolute;
  top: 5px;
  left: 120px;
`

const CaptionItem: React.ComponentType<any> = styled(TableCaptionItem)``

const CaptionColor: React.ComponentType<any> = styled(TableCaptionColor)``

const TableCaption: React.ComponentType<any> = styled(Box)`
  border-top: 1px solid ${(props) => props.theme.colors.border};
`

interface Props {
  votes: Vote[]
  headBlockProducer: string
}

type State = {
  offset: number
}

@observer
export class ListVotedProducers extends React.Component<Props, State> {
  private perPage = 100

  constructor(props: Props, state: any) {
    super(props)
    this.state = { offset: 0 }
  }

  renderProducerSpinner(producer: string): JSX.Element {
    if (this.props.headBlockProducer === producer) {
      return <ProducerSpinner fadeIn="none" name="three-bounce" color={theme.colors.ternary} />
    }
    return <span />
  }

  renderItem = (vote: Vote, rank: number) => {
    const bgColor = getRankBgColor({ rank, votePercent: vote.votePercent })

    return (
      <UiTableRowAlternated key={vote.producer}>
        <UiTableCellRank bg={bgColor} fontSize={[2]}>
          {rank}
        </UiTableCellRank>
        <UiTableCellAccount fontSize={[2]}>
          <MonospaceTextLink to={Links.viewAccount({ id: vote.producer })}>
            {vote.producer}
          </MonospaceTextLink>
          {this.renderProducerSpinner(vote.producer)}
        </UiTableCellAccount>
        <UiTableCell fontSize={[2]}>{vote.votePercent.toFixed(3)} %</UiTableCell>
        <UiTableCell fontSize={[2]}>
          {formatNumber(vote.decayedVote)} {Config.price_ticker_name}
        </UiTableCell>
      </UiTableRowAlternated>
    )
  }

  onNext = () => {
    this.setState((prevState) => ({ offset: prevState.offset + this.perPage }))
  }

  onPrev = () => {
    this.setState((prevState) => ({ offset: prevState.offset - this.perPage }))
  }

  renderItems = (): JSX.Element => {
    // @ts-ignore Arguments to `map` are to hard to understand for TypeScript it seems

    return (
      <UiTableBody>
        {this.props.votes
          .slice(this.state.offset, this.state.offset + this.perPage)
          .map((vote: Vote, index: number) => this.renderItem(vote, this.state.offset + index + 1))}
      </UiTableBody>
    )
  }

  renderHeader = (): JSX.Element => {
    return (
      <UiTableHead>
        <UiTableRow>
          <UiTableCellRankHeader fontSize={[2]}>{t("vote.list.header.rank")}</UiTableCellRankHeader>
          <UiTableCellAccount fontSize={[2]}>{t("vote.list.header.account")}</UiTableCellAccount>
          <UiTableCell color="text" fontSize={[2]}>
            {t("vote.list.header.votePercent")}
          </UiTableCell>
          <UiTableCell color="text" fontSize={[2]}>
            {t("vote.list.header.decayedVote")}
          </UiTableCell>
        </UiTableRow>
      </UiTableHead>
    )
  }

  renderCaption = (): JSX.Element => {
    return (
      <TableCaption pt={[4]} pb={[4]} pl={[4]} pr={[3]} width={["100%"]}>
        <CaptionItem>
          <CaptionColor bg={["#27cfb7"]} />
          <Text fontWeight="300" fontSize={[1]}>
            {t("vote.list.legend.active")}
          </Text>
        </CaptionItem>
        <CaptionItem>
          <CaptionColor bg={["#ffb866"]} />
          <Text fontWeight="300" fontSize={[1]}>
            {t("vote.list.legend.standBy")}
          </Text>
        </CaptionItem>
        <CaptionItem>
          <CaptionColor bg={["#d0d0d0"]} />
          <Text fontWeight="300" fontSize={[1]}>
            {t("vote.list.legend.runnerUps")}
          </Text>
        </CaptionItem>
      </TableCaption>
    )
  }

  onClickPage = (offset: number) => {
    this.setState((prevState) => ({ offset: prevState.offset * this.perPage }))
  }

  render() {
    const numberOfPages = this.props.votes.length / this.perPage
    return (
      <Panel
        title={t("producer.title")}
        renderSideTitle={() => (
          <VotedProducerPagination
            currentPage={this.state.offset / this.perPage}
            onClickPage={this.onClickPage}
            numberOfPages={numberOfPages}
            showPrev={this.state.offset !== 0}
            showNext={this.state.offset + this.perPage < this.props.votes.length}
            onNextClick={this.onNext}
            onPrevClick={this.onPrev}
          />
        )}
      >
        <Cell overflowX="auto">
          <UiTable>
            {this.renderHeader()}
            {this.renderItems()}
          </UiTable>
          <Cell>{this.renderCaption()}</Cell>
          <Grid>
            <Cell justifySelf={["right"]} alignSelf={["right"]} px={[3, 4, 4]} pb={[3, 4, 4]}>
              <VotedProducerPagination
                currentPage={this.state.offset / this.perPage}
                onClickPage={this.onClickPage}
                numberOfPages={numberOfPages}
                showPrev={this.state.offset !== 0}
                showNext={this.state.offset + this.perPage < this.props.votes.length}
                onNextClick={this.onNext}
                onPrevClick={this.onPrev}
              />
            </Cell>
          </Grid>
        </Cell>
      </Panel>
    )
  }
}
