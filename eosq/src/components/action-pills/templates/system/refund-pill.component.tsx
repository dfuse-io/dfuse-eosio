import * as React from "react"
import { GenericPillComponent, PillRenderingContext } from "../generic-pill.component"
import Box from "../../../../atoms/ui-box/ui-box.component"
import { getRefundLevel1Fields } from "../pill-template.helpers"
import { FormattedText } from "../../../formatted-text/formatted-text"

export class RefundPillComponent extends GenericPillComponent {
  static requireFields: string[] = ["owner"]

  static contextForRendering = (): PillRenderingContext => {
    return {
      networks: ["all"],
      validActions: [{ contract: "eosio", action: "refund" }]
    }
  }

  static i18n() {
    return {
      en: {
        refund: {
          summary: "<0>{{refundAmount}}</0> refunded to <1>{{owner}}</1>"
        }
      },
      zh: {
        refund: {
          summary: "<0>{{refundAmount}}</0> 已退给 <1>{{owner}}</1>"
        }
      }
    }
  }

  renderContent = (): JSX.Element => {
    if (this.hasInlineTraces()) {
      return (
        <Box fontSize={[1]} mx={[2]} minWidth="10px" minHeight="26px" alignItems="center">
          <FormattedText
            fields={getRefundLevel1Fields(this.props.action, this.props.traceInfo)}
            i18nKey="pillTemplates.refund.summary"
            fontSize={[1]}
          />
        </Box>
      )
    }

    return this.renderDefaultContent()
  }
}
