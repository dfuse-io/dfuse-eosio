import * as React from "react"
import { TransferBox } from "../../../../atoms/pills/pill-transfer-box"
import { GenericPillComponent, PillRenderingContext } from "../generic-pill.component"

export class IssuePillComponent extends GenericPillComponent {
  static requireFields: string[] = ["to", "quantity"]

  static contextForRendering = (): PillRenderingContext => {
    return {
      networks: ["all"],
      validActions: [{ action: "issue" }]
    }
  }

  renderContent = () => {
    const { action } = this.props

    return <TransferBox from={action.account} to={action.data.to} amount={action.data.quantity} />
  }
}
