import Button from "@material-ui/core/Button"
import InputLabel from "@material-ui/core/InputLabel"
import H from "history"
import { t } from "i18next"
import { observer } from "mobx-react"
import queryString from "query-string"
import * as React from "react"
import { Text } from "../../../atoms/text/text.component"
import { DropDownOption, UiDropDown } from "../../../atoms/ui-dropdown/ui-dropdown.component"
import { Cell, Grid } from "../../../atoms/ui-grid/ui-grid.component"
import { UiInput } from "../../../atoms/ui-text-field/ui-text-field"
import { styled } from "../../../theme"
import { Links } from "../../../routes"
import { AbiLoader } from "../../../services/abi-loader"
import { contractTableStore } from "../../../stores"
import { fetchContractTableRowsOnContractPage } from "../../../services/contract-table"

const StyledButton: React.ComponentType<any> = styled(Button)`
  padding-left: 20px !important;
  padding-right: 20px !important;
  background-color: ${(props) => props.theme.colors.ternary} !important;
  border: none !important;
  font-weight: bold !important;
  border-radius: 0px !important;
  height: 35px !important;
  min-height: 35px !important;
  color: ${(props) => props.theme.colors.primary} !important;
`

interface Props {
  abiLoader: AbiLoader
  accountName: string
  location: H.Location
  history: H.History
}

@observer
export class AccountTableSearch extends React.Component<Props> {
  constructor(props: Props) {
    super(props)

    this.initContractTableStore()
  }

  onClick = () => {
    fetchContractTableRowsOnContractPage(contractTableStore.params)

    this.props.history.push(
      `${Links.viewAccountTabs({
        id: this.props.accountName,
        currentTab: "tables"
      })}?${queryString.stringify(contractTableStore.urlParams)}`
    )
  }

  initContractTableStore() {
    const parsed = queryString.parse(this.props.location.search)
    contractTableStore.initFromUrlParams(this.props.abiLoader, this.props.accountName, parsed)
  }

  componentDidMount() {
    fetchContractTableRowsOnContractPage(contractTableStore.params)
    this.initContractTableStore()
  }

  selectTableWithHistory = (tableName: string) => {
    contractTableStore.tableName = tableName
  }

  renderDropDown(): JSX.Element {
    const dropDownOptions: DropDownOption[] = this.props.abiLoader.tableNames.map((tableName) => {
      return { label: tableName, value: tableName }
    })

    const parsed = queryString.parse(this.props.location.search)
    let selectedTableName = parsed.tableName
    if (!selectedTableName && this.props.abiLoader.tableNames.length > 0) {
      // eslint-disable-next-line prefer-destructuring
      selectedTableName = this.props.abiLoader.tableNames[0]
    }

    if (this.props.abiLoader) {
      return (
        <UiDropDown
          label={t("accountTables.tables.dropdown.placeholder")}
          placeholder={t("accountTables.tables.dropdown.placeholder")}
          options={dropDownOptions}
          defaultValue={selectedTableName}
          onSelect={this.selectTableWithHistory}
          id="table-selector"
        />
      )
    }

    return <span />
  }

  renderScopeInput() {
    return [
      <InputLabel key="scope-input-label" htmlFor="scope-input">
        <Text pl={[2]} fontSize={[1]}>
          {t("accountTables.search.scope")}
        </Text>
      </InputLabel>,
      <UiInput
        key="scope-input"
        id="scope-input"
        disableUnderline={true}
        value={contractTableStore.scope}
        placeholder={t("accountTables.search.scope")}
        onChange={(e: any) => {
          contractTableStore.scope = e.target.value
        }}
      />
    ]
  }

  renderLowerBoundInput() {
    return [
      <InputLabel key="lower-bound-input-label" htmlFor="lower-bound-input">
        <Text pl={[2]} fontSize={[1]}>
          {t("accountTables.search.lowerBound")}
        </Text>
      </InputLabel>,
      <UiInput
        key="lower-bound-input"
        id="lower-bound-input"
        disableUnderline={true}
        value={contractTableStore.lowerBound}
        placeholder={t("accountTables.search.lowerBound")}
        onChange={(e: any) => {
          contractTableStore.lowerBound = e.target.value
        }}
      />
    ]
  }

  render() {
    if (!this.props.abiLoader.abi) {
      return null
    }

    return (
      <Cell>
        <Grid
          ml={[4]}
          mb={[3]}
          maxWidth="600px"
          gridTemplateColumns={["auto", "200px 285px 265px 1fr", "200px 265px 265px 1fr"]}
          gridTemplateRows={["1fr 1fr 1fr 1fr 1fr", "auto"]}
        >
          <Cell gridColumn={["1"]} gridRow={["1", "1"]} mr={[3]}>
            {this.renderDropDown()}
          </Cell>
          <Cell gridColumn={["1", "2"]} gridRow={["2", "1"]} mt="2px">
            {this.renderScopeInput()}
          </Cell>
          <Cell gridColumn={["1", "3"]} gridRow={["3", "1"]} mt="2px">
            {this.renderLowerBoundInput()}
          </Cell>
          <Cell gridColumn={["1", "4"]} gridRow={["4", "1"]} ml={[0, 3]} mt={[3]}>
            <StyledButton onClick={this.onClick}>
              <Text fontWeight="bold" color="primary">
                {t("accountTables.search.load")}
              </Text>
            </StyledButton>
          </Cell>
        </Grid>
        <hr color="#f1f1f1" />
      </Cell>
    )
  }
}
