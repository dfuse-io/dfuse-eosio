import * as React from "react"
import { Cell, Grid } from "../../atoms/ui-grid/ui-grid.component"
import { styled } from "../../theme"
import Button from "@material-ui/core/Button/Button"
import { DropDownOption, UiDropDown } from "../../atoms/ui-dropdown/ui-dropdown.component"
import { getSearchSystemOptions, getSearchTransfersOptions } from "../../helpers/search.helpers"
import { Text } from "../../atoms/text/text.component"
import { theme } from "../../theme"
import { t } from "i18next"

const SelectorButton: React.ComponentType<any> = styled(Button)`
  white-space: nowrap;
  width: 100% !important;
  padding-left: 20px !important;
  padding-right: 20px !important;
  background-color: ${(props) => props.theme.colors.primary} !important;
  border: none !important;
  font-weight: bold !important;
  border-radius: 0px !important;
  height: 32px !important;
  min-height: 32px !important;
  color: ${(props) => props.theme.colors.text} !important;
`

interface Props {
  chevron: boolean
  resetHard?: boolean
  accountName?: string
  defaultQuery?: string
  onSubmit: (query: string) => void
}

interface State {
  query: string
}

export class SearchTransactions extends React.Component<Props, State> {
  static defaultProps = {
    chevron: false,
    resetHard: false
  }

  constructor(props: Props) {
    super(props)

    this.state = { query: props.defaultQuery || "" }
  }

  componentDidUpdate(prevProps: Readonly<Props>): void {
    if (this.props.defaultQuery !== prevProps.defaultQuery) {
      this.setState({ query: this.props.defaultQuery || "" })
    }
  }

  onSelect = (query: string) => {
    this.setState({ query: query }, () => this.props.onSubmit(this.state.query))
  }

  renderSelectors() {
    if (!this.props.accountName) {
      return null
    }

    const transferOptions = getSearchTransfersOptions(this.props.accountName)

    const systemOptions: DropDownOption[] = getSearchSystemOptions(this.props.accountName)

    return (
      <Grid
        pb={[4]}
        gridColumnGap={[1, 2, 4]}
        gridTemplateColumns={["1fr", "1fr 1fr 1fr 1fr", "1fr 1fr 1fr 1fr"]}
      >
        <Cell>
          <Cell p={[2]}>
            <Text color={theme.colors.grey5} fontSize={[2]}>
              {t("transactionSearch.buttonLabels.account")}
            </Text>
          </Cell>
          <SelectorButton onClick={() => this.onSelect(`auth:${this.props.accountName}`)}>
            {t("transactionSearch.buttons.signedBy", { accountName: this.props.accountName })}
          </SelectorButton>
        </Cell>
        <Cell>
          <Cell p={[2]} height="34px">
            {" "}
            <Text color={theme.colors.grey5} fontSize={[2]} />
          </Cell>
          <SelectorButton onClick={() => this.onSelect(`receiver:${this.props.accountName}`)}>
            {t("transactionSearch.buttons.notifications")}
          </SelectorButton>
        </Cell>
        <Cell>
          <Cell p={[2]}>
            <Text color={theme.colors.grey5} fontSize={[2]}>
              {t("transactionSearch.buttonLabels.tokens")}
            </Text>
          </Cell>
          <UiDropDown
            defaultValue={transferOptions[0].value}
            id="transfers-filter"
            options={transferOptions}
            onSelect={(query: string) => this.onSelect(query)}
            value={this.state.query}
          />
        </Cell>
        <Cell>
          <Cell p={[2]}>
            <Text color={theme.colors.grey5} fontSize={[2]}>
              {t("transactionSearch.buttonLabels.system")}
            </Text>
          </Cell>
          <UiDropDown
            defaultValue={systemOptions[0].value}
            id="system-filter"
            options={systemOptions}
            onSelect={(query: string) => this.onSelect(query)}
            value={this.state.query}
          />
        </Cell>
      </Grid>
    )
  }

  render() {
    return (
      <Grid px={[4]} py={[3]} mb={[4]} gridTemplateRows={["1fr auto auto"]}>
        {this.renderSelectors()}
      </Grid>
    )
  }
}
