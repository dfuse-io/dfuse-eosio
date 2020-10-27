import * as React from "react"
import { GenericPillComponent, PillRenderingContext } from "../generic-pill.component"
import { Box } from "@dfuse/explorer"
import { FormattedText } from "../../../formatted-text/formatted-text"
import { getLinkAuthLevel1Fields, getLinkAuthLevel2Fields } from "../pill-template.helpers"

export class LinkAuthPillComponent extends GenericPillComponent {
  static requireFields: string[] = ["account", "requirement", "type"]

  static contextForRendering = (): PillRenderingContext => {
    return {
      networks: ["all"],
      validActions: [{ contract: "eosio", action: "linkauth" }]
    }
  }

  static i18n() {
    return {
      en: {
        linkauth: {
          summary:
            "<0>{{account}}</0> linked <1>@{{requirement}}</1> to <2>{{type}}</2> in <3>{{code}}</3>",
          detail: "<0>@{{requirement}}</0> is now linked to  <1>{{type}}</1> in <2>{{code}}</2>"
        }
      },
      zh: {
        linkauth: {
          summary:
            "<0>{{account}}</0> 已连接 <1>@{{requirement}}</1> 到 <2>{{type}}</2> 于 <3>{{code}}</3>",
          detail: "<0>@{{requirement}}</0> 现已连接到  <1>{{type}}</1> 于 <2>{{code}}</2>"
        }
      }
    }
  }

  renderContent = () => {
    const { action } = this.props

    return (
      <Box fontSize={[1]} mx={[2]} minWidth="10px" minHeight="26px" alignItems="center">
        <FormattedText
          fields={getLinkAuthLevel1Fields(action)}
          i18nKey="pillTemplates.linkauth.summary"
          fontSize={[1]}
        />
      </Box>
    )
  }

  renderLevel2Template = () => {
    const { data } = this.props.action

    if (data) {
      return (
        <Box>
          <FormattedText
            fields={getLinkAuthLevel2Fields(this.props.action)}
            i18nKey="pillTemplates.linkauth.detail"
            fontSize={[1]}
          />
        </Box>
      )
    }

    return null
  }
}
