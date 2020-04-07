import * as React from "react"
import { GenericPillComponent, PillRenderingContext } from "../generic-pill.component"
import Box from "../../../../atoms/ui-box/ui-box.component"
import { FormattedText } from "../../../formatted-text/formatted-text"
import { getDelegatebwLevel1Fields, getDelegatebwLevel2Fields } from "../pill-template.helpers"

export class DelegateBandwidthPillComponent extends GenericPillComponent {
  static requireFields: string[] = ["from", "receiver", "stake_cpu_quantity", "stake_net_quantity"]

  static contextForRendering = (): PillRenderingContext => {
    return {
      networks: ["all"],
      validActions: [{ contract: "eosio", action: "delegatebw" }]
    }
  }

  static i18n() {
    return {
      en: {
        delegatebw: {
          summary:
            "<0>{{from}}</0> delegated <1>{{amountCPU}}</1> for CPU and <2>{{amountNET}}</2> for NET to <3>{{to}}</3>",
          detail: "Delegated <0>{{amountCPU}}</0> for CPU and <1>{{amountNET}}</1> for Network"
        }
      },
      zh: {
        delegatebw: {
          summary:
            "<0>{{from}}</0> 委托 <1>{{amountCPU}}</1> 到CPU 和 <2>{{amountNET}}</2> 到NET 给 <3>{{to}}</3>",
          detail: "委托 <0>{{amountCPU}}</0> 到CPU <1>{{amountNET}}</1> 到网络带宽"
        }
      }
    }
  }

  renderContent = () => {
    const { action } = this.props

    return (
      <Box fontSize={[1]} mx={[2]} minWidth="10px" minHeight="26px" alignItems="center">
        <FormattedText
          fields={getDelegatebwLevel1Fields(action)}
          i18nKey="pillTemplates.delegatebw.summary"
          fontSize={[1]}
        />
      </Box>
    )
  }

  renderLevel2Template = () => {
    const { data } = this.props.action.data

    if (data) {
      return (
        <Box>
          <FormattedText
            fields={getDelegatebwLevel2Fields(this.props.action)}
            i18nKey="pillTemplates.delegatebw.detail"
            fontSize={[1]}
          />
        </Box>
      )
    }

    return null
  }
}
