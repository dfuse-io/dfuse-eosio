import { observer } from "mobx-react"
import {
  ListTransactions,
  TransactionListInfo
} from "../../../components/list-transactions/list-transactions.component"
import * as React from "react"
import { Cell, Grid } from "../../../atoms/ui-grid/ui-grid.component"
import { Text } from "../../../atoms/text/text.component"
import { t } from "i18next"
import { theme } from "../../../theme"

interface Props {
  transactionList: TransactionListInfo[]
  navigationRender: JSX.Element
  accountName: string
}

@observer
export class AccountTransactionsContents extends React.Component<Props, any> {
  render() {
    return (
      <Cell>
        <Cell pl={[4]} pt={[4]}>
          <Text fontSize={[5]} color="header">
            {t("account.transactions.title")}
          </Text>
          <Text fontSize={[2]} mt={[1]} color={theme.colors.grey5}>
            {t("account.transactions.subTitle")}
          </Text>
        </Cell>
        <Grid gridTemplateColumns={["1fr"]}>
          <Cell justifySelf="right" alignSelf="right" p={[3]}>
            {this.props.navigationRender}
          </Cell>
        </Grid>
        <Cell overflowX="auto">
          <ListTransactions
            transactionInfos={this.props.transactionList}
            pageContext={{ accountName: this.props.accountName }}
          />
        </Cell>
        <Grid gridTemplateColumns={["1fr"]}>
          <Cell justifySelf="right" alignSelf="right" p={[3]}>
            {this.props.navigationRender}
          </Cell>
        </Grid>
      </Cell>
    )
  }
}
