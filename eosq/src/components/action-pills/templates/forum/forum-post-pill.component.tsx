/* eslint-disable no-prototype-builtins */
import * as React from "react"
import { Box } from "@dfuse/explorer"
import { GenericPillComponent, PillRenderingContext } from "../generic-pill.component"
import { Trans } from "react-i18next"
import { Text } from "../../../../atoms/text/text.component"
import { Cell } from "../../../../atoms/ui-grid/ui-grid.component"
import { debugLog } from '../../../../services/logger'

export class ForumPostPillComponent extends GenericPillComponent {
  static requireFields: string[] = [
    "certify",
    "content",
    "json_metadata",
    "post_uuid",
    "poster",
    "reply_to_post_uuid",
    "reply_to_poster"
  ]

  static contextForRendering = (): PillRenderingContext => {
    return {
      networks: ["eos-mainnet"],
      validActions: [{ contract: "eosforumdapp", action: "post" }],
      needsTranslate: true
    }
  }

  static i18n() {
    return {
      en: {
        forumPost: {
          summary: "<0>Title:</0> {{title}} <1>Content:</1> {{content}}",
          detail: "<0>Sub:</0> {{sub}} <1></1><2>Attachment:</2> {{attachment}}"
        }
      },
      zh: {
        forumPost: {
          summary: "<0>标题：</0> {{title}} <1>内容：</1> {{content}}",
          detail: "<0>Sub:</0> {{sub}} <1></1><2>附件：</2> {{attachment}}"
        }
      }
    }
  }

  checkJson = (jsonData: any) => {
    return (
      jsonData.hasOwnProperty("type") &&
      jsonData.hasOwnProperty("title") &&
      jsonData.hasOwnProperty("sub") &&
      jsonData.hasOwnProperty("attachment") &&
      jsonData.attachment.hasOwnProperty("value")
    )
  }

  parseProposalJson = (data: any) => {
    try {
      return JSON.parse(data.json_metadata)
    } catch {
      debugLog("Couldn't parse post JSON")
      return {}
    }
  }

  renderContent = () => {
    const { action } = this.props
    const jsonData = this.parseProposalJson(action.data)

    if (this.checkJson(jsonData)) {
      return (
        <Box fontSize={[1]} mx={[2]} minWidth="10px" alignItems="center">
          <Trans
            i18nKey="pillTemplates.forumPost.summary"
            values={{
              content: action.data.content,
              title: jsonData.title
            }}
            components={[
              <Text fontSize={[1]} fontWeight="bold" mr={[1]} key="1">
                Title:
              </Text>,
              <Text fontSize={[1]} fontWeight="bold" ml={[1]} mr={[1]} key="2">
                Content:
              </Text>
            ]}
          />
        </Box>
      )
    }

    return this.renderDefaultContent()
  }

  renderLevel2Template = () => {
    const { data } = this.props.action
    const jsonData = this.parseProposalJson(data)

    if (data && this.checkJson(jsonData)) {
      return (
        <Cell>
          <Trans
            i18nKey="pillTemplates.forumPost.detail"
            values={{
              sub: jsonData.sub,
              attachment: jsonData.attachment.value
            }}
            components={[
              <Text display="inline-block" fontWeight="bold" mr={[1]} key="1">
                Sub:
              </Text>,
              <br key="2" />,
              <Text display="inline-block" fontWeight="bold" mr={[1]} key="3">
                Attachement:
              </Text>
            ]}
          />
        </Cell>
      )
    }

    return null
  }
}
