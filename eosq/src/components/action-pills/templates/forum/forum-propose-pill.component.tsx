import * as React from "react"
import { Box } from "@dfuse/explorer"
import { GenericPillComponent, PillRenderingContext } from "../generic-pill.component"
import { Trans } from "react-i18next"
import { Text } from "../../../../atoms/text/text.component"
import { Cell } from "../../../../atoms/ui-grid/ui-grid.component"

export class ForumProposePillComponent extends GenericPillComponent {
  static requireFields: string[] = [
    "expires_at",
    "proposal_json",
    "proposal_name",
    "proposer",
    "title"
  ]

  static contextForRendering = (): PillRenderingContext => {
    return {
      networks: ["eos-mainnet"],
      validActions: [{ contract: "eosforumrcpp", action: "propose" }],
      needsTranslate: true
    }
  }

  static i18n() {
    return {
      en: {
        forumPropose: {
          summary:
            "<0>Proposal Name:</0> {{proposalName}} <1>Title:</1> {{title}} <2>Expires At:</2> {{expiresAt}}",
          detail: "<0>Content:</0> {{content}}"
        }
      },
      zh: {
        forumPropose: {
          summary:
            "<0>提案名称：</0> {{proposalName}} <1>标题：</1> {{title}} <2>过期时间：</2> {{expiresAt}}",
          detail: "<0>内容：</0> {{content}}"
        }
      }
    }
  }

  checkJson = (jsonData: any) => {
    // eslint-disable-next-line no-prototype-builtins
    return jsonData.hasOwnProperty("type") && jsonData.hasOwnProperty("content")
  }

  parseProposalJson = (data: any) => {
    try {
      return JSON.parse(data.json_metadata)
    } catch {
      console.warn("Couldn't parse proposal JSON")
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
            i18nKey="pillTemplates.forumPropose.summary"
            values={{
              proposalName: action.data.proposal_name,
              title: action.data.title,
              expiresAt: action.data.expires_at
            }}
            components={[
              <Text fontSize={[1]} fontWeight="bold" mr={[1]} key="1">
                Proposal Name:
              </Text>,
              <Text fontSize={[1]} fontWeight="bold" ml={[1]} mr={[1]} key="2">
                Title:
              </Text>,
              <Text fontSize={[1]} fontWeight="bold" ml={[1]} mr={[1]} key="3">
                Expires At:
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
    if (!data) {
      return <span />
    }

    const jsonData = this.parseProposalJson(data)

    if (data && this.checkJson(jsonData)) {
      return (
        <Cell>
          <Trans
            i18nKey="pillTemplates.forumPropose.detail"
            values={{
              content: jsonData.content
            }}
            components={[
              <Text display="inline-block" fontWeight="bold" mr={[1]} key="1">
                Content:
              </Text>
            ]}
          />
        </Cell>
      )
    }

    return null
  }
}
