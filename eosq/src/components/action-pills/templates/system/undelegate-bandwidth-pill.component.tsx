import * as React from "react"
import { GenericPillComponent, PillRenderingContext } from "../generic-pill.component"
import Box from "../../../../atoms/ui-box/ui-box.component"
import { FormattedText } from "../../../formatted-text/formatted-text"
import { getUndelegatebwLevel1Fields, getUndelegatebwLevel2Fields } from "../pill-template.helpers"

export class UnDelegateBandwidthPillComponent extends GenericPillComponent {
  static requireFields: string[] = ["unstake_cpu_quantity", "unstake_net_quantity", "from"]

  static contextForRendering = (): PillRenderingContext => {
    return {
      networks: ["all"],
      validActions: [{ contract: "eosio", action: "undelegatebw" }]
    }
  }

  static i18n() {
    return {
      en: {
        undelegatebw: {
          summary:
            "<0>{{from}}</0> undelegated <1>{{amountCPU}}</1> from CPU and <2>{{amountNET}}</2> from NET",
          detail: "<0>{{total}}</0> currently unstaking"
        }
      },
      zh: {
        undelegatebw: {
          summary:
            "<0>{{from}}</0> 取消委托 <1>{{amountCPU}}</1> 的CPU 和 <2>{{amountNET}}</2> 的NET",
          detail: "<0>{{total}}</0> 目前正在被抵押"
        }
      }
    }
  }

  renderContent = () => {
    const { action } = this.props

    return (
      <Box fontSize={[1]} mx={[2]} minWidth="10px" minHeight="26px" alignItems="center">
        <FormattedText
          fields={getUndelegatebwLevel1Fields(action)}
          i18nKey="pillTemplates.undelegatebw.summary"
          fontSize={[1]}
        />
      </Box>
    )
  }

  renderLevel2Template = () => {
    const { action } = this.props
    if (action.data) {
      return (
        <Box>
          <FormattedText
            fields={getUndelegatebwLevel2Fields(action)}
            i18nKey="pillTemplates.undelegatebw.detail"
            fontSize={[1]}
          />
        </Box>
      )
    }

    return null
  }
}
