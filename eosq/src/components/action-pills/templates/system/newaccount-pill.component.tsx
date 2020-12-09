import * as React from "react"
import { GenericPillComponent, PillRenderingContext } from "../generic-pill.component"
import { Box, PillLogoProps } from "@dfuse/explorer"
import { Grid } from "../../../../atoms/ui-grid/ui-grid.component"
import { FormattedText } from "../../../formatted-text/formatted-text"
import { getNewAccountLevel1Fields, getNewAccountLevel2Fields } from "../pill-template.helpers"

import { ACCOUNT_CREATORS } from "../all-logos"
import { Config } from "../../../../models/config"

export class NewAccountPillComponent extends GenericPillComponent {
  get logoParams(): PillLogoProps | undefined {
    const availableCreators = ACCOUNT_CREATORS[Config.network_id] || []

    const creatorData = availableCreators.find((creator: any) => {
      return creator.contract === this.props.action.data.creator
    })

    if (creatorData) {
      return {
        path: creatorData.path,
        website: creatorData.website,
      }
    }

    return undefined
  }

  static requireFields: string[] = ["creator", "name", "owner", "active"]

  static contextForRendering = (): PillRenderingContext => {
    return {
      networks: ["all"],
      validActions: [{ contract: "eosio", action: "newaccount" }],
    }
  }

  static i18n() {
    return {
      en: {
        newaccount: {
          summary: "<0>{{creator}}</0> created <1>{{name}}</1>",
          detailAccount:
            "<1>{{account}}</1>@<2>{{accountPermission}}</2> created for @<0>{{permission}}</0>",
          detailKey: "<1>{{key}}</1> created for @<0>{{permission}}</0>",
          detailWait: "<1>{{wait}}</1> created for @<0>{{permission}}</0>",
        },
      },
      zh: {
        newaccount: {
          summary: "<0>{{creator}}</0> 创建了 <1>{{name}}</1>",
          detailAccount:
            "<1>{{account}}</1>@<2>{{accountPermission}}</2> 已创建在 @<0>{{permission}}</0> 上",
          detailKey: "<1>{{key}}</1> 已创建在 @<0>{{permission}}</0> 上",
          detailWait: "<1>{{wait}}</1> 已创建在 @<0>{{permission}}</0> 上",
        },
      },
    }
  }

  renderContent = () => {
    return (
      <Box fontSize={[1]} mx={[2]} minWidth="10px" minHeight="26px" alignItems="center">
        <FormattedText
          fields={getNewAccountLevel1Fields(this.props.action)}
          i18nKey="pillTemplates.newaccount.summary"
          fontSize={[1]}
        />
      </Box>
    )
  }

  renderTemplateItemAccount = (permission: any, parent: string) => {
    return (
      <FormattedText
        fields={getNewAccountLevel2Fields(permission, parent, "account")}
        i18nKey="pillTemplates.newaccount.detailAccount"
        fontSize={[1]}
      />
    )
  }

  renderTemplateItemKey = (permission: any, parent: string) => {
    return (
      <FormattedText
        fields={getNewAccountLevel2Fields(permission, parent, "key")}
        i18nKey="pillTemplates.newaccount.detailKey"
        fontSize={[1]}
      />
    )
  }

  renderTemplateItemWait = (permission: any, parent: string) => {
    return (
      <FormattedText
        fields={getNewAccountLevel2Fields(permission, parent, "wait")}
        i18nKey="pillTemplates.newaccount.detailWait"
        fontSize={[11]}
      />
    )
  }

  renderLevel2Template = () => {
    const { data } = this.props.action

    if (data) {
      return (
        <Grid gridTemplateColumns={["1fr"]}>
          {this.renderLevel2TemplateContent(data, "owner", "accounts")}
          {this.renderLevel2TemplateContent(data, "owner", "keys")}
          {this.renderLevel2TemplateContent(data, "owner", "waits")}
          {this.renderLevel2TemplateContent(data, "active", "accounts")}
          {this.renderLevel2TemplateContent(data, "active", "keys")}
          {this.renderLevel2TemplateContent(data, "active", "waits")}
        </Grid>
      )
    }

    return null
  }

  renderLevel2TemplateContent(data: any, parent: string, type: "accounts" | "keys" | "waits") {
    const templateMethods = {
      accounts: this.renderTemplateItemAccount,
      keys: this.renderTemplateItemKey,
      waits: this.renderTemplateItemWait,
    }

    return (data[parent][type] || []).map((permission: any, index: number) => {
      return (
        <Box mx={[2]} key={index} minWidth="10px" minHeight="26px" alignItems="center">
          {templateMethods[type](permission, parent)}
        </Box>
      )
    })
  }
}
