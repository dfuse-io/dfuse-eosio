import * as React from "react"
import { GenericPillComponent, PillRenderingContext } from "../generic-pill.component"
import Box from "../../../../atoms/ui-box/ui-box.component"
import { Grid } from "../../../../atoms/ui-grid/ui-grid.component"
import { FormattedText } from "../../../formatted-text/formatted-text"
import { getUpdateAuthLevel1Fields, getUpdateAuthLevel2Fields } from "../pill-template.helpers"

export class UpdateAuthPillComponent extends GenericPillComponent {
  static requireFields: string[] = ["auth", "parent", "permission"]

  static contextForRendering = (): PillRenderingContext => {
    return {
      networks: ["all"],
      validActions: [{ contract: "eosio", action: "updateauth" }]
    }
  }

  static i18n() {
    return {
      en: {
        updateauth: {
          summary: "<0>{{account}}</0> updated the permission for <1>@{{permission}}</1>",
          detailAccount:
            "@<0>{{permission}}</0> updated to <1>{{account}}</1>@<2>{{accountPermission}}</2> under @<3>{{parent}}<3>",
          detailKey: "@<0>{{permission}}</0> updated to <1>{{key}}</1> under @<2>{{parent}}</2>",
          detailWait:
            "@<0>{{permission}}</0> updated to <1>{{wait}}</1> seconds under @<2>{{parent}}</2>",
          detailAccountNoParent:
            "@<0>{{permission}}</0> updated to <1>{{account}}</1>@<2>{{accountPermission}}</2>",
          detailWaitNoParent: "@<0>{{permission}}</0> updated to <1>{{wait}}</1>",
          detailKeyNoParent: "@<0>{{permission}}</0> updated to <1>{{key}}</1>"
        }
      },
      zh: {
        updateauth: {
          summary: "<0>{{account}}</0> 为 <1>@{{permission}}</1> 更新了权限",
          detailAccount:
            "@<0>{{permission}}</0> 更新为 <1>{{account}}</1>@<2>{{accountPermission}}</2> ，嵌套在 @<3>{{parent}}<3> 之下",
          detailKey:
            "@<0>{{permission}}</0> 更新为 <1>{{key}}</1> ，嵌套在 @<2>{{parent}}</2> 之下",
          detailWait:
            "@<0>{{permission}}</0> 更新为 <1>{{wait}}</1> 秒，嵌套在 @<2>{{parent}}</2> 之下",
          detailAccountNoParent:
            "@<0>{{permission}}</0> 更新为 <1>{{account}}</1>@<2>{{accountPermission}}</2>",
          detailWaitNoParent: "@<0>{{permission}}</0> 更新为 <1>{{wait}}</1>",
          detailKeyNoParent: "@<0>{{permission}}</0> 更新为 <1>{{key}}</1>"
        }
      }
    }
  }

  renderContent = () => {
    const { action } = this.props

    return (
      <Box fontSize={[1]} mx={[2]} minWidth="10px" minHeight="26px" alignItems="center">
        <FormattedText
          fields={getUpdateAuthLevel1Fields(action)}
          i18nKey="pillTemplates.updateauth.summary"
          fontSize={[1]}
        />
      </Box>
    )
  }

  renderTemplateItemAccount = (accountPermission: any, data: any) => {
    const i18nKey =
      data.parent && data.parent.length > 0
        ? "pillTemplates.updateauth.detailAccount"
        : "pillTemplates.updateauth.detailAccountNoParent"

    return (
      <FormattedText
        fields={getUpdateAuthLevel2Fields(accountPermission, data, "account")}
        i18nKey={i18nKey}
        fontSize={[1]}
      />
    )
  }

  renderTemplateItemKey = (keyPermission: any, data: any) => {
    const i18nKey =
      data.parent && data.parent.length > 0
        ? "pillTemplates.updateauth.detailKey"
        : "pillTemplates.updateauth.detailKeyNoParent"

    return (
      <FormattedText
        fields={getUpdateAuthLevel2Fields(keyPermission, data, "key")}
        i18nKey={i18nKey}
        fontSize={[1]}
      />
    )
  }

  renderTemplateItemWait = (waitPermission: any, data: any) => {
    const i18nKey =
      data.parent && data.parent.length > 0
        ? "pillTemplates.updateauth.detailWait"
        : "pillTemplates.updateauth.detailWaitNoParent"

    return (
      <FormattedText
        fields={getUpdateAuthLevel2Fields(waitPermission, data, "wait")}
        i18nKey={i18nKey}
        fontSize={[1]}
      />
    )
  }

  renderLevel2Template = () => {
    const { data } = this.props.action

    if (data) {
      return (
        <Grid gridTemplateColumns={["1fr"]}>
          {(data.auth.accounts || []).map((accountPermission: any, index: number) => {
            return (
              <Box mx={[2]} key={index} minWidth="10px" minHeight="26px" alignItems="center">
                {this.renderTemplateItemAccount(accountPermission, data)}
              </Box>
            )
          })}
          {(data.auth.keys || []).map((keyPermission: any, index: number) => {
            return (
              <Box mx={[2]} key={index} minWidth="10px" minHeight="26px" alignItems="center">
                {this.renderTemplateItemKey(keyPermission, data)}
              </Box>
            )
          })}
          {(data.auth.waits || []).map((waitPermission: any, index: number) => {
            return (
              <Box mx={[2]} key={index} minWidth="10px" minHeight="26px" alignItems="center">
                {this.renderTemplateItemWait(waitPermission, data)}
              </Box>
            )
          })}
        </Grid>
      )
    }

    return null
  }
}
