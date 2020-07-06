import * as React from "react"
import { Box } from "@dfuse/explorer"
import { GenericPillComponent, PillRenderingContext } from "../generic-pill.component"
import { Trans } from "react-i18next"
import { EllipsisText, Text } from "../../../../atoms/text/text.component"
import { DetailLine } from "@dfuse/explorer"
import { t } from "i18next"

export class DecenTwitterTweetPillComponent extends GenericPillComponent {
  static requireFields: string[] = ["msg"]

  static contextForRendering = (): PillRenderingContext => {
    return {
      networks: ["eos-mainnet"],
      validActions: [{ contract: "decentwitter", action: "tweet" }],
      needsTranslate: true
    }
  }

  static i18n() {
    return {
      en: {
        tweet: {
          summary: " <0>Message:</0>{{message}}",
          detail: "Message:"
        }
      },
      zh: {
        tweet: {
          summary: " <0>发信：</0>{{message}}",
          detail: "发信："
        }
      }
    }
  }

  renderContent = () => {
    const { action } = this.props

    return (
      <Box fontSize={[1]} mx={[2]} minWidth="10px" alignItems="center">
        <EllipsisText fontSize={[1]}>
          <Trans
            i18nKey="pillTemplates.tweet.summary"
            values={{
              message: action.data.msg
            }}
            components={[
              <Text fontSize={[1]} display="inline-block" fontWeight="bold" mr={[1]} key="1">
                Message:
              </Text>
            ]}
          />
        </EllipsisText>
      </Box>
    )
  }

  renderLevel2Template = () => {
    const { data } = this.props.action

    if (data) {
      return <DetailLine label={t("pillTemplates.tweet.detail")}>{data.msg}</DetailLine>
    }

    return null
  }
}
