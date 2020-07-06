import * as React from "react"
import { GenericPillComponent, PillRenderingContext } from "../generic-pill.component"
import { Box } from "@dfuse/explorer"
import { FormattedText } from "../../../formatted-text/formatted-text"

export class RegProxyPillComponent extends GenericPillComponent {
  static requireFields: string[] = ["isproxy", "proxy"]

  static contextForRendering = (): PillRenderingContext => {
    return {
      networks: ["all"],
      validActions: [{ contract: "eosio", action: "regproxy" }]
    }
  }

  static i18n() {
    return {
      en: {
        regProxy: {
          summaryReg: "<0>{{account}}</0> registered as a proxy",
          summaryUnreg: "<0>{{account}}</0> unregistered as a proxy"
        }
      },
      zh: {
        regProxy: {
          summaryReg: "<0>{{account}}</0> 注册成为代理",
          summaryUnreg: "<0>{{account}}</0> 撤销了代理"
        }
      }
    }
  }

  renderContent = () => {
    const { action } = this.props

    const i18nKey = action.data.isproxy
      ? "pillTemplates.regProxy.summaryReg"
      : "pillTemplates.regProxy.summaryUnreg"

    const fields = [{ name: "account", type: "accountLink", value: action.data.proxy }]
    return (
      <Box fontSize={[1]} mx={[2]} minWidth="10px" alignItems="center">
        <FormattedText fields={fields} i18nKey={i18nKey} fontSize={[1]} />
      </Box>
    )
  }
}
