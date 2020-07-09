import * as React from "react"
import { GenericPillComponent, PillRenderingContext } from "../generic-pill.component"
import { Box, hex2binary, hex2sha256 } from "@dfuse/explorer"
import { EllipsisText, ExternalTextLink } from "../../../../atoms/text/text.component"
import { MonospaceTextLink } from "../../../../atoms/text-elements/misc"
import { Links } from "../../../../routes"
import { Cell } from "../../../../atoms/ui-grid/ui-grid.component"
import { getBlobUrlFromPayload } from "../pill-template.helpers"

import { t } from "i18next"

export class SetcodePillComponent extends GenericPillComponent {
  static requireFields: string[] = ["unstake_cpu_quantity", "unstake_net_quantity", "from"]
  downloadUrl = ""

  static contextForRendering = (): PillRenderingContext => {
    return {
      networks: ["all"],
      needsTranslate: true,
      validActions: [{ contract: "eosio", action: "setcode" }]
    }
  }

  static i18n() {
    return {
      en: {
        downloadCode: "Download code"
      },
      zh: {
        downloadCode: "下载代码"
      }
    }
  }

  renderContent = (): JSX.Element => {
    const { data } = this.props.action
    const sha = hex2sha256(data.code)

    return (
      <Box fontSize={[1]} mx={[2]} minWidth="10px" alignItems="center">
        <EllipsisText fontSize={[1]}>
          SHA256[ <b>{sha}</b> ] set for account{" "}
          <MonospaceTextLink fontSize={[1]} to={Links.viewAccount({ id: data.account })}>
            {data.account}
          </MonospaceTextLink>
        </EllipsisText>
      </Box>
    )
  }

  renderLevel2Template = (): JSX.Element | null => {
    const { data } = this.props.action
    if (!data) {
      return <span />
    }

    const [sha, downloadUrl] = getBlobUrlFromPayload(hex2binary(data.code), this.downloadUrl)
    this.downloadUrl = downloadUrl

    // Tie the addressable version of the blob to the download link.
    return (
      <Cell>
        <ExternalTextLink download={`${data.account}_${sha}.wasm`} to={downloadUrl}>
          {t("pillTemplates.downloadCode")}
        </ExternalTextLink>
      </Cell>
    )
  }
}
