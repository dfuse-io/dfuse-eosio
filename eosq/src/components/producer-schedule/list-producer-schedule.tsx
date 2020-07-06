import { t } from "i18next"
import * as React from "react"
import { styled, theme } from "../../theme"
import {
  TableCaptionColor,
  TableCaptionItem,
  UiTable,
  UiTableBody,
  UiTableCell,
  UiTableHead,
  UiTableRow
} from "../../atoms/ui-table/ui-table.component"
import { Text } from "../../atoms/text/text.component"
import { Box } from "@dfuse/explorer"
import { Links } from "../../routes"
import { MonospaceTextLink } from "../../atoms/text-elements/misc"
import { Cell } from "../../atoms/ui-grid/ui-grid.component"

import { observer } from "mobx-react"
import { ReactNode } from "react"
import { ContentLoaderComponent } from "../content-loader/content-loader.component"
import { fetchProducerSchedule } from "../../services/producer-schedule"

import { Spinner } from "../../atoms/spinner/spinner"
import { ProducerScheduleItem } from "../../clients/websocket/eosws"

const UiTableCellRankHeader: React.ComponentType<any> = styled(UiTableCell)`
  text-align: center !important;
  width: 30px !important;
  padding: 16px 14px 4px 14px !important;
`

const UiTableCellRank: React.ComponentType<any> = styled(UiTableCell)`
  color: white !important;
  background-color: ${(props: any) => props.bg} !important;
  text-align: center !important;
  width: 40px !important;
  padding: 16px 14px 4px 14px !important;
`

const ProducerSpinner: React.ComponentType<any> = styled(Spinner)`
  transform: scale(0.3) translateY(14px);
  position: absolute;
  top: 5px;
  left: 100px;
`

const CaptionItem: React.ComponentType<any> = styled(TableCaptionItem)``

const CaptionColor: React.ComponentType<any> = styled(TableCaptionColor)``

const TableCaption: React.ComponentType<any> = styled(Box)`
  border-top: 1px solid ${(props) => props.theme.colors.border};
`

interface Props {
  headBlockProducer: string
}

@observer
export class ListProducerSchedule extends ContentLoaderComponent<Props, any> {
  constructor(props: Props) {
    super(props)
    fetchProducerSchedule()
  }

  renderProducerSpinner(producer: string): JSX.Element {
    if (this.props.headBlockProducer === producer) {
      return <ProducerSpinner fadeIn="none" name="three-bounce" color={theme.colors.ternary} />
    }
    return <span />
  }

  renderItem = (producer: ProducerScheduleItem, index: number) => {
    const bgColor = index % 2 ? "#00c8b1" : "#27cfb7"

    return (
      <UiTableRow key={producer.producer_name}>
        <UiTableCellRank fontSize={[2]} bg={bgColor}>
          {index}
        </UiTableCellRank>
        <UiTableCell fontSize={[2]}>
          <MonospaceTextLink to={Links.viewAccount({ id: producer.producer_name })}>
            {producer.producer_name}
          </MonospaceTextLink>
          {this.renderProducerSpinner(producer.producer_name)}
        </UiTableCell>

        <UiTableCell fontSize={[2]}>{producer.block_signing_key}</UiTableCell>
      </UiTableRow>
    )
  }

  renderItems = (producerSchedule: ProducerScheduleItem[]): JSX.Element => {
    // @ts-ignore Arguments to `map` are to hard to understand for TypeScript it seems

    return (
      <UiTableBody>
        {producerSchedule.map((producer: ProducerScheduleItem, index: number) =>
          this.renderItem(producer, index)
        )}
      </UiTableBody>
    )
  }

  renderHeader = (): JSX.Element => {
    return (
      <UiTableHead>
        <UiTableRow>
          <UiTableCellRankHeader fontSize={[2]}>Rank</UiTableCellRankHeader>
          <UiTableCell fontSize={[2]}>Name</UiTableCell>
          <UiTableCell fontSize={[2]}>Block Signing Key</UiTableCell>
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
        {/*
        // We disabled Stand-By coloring for now since the conditonal there is wrong, see `getRankBgColor`
        <CaptionItem>
          <CaptionColor bg={["#ffb866"]} />
          <Text fontWeight="300" fontSize={[1]}>
            {t("vote.list.legend.standBy")}
          </Text>
        </CaptionItem>
        */}
        <CaptionItem>
          <CaptionColor bg={["#d0d0d0"]} />
          <Text fontWeight="300" fontSize={[1]}>
            {t("vote.list.legend.runnerUps")}
          </Text>
        </CaptionItem>
      </TableCaption>
    )
  }

  renderContent = (producerSchedule: ProducerScheduleItem[]): ReactNode => {
    return (
      <Cell>
        <UiTable>
          {this.renderHeader()}
          {this.renderItems(producerSchedule)}
        </UiTable>
      </Cell>
    )
  }

  render() {
    return <Cell>{this.handleRender(fetchProducerSchedule, "loading...")}</Cell>
  }
}
