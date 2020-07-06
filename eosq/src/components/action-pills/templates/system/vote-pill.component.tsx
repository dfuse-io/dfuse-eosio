import * as React from "react"
import { GenericPillComponent, PillRenderingContext } from "../generic-pill.component"
import { MonospaceTextLink } from "../../../../atoms/text-elements/misc"
import { Links } from "../../../../routes"
import { Box } from "@dfuse/explorer"
import { Cell } from "../../../../atoms/ui-grid/ui-grid.component"
import { FormattedText } from "../../../formatted-text/formatted-text"

export class VotePillComponent extends GenericPillComponent {
  static requireFields: string[] = ["producers", "voter", "proxy"]

  static contextForRendering = (): PillRenderingContext => {
    return {
      networks: ["all"],
      validActions: [
        { contract: "eosio", action: "voteproxy" },
        { contract: "eosio", action: "voteproducer" }
      ]
    }
  }

  static i18n() {
    return {
      en: {
        voteForBp: {
          summary: "<0>{{account}}</0> voted for block producers"
        },
        voteByProxy: {
          summary: "<0>{{account}}</0> proxied their vote to  <1>{{proxy}}</1>"
        }
      },
      zh: {
        voteForBp: {
          summary: "<0>{{account}}</0> 已为BP节点投票"
        },
        voteByProxy: {
          summary: "<0>{{account}}</0> 将其投票代理于 <1>{{proxy}}</1>"
        }
      }
    }
  }

  renderContent = () => {
    const { action } = this.props

    if (action.data.proxy && action.data.proxy.length > 0) {
      const fields = [
        { name: "account", type: "accountLink", value: action.data.voter },
        { name: "proxy", type: "accountLink", value: action.data.proxy }
      ]

      return (
        <Box fontSize={[1]} mx={[2]} minWidth="10px" minHeight="26px" alignItems="center">
          <FormattedText
            fields={fields}
            i18nKey="pillTemplates.voteByProxy.summary"
            fontSize={[1]}
          />
        </Box>
      )
    }

    const fields = [{ name: "account", type: "accountLink", value: action.data.voter }]

    return (
      <Box fontSize={[1]} mx={[2]} minWidth="10px" minHeight="26px" alignItems="center">
        <FormattedText fields={fields} i18nKey="pillTemplates.voteForBp.summary" fontSize={[1]} />
      </Box>
    )
  }

  renderLevel2Template = () => {
    const { data } = this.props.action

    if (data) {
      return (
        <Cell whiteSpace="normal">
          {((data.producers as any[]) || []).map((accountName: string) => {
            return (
              <Cell pr={[3]} pb={[2]} display="inline-block" key={accountName}>
                <MonospaceTextLink to={Links.viewAccount({ id: accountName })}>
                  {accountName}
                </MonospaceTextLink>
              </Cell>
            )
          })}
        </Cell>
      )
    }

    return null
  }
}
