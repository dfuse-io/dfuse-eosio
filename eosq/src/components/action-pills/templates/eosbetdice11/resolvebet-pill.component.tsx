import * as React from "react"
import { GenericPillComponent } from "../generic-pill.component"
import { Trans } from "react-i18next"
import Box from "../../../../atoms/ui-box/ui-box.component"
import { Text } from "../../../../atoms/text/text.component"
import { Cell } from "../../../../atoms/ui-grid/ui-grid.component"
import { getResolveBetAmounts, getResolveBetLevel1Fields } from "../pill-template.helpers"
import { FormattedText } from "../../../formatted-text/formatted-text"

export class ResolveBetPillComponent extends GenericPillComponent {
  static requireFields = ["bet_id", "sig"]

  static contextForRendering() {
    return {
      networks: ["eos-mainnet"],
      needsTranslate: true,
      validActions: [{ contract: "eosbetdice11", action: "resolvebet" }]
    }
  }

  static i18n() {
    return {
      en: {
        resolveBet: {
          summary: "<0>{{account}}</0> won <1>{{EOSAmount}}</1> on bet {{betId}}",
          detail: "<0>Bet ID:</0> {{betId}}"
        }
      },
      zh: {
        resolveBet: {
          summary: "<0>{{account}}</0> 的赌注 {{betId}} 赢了 <1>{{EOSAmount}}</1> ",
          detail: "<0>赌注 ID:</0> {{betId}}"
        }
      }
    }
  }

  renderContent = () => {
    const { action } = this.props

    if (this.hasInlineTraces()) {
      const traceData = getResolveBetAmounts(this.props.traceInfo)

      if (traceData[2].length > 0) {
        return (
          <Box fontSize={[1]} mx={[2]} minWidth="10px" alignItems="center">
            <FormattedText
              fields={getResolveBetLevel1Fields(action, this.props.traceInfo)}
              i18nKey="pillTemplates.resolveBet.summary"
              fontSize={[1]}
            />
          </Box>
        )
      }
    }
    return this.renderDefaultContent()
  }

  renderLevel2Template = () => {
    const { data } = this.props.action
    const i18nKey = "pillTemplates.resolveBet.detail"
    if (data) {
      return (
        <Cell>
          <Trans
            i18nKey={i18nKey}
            values={{
              betId: data.bet_id
            }}
            components={[
              <Text
                display="inline-block"
                fontSize={[1]}
                fontWeight="bold"
                ml={[1]}
                mr={[1]}
                key="1"
              >
                Bet Id:
              </Text>
            ]}
          />
        </Cell>
      )
    }

    return null
  }
}
