import { t } from "i18next"
import { max } from "ramda"
import * as React from "react"
import { compactCount, compactString, formatNumber } from "../../helpers/formatters"
import { styled } from "../../theme"
import { BlockSummary } from "../../models/block"
import { Links } from "../../routes"
import { Cell } from "../../atoms/ui-grid/ui-grid.component"
import { MonospaceTextLink } from "../../atoms/text-elements/misc"
import { faLock, faLockOpen } from "@fortawesome/free-solid-svg-icons"

import {
  TableIcon,
  TableIconLight,
  UiTable,
  UiTableBody,
  UiTableCell,
  UiTableHead,
  UiTableRow,
  UiTableRowAlternated
} from "../../atoms/ui-table/ui-table.component"
import { TextLink } from "../../atoms/text/text.component"
import { formatDateFromString } from "../../helpers/moment.helpers"

const UiTableCellTransactionRatio: React.ComponentType<any> = styled(UiTableCell)`
  border-left: 1px dashed #ddd;
  padding-left: 0px !important;
`
const TransactionRatioCell: React.ComponentType<any> = styled(Cell)`
  height: 30px;
`

const Ratio: React.ComponentType<any> = styled(Cell)`
  background-color: ${(props) => props.theme.colors.banner};
  height: 100%;
  float: right;
`

interface Props {
  blocks: BlockSummary[]
}

interface State {
  blockScheduleSelected: number
}

export class ListBlocks extends React.Component<Props, State> {
  renderTimeStamp(timestamp: string) {
    if (!timestamp || timestamp === "") {
      return null
    }
    return (
      <Cell color="text" title={formatDateFromString(timestamp, true)}>
        {formatDateFromString(timestamp, false)}
      </Cell>
    )
  }

  renderBlockIrreversible(block: BlockSummary): JSX.Element | null {
    return block.irreversible ? <TableIcon icon={faLock} /> : <TableIconLight icon={faLockOpen} />
  }

  renderItem = (block: BlockSummary, maxTransactionCount: number) => {
    const ratioWidth = Math.ceil(
      maxTransactionCount <= 0 ? 0 : (block.transaction_count / maxTransactionCount) * 100
    )

    return (
      <UiTableRowAlternated key={block.id}>
        <UiTableCell fontSize={[2]}>
          <TextLink to={Links.viewBlock({ id: block.id })}>
            {formatNumber(block.block_num)}
          </TextLink>
          {this.renderBlockIrreversible(block)}
        </UiTableCell>
        <UiTableCell fontSize={[2]}>
          <TextLink fontSize={[2]} to={Links.viewBlock({ id: block.id })}>
            {compactString(block.id, 12, 0)}
          </TextLink>
        </UiTableCell>
        <UiTableCell fontSize={[2]}>{this.renderTimeStamp(block.header.timestamp)}</UiTableCell>

        <UiTableCell fontSize={[2]}>
          <MonospaceTextLink fontSize={[2]} to={Links.viewAccount({ id: block.header.producer })}>
            {block.header.producer}
          </MonospaceTextLink>
        </UiTableCell>
        <UiTableCellTransactionRatio fontSize={[2]}>
          <TransactionRatioCell py="3px" pr={[0]} alignSelf="center" justifySelf="right" w="100%">
            <Ratio width={`${ratioWidth}%`}>&nbsp;</Ratio>
          </TransactionRatioCell>
        </UiTableCellTransactionRatio>
        <UiTableCell color="text" fontSize={[2]}>
          {compactCount(block.transaction_count)}
        </UiTableCell>
      </UiTableRowAlternated>
    )
  }

  findMaxTransactionCount(blocks: BlockSummary[]) {
    return blocks.reduce(
      (maxTransactionCount, block) => max(block.transaction_count, maxTransactionCount),
      0
    )
  }

  renderItems = (): JSX.Element => {
    if (this.props.blocks) {
      const maxTransactionCount = this.findMaxTransactionCount(this.props.blocks)

      return (
        <UiTableBody>
          {this.props.blocks.map((block: BlockSummary) =>
            this.renderItem(block, maxTransactionCount)
          )}
        </UiTableBody>
      )
    }
    return <span />
  }

  renderHeader = () => {
    return (
      <UiTableHead>
        <UiTableRow>
          <UiTableCell fontSize={[2]}>{t("block.list.header.block_num")}</UiTableCell>
          <UiTableCell fontSize={[2]}>{t("block.list.header.id")}</UiTableCell>
          <UiTableCell fontSize={[2]}>{t("block.list.header.timestamp")}</UiTableCell>
          <UiTableCell fontSize={[2]}>{t("block.list.header.producer")}</UiTableCell>
          <UiTableCell fontSize={[2]}>{t("block.list.header.transactionCount")}</UiTableCell>
          <UiTableCell fontSize={[2]}>&nbsp;</UiTableCell>
        </UiTableRow>
      </UiTableHead>
    )
  }

  render() {
    return (
      <UiTable>
        {this.renderHeader()}
        {this.renderItems()}
      </UiTable>
    )
  }
}
