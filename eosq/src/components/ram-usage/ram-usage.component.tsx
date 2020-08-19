import { observer } from "mobx-react"
import * as React from "react"
import { Cell } from "../../atoms/ui-grid/ui-grid.component"
import { RAMOp } from "@dfuse/client"
import { Links } from "../../routes"
import { formatBytes, Box } from "@dfuse/explorer"

import { FormattedText } from "../formatted-text/formatted-text"
import { t } from "i18next"

interface Props {
  ramops: RAMOp[]
  type: string
}

@observer
export class RamUsage extends React.Component<Props> {
  renderContent() {
    return this.props.ramops.map((ramop: RAMOp, index: number) => {
      const i18nKey =
        ramop.delta < 0 ? "transaction.ramUsage.released" : "transaction.ramUsage.consumed"

      const fields = [
        {
          type: "accountLink",
          value: ramop.payer,
          name: "accountName"
        },
        { type: "bold", value: formatBytes(Math.abs(ramop.delta)), name: "bytes" },
        { type: "bold", value: formatBytes(ramop.usage, 21000), name: "totalBytes" }
      ]

      return (
        <Box
          key={index}
          fontSize={[2]}
          mx={[2]}
          minWidth="10px"
          minHeight="26px"
          alignItems="center"
        >
          <FormattedText fontSize={[2]} i18nKey={i18nKey} fields={fields} />
        </Box>
      )
    })
  }

  renderContentDetail() {
    return this.props.ramops.map((ramop: RAMOp, index: number) => {
      const i18nKey =
        ramop.delta < 0
          ? "transaction.ramUsage.releasedDetail"
          : "transaction.ramUsage.consumedDetail"

      const fields = [
        {
          type: "accountLink",
          value: ramop.payer,
          name: "accountName",
          link: Links.viewAccount({ id: ramop.payer })
        },
        { type: "bold", value: formatBytes(Math.abs(ramop.delta)), name: "bytes" },
        {
          type: "plain",
          value: t(`transaction.ramUsage.operations.${ramop.op}`),
          name: "operation"
        }
      ]

      return (
        <Box key={index} fontSize={[1]} minWidth="10px" minHeight="26px" alignItems="center">
          <FormattedText fontSize={[1]} i18nKey={i18nKey} fields={fields} />
        </Box>
      )
    })
  }

  render() {
    return (
      <Cell>
        <Cell minWidth="800px" width="100%">
          {this.props.type === "detailed" ? this.renderContentDetail() : this.renderContent()}
        </Cell>
      </Cell>
    )
  }
}
