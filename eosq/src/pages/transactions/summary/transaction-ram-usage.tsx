import { observer } from "mobx-react"
import * as React from "react"
import { Cell } from "../../../atoms/ui-grid/ui-grid.component"
import { TransactionLifecycle } from "@dfuse/client"
import { translate } from "react-i18next"
import { RamUsage } from "../../../components/ram-usage/ram-usage.component"
import { summarizeRamOps } from "../../../helpers/transaction.helpers"

interface Props {
  transactionLifeCycle: TransactionLifecycle
}

@observer
class BaseTransactionRamUsage extends React.Component<Props> {
  render() {
    const ramops = summarizeRamOps(this.props.transactionLifeCycle.ramops || [])
    return (
      <Cell p="20px" bg="white">
        <RamUsage type="summary" ramops={ramops} />
      </Cell>
    )
  }
}

export const TransactionRamUsage = translate()(BaseTransactionRamUsage)
