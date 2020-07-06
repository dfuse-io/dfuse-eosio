import * as React from "react"
import { Box } from "@dfuse/explorer"
import { GenericPillComponent } from "../generic-pill.component"
import { Trans } from "react-i18next"
import { Text } from "../../../../atoms/text/text.component"
import { extractValueWithUnits } from "@dfuse/explorer"
import { Cell } from "../../../../atoms/ui-grid/ui-grid.component"
import { FormattedText } from "../../../formatted-text/formatted-text"
import { getBetReceiptLevel1Fields } from "../pill-template.helpers"

export class BetReceiptPillComponent extends GenericPillComponent {
  static requireFields = [
    "amt_contract",
    "bet_amt",
    "bet_id",
    "bettor",
    "payout",
    "random_roll",
    "roll_under",
    "seed",
    "signature"
  ]

  static contextForRendering() {
    return {
      networks: ["eos-mainnet"],
      validActions: [{ contract: "eosbetdice11", action: "betreceipt" }],
      needsTranslate: true
    }
  }

  static i18n() {
    return {
      en: {
        betReceipt: {
          summaryWon: "<0>{{account}}</0> won <1>{{EOSAmount}}</1> on a roll of <2>{{roll}}</2>",
          summaryLost: "{{account}} lost {{EOSAmount}} on a roll of {{roll}}",
          detail:
            "<0>Roll Under:</0> {{rollUnder}} <1></1><2>Earnings:</2> {{EOSAmount}} <3></3><4>Seed:</4> {{seed}}"
        }
      },
      zh: {
        betReceipt: {
          summaryWon: "<0>{{account}}</0> 掷出 <2>{{roll}}</2> 而赢得 <1>{{EOSAmount}}</1>",
          summaryLost: "{{account}} 掷出 {{roll}} 而输掉 {{EOSAmount}} ",
          detail:
            "<0>掷出低于：</0> {{rollUnder}} <1></1><2>收益：</2> {{EOSAmount}} <3></3><4>种子：</4> {{seed}}"
        }
      }
    }
  }

  calculateBalance(): [number, string] {
    const { action } = this.props
    const [betAmount, unit] = extractValueWithUnits(action.data.bet_amt)
    const payout = extractValueWithUnits(action.data.payout)[0]

    const balance = parseFloat(payout) - parseFloat(betAmount)
    return [balance, unit]
  }

  renderContent = () => {
    const { action } = this.props

    const [balance] = this.calculateBalance()
    const i18nKey =
      balance < 0 ? "pillTemplates.betReceipt.summaryLost" : "pillTemplates.betReceipt.summaryWon"

    return (
      <Box fontSize={[1]} mx={[2]} minWidth="10px" alignItems="center">
        <FormattedText
          i18nKey={i18nKey}
          fontSize={[1]}
          fields={getBetReceiptLevel1Fields(action)}
        />
      </Box>
    )
  }

  renderLevel2Template = () => {
    const { data } = this.props.action
    const i18nKey = "pillTemplates.betReceipt.detail"
    const [balance, unit] = this.calculateBalance()
    if (data) {
      return (
        <Cell>
          <Trans
            i18nKey={i18nKey}
            values={{
              rollUnder: data.roll_under,
              EOSAmount: `${balance.toFixed(4)} ${unit}`,
              seed: data.seed
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
                Roll under:
              </Text>,
              <br key="2" />,
              <Text
                display="inline-block"
                fontSize={[1]}
                fontWeight="bold"
                ml={[1]}
                mr={[1]}
                key="3"
              >
                Earnings:
              </Text>,
              <br key="4" />,
              <Text
                display="inline-block"
                fontSize={[1]}
                fontWeight="bold"
                ml={[1]}
                mr={[1]}
                key="5"
              >
                Seed:
              </Text>
            ]}
          />
        </Cell>
      )
    }

    return null
  }
}
