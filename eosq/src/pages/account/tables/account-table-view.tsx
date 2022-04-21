import { t } from "i18next"
import H from "history"
import { observer } from "mobx-react"
import * as React from "react"
import queryString from "query-string"
import { JsonWrapper, humanizeSnakeCase, NBSP, Box } from "@dfuse/explorer"
import { BorderLessPanel } from "../../../atoms/panel/panel.component"
import { Text } from "../../../atoms/text/text.component"
import { Cell, Grid } from "../../../atoms/ui-grid/ui-grid.component"
import { UiSwitch } from "../../../atoms/ui-switch/switch"
import {
  UiTable,
  UiTableBody,
  UiTableCell,
  UiTableHead,
  UiTableRow,
  UiTableRowAlternated
} from "../../../atoms/ui-table/ui-table.component"
import { ContentLoaderComponent } from "../../../components/content-loader/content-loader.component"
import { FormattedContractElement } from "../../../components/formatted-contract-element/formatted-contract-element"

import { fetchContractTableRowsOnContractPage } from "../../../services/contract-table"
import { Links } from "../../../routes"
import { NavigationButtons } from "../../../atoms/navigation-buttons/navigation-buttons"
import { contractTableStore } from "../../../stores"
import { AbiStructField } from "@dfuse/client"
import { GetTableRowParams } from "../../../clients/websocket/eosws"

interface Props {
  location: H.Location
  history: H.History
}

interface State {
  formatted: boolean
}

@observer
export class AccountTableView extends ContentLoaderComponent<Props, State> {
  gridTemplateColumns = ""
  tableFields: AbiStructField[] = []

  constructor(props: Props) {
    super(props)

    const parsed = queryString.parse(this.props.location.search)
    if (typeof parsed.offset === "string" && parsed.offset) {
      contractTableStore.offset = this.extractOffset(parsed.offset)
    }

    this.state = {
      formatted: true
    }
  }

  extractOffset(value: string): number {
    if (!/[0-9]+/.test(value)) return 0

    return parseInt(value, 10)
  }

  loadMore = (lastRowKey: string) => {
    contractTableStore.lowerBound = lastRowKey
    contractTableStore.upperBound = undefined

    this.fetchContent(1, contractTableStore.params)
  }

  fetchContent(direction: number, params: GetTableRowParams) {
    this.setState(
      (prevState: State) => {
        contractTableStore.offset += contractTableStore.limit * direction

        return {
          formatted: prevState.formatted
        }
      },
      () => {
        this.props.history.push(
          `${Links.viewAccountTabs({
            id: contractTableStore.accountName,
            currentTab: "tables"
          })}?${queryString.stringify(contractTableStore.urlParams)}`
        )

        fetchContractTableRowsOnContractPage(params)
      }
    )
  }

  loadLess = (lastRowKey: string) => {
    contractTableStore.upperBound = lastRowKey
    contractTableStore.lowerBound = undefined

    this.fetchContent(-1, contractTableStore.params)
  }

  renderItem = (row: any, index: number) => {
    return <UiTableRowAlternated key={index}>{this.renderRowCells(row)}</UiTableRowAlternated>
  }

  renderCellContent(value: any, field: AbiStructField) {
    if (value === null) {
      return <Text>null</Text>
    }

    // This condition will catch both objects and arrays, which is ok in this case
    if (typeof value === "object") {
      return <JsonWrapper>{JSON.stringify(value, null, "   ")}</JsonWrapper>
    }

    if (this.state.formatted) {
      return <FormattedContractElement label={field.name} value={value} type={field.type} />
    }

    return <Text>{value}</Text>
  }

  renderRowCells(row: any) {
    return this.tableFields.map((field: AbiStructField, index: number) => {
      /*ultra-duncan---BLOCK-154 Fix display for 0 value field --- */
      let value = row[field.name]
      if (value === 0 || value === null)
        value = `${value}`
      if (value) {
        return (
          <UiTableCell fontSize={[2]} key={index}>
            {this.renderCellContent(value, field)}
            {NBSP}
          </UiTableCell>
        )
      }

      return <UiTableCell fontSize={[2]} key={index} />
    })
  }

  renderItems = (tableRows: any[]): React.ReactChild[] => {
    return tableRows.map((row: any, index: number) => {
      return this.renderItem(row, index)
    })
  }

  renderHeaderCells = () => {
    return this.tableFields.map((field: AbiStructField, index: number) => {
      return (
        <UiTableCell fontSize={[3]} key={index}>
          {this.state.formatted ? humanizeSnakeCase(field.name) : field.name}
        </UiTableCell>
      )
    })
  }

  renderHeader = (): React.ReactChild => {
    return <UiTableRow>{this.renderHeaderCells()}</UiTableRow>
  }

  onSwitchFormat = (formatted: boolean) => {
    this.setState({ formatted })
  }

  renderSwitchGrid = (): JSX.Element => {
    return (
      <Grid gridTemplateColumns={["1fr"]}>
        <Grid
          mr={[4]}
          justifyItems="center"
          alignItems="center"
          justifySelf="end"
          gridTemplateColumns={["auto auto"]}
          grid-template-rows={["35px"]}
        >
          <Text pr={[1]} display="inline-block">
            {t("account.tables.formatted")}
          </Text>
          <UiSwitch
            checked={this.state.formatted}
            onChange={(checked: boolean) => this.onSwitchFormat(checked)}
          />
        </Grid>
      </Grid>
    )
  }

  renderNavigation = (tableRows: any) => {
    if (!tableRows.rows) {
      return <span />
    }
    const lastRow = tableRows.rows[tableRows.rows.length - 1]
    const { firstTableKey } = contractTableStore
    const lastRowKey = lastRow && firstTableKey ? lastRow[firstTableKey] : 0

    return (
      <NavigationButtons
        onNext={() => this.loadMore(lastRowKey)}
        onPrev={() => this.loadLess(lastRowKey)}
        showNext={tableRows.more}
        showPrev={contractTableStore.offset !== 0}
        showFirst={false}
        variant="light"
      />
    )
  }

  renderEmptyTable = (): JSX.Element => {
    return (
      <Cell>
        <BorderLessPanel title={contractTableStore.tableName}>
          <Grid>
            <Cell overflowX="auto">
              <UiTable>
                <UiTableHead>{this.renderHeader()}</UiTableHead>
                <UiTableBody>
                  <UiTableRowAlternated>
                    <tr>
                      <Box px="20px" py={[4]}>
                        Empty Table
                      </Box>
                    </tr>
                  </UiTableRowAlternated>
                </UiTableBody>
              </UiTable>
            </Cell>
          </Grid>
        </BorderLessPanel>
      </Cell>
    )
  }

  renderContent = (tableRows: any): JSX.Element => {
    return (
      <Cell>
        <BorderLessPanel
          title={contractTableStore.tableName}
          renderSideTitle={() => {
            return (
              <Grid gridTemplateColumns={["2fr 1fr"]} alignItems="center">
                {this.renderSwitchGrid()}
                {this.renderNavigation(tableRows)}
              </Grid>
            )
          }}
        >
          <Grid>
            <Cell overflowX="auto">
              <UiTable>
                <UiTableHead>{this.renderHeader()}</UiTableHead>
                <UiTableBody>{this.renderItems(tableRows.rows || [])}</UiTableBody>
              </UiTable>
            </Cell>
          </Grid>
          <Grid gridTemplateColumns={["1fr"]}>
            <Cell justifySelf="right" alignSelf="right" p={[4]}>
              {this.renderNavigation(tableRows)}
            </Cell>
          </Grid>
        </BorderLessPanel>
      </Cell>
    )
  }

  render() {
    this.tableFields = contractTableStore.abiLoader!.getTableFields(contractTableStore.tableName)
    this.gridTemplateColumns = "1fr ".repeat(this.tableFields.length)

    if (this.tableFields && contractTableStore.tableName !== "") {
      if (contractTableStore.loading) {
        return this.renderLoading("loading table")
      }

      if (contractTableStore.error) {
        return this.renderError()
      }

      if (contractTableStore.nRows > 0) {
        return this.renderContent(contractTableStore.tableRows)
      }

      if (contractTableStore.nRows === 0) {
        return this.renderEmptyTable()
      }

      return this.renderError()
    }

    return <div />
  }
}
