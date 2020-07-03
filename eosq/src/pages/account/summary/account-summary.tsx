import { observer } from "mobx-react"
import { ContentLoaderComponent } from "../../../components/content-loader/content-loader.component"
import { Cell, Grid } from "../../../atoms/ui-grid/ui-grid.component"
import { AccountPieChart } from "./account-pie-chart"
import { AccountStatusBars } from "./account-status-bars"
import { AccountPermissions } from "./account-permissions"
import * as React from "react"
import { Account } from "../../../models/account"
import { DefaultAccountWidget } from "./widgets/default-account-widget"
import { ProducerWidget } from "./widgets/producer-widget"
import { voteStore } from "../../../stores"
import { AccountTokens } from "./account-tokens"
import { UiHrDense } from "../../../atoms/ui-hr/ui-hr"
import { useAccountBalances } from "../../../hooks/use-account-balances"
// temp ignore for dev

import { LCE } from "@dfuse/explorer"
import { Config } from "../../../models/config"

interface Props {
  account: Account
}

const Widget: React.FC<Props> = ({ account }) => {
  if (account.block_producer_info && voteStore.votes && voteStore.votes.length > 0) {
    return <ProducerWidget account={account} votes={voteStore.votes} />
  }

  return <DefaultAccountWidget account={account} />
}

const Tokens: React.FC<{ accountName: string }> = ({ accountName }) => {
  const response = useAccountBalances(accountName)
  const tokens = response.resultOr([])

  return (
    <LCE promise={response}>
      {tokens.length <= 0 ? null : (
        <>
          <AccountTokens account={accountName} tokens={tokens} />
          <UiHrDense />
        </>
      )}
    </LCE>
  )
}

@observer
export class AccountSummary extends ContentLoaderComponent<Props, any> {
  render() {
    const { account } = this.props

    return (
      <Cell bg="white" pb={[0]} border="1px solid" borderColor="border">
        <Grid
          gridTemplateColumns={["1fr", "5fr 2fr", "5fr 2fr"]}
          height="auto"
          gridColumnGap={[30]}
          borderBottom="1px dotted"
          borderColor="grey6"
          pb="40px"
        >
          <Grid px={[4]}>
            <AccountPieChart account={account} />
            <Cell mt={[2]}>
              <UiHrDense />
              <AccountStatusBars account={account} />
            </Cell>
          </Grid>
          <Cell pr={[4]}>
            <Widget account={account} />
          </Cell>
        </Grid>
        {Config.disable_token_meta ? null : <Tokens accountName={account.account_name} />}
        <AccountPermissions account={account} />
      </Cell>
    )
  }
}
