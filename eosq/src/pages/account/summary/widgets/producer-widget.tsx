import { observer } from "mobx-react"
import { Cell } from "../../../../atoms/ui-grid/ui-grid.component"
import * as React from "react"
import { Account } from "../../../../models/account"
import { DetailLine, formatTransactionID } from "@dfuse/explorer"
import { t } from "i18next"

import { Text, TextLink, ExternalTextLink } from "../../../../atoms/text/text.component"
import { MonospaceTextLink } from "../../../../atoms/text-elements/misc"
import { Links } from "../../../../routes"
import { theme, styled } from "../../../../theme"
import { getRankBgColor, getRankInfo, getRankStatus } from "../../../../helpers/account.helpers"
import { Vote } from "../../../../models/vote"
import { SocialLinks } from "../../../../atoms/social-links/social-links.component"
import { processSocialNetworkNames } from "../../../../helpers/social-networks.helper"
import { formatDateFromString } from "../../../../helpers/moment.helpers"
import { UiHrSpaced } from "../../../../atoms/ui-hr/ui-hr"
import { SearchShortcut } from "../../../../components/search-shortcut/search-shortcut"

const TitleHeader: React.ComponentType<any> = styled(Cell)`
  padding: 10px;
  margin-bottom: 30px;
`

interface Props {
  account: Account
  votes: Vote[]
}

@observer
export class ProducerWidget extends React.Component<Props, any> {
  render() {
    const { account } = this.props
    const rankInfo = getRankInfo(account, this.props.votes)
    const rankBgColor = getRankBgColor(rankInfo)
    const socialLinks = processSocialNetworkNames(account)

    return (
      <Cell pl={[4, 0, 0]} bg="white" pt={[4]}>
        <TitleHeader fontSize={[3]} bg={rankBgColor} color={theme.colors.primary}>
          <Text
            fontSize={[3]}
            display="inline-block"
            color={theme.colors.primary}
            fontWeight="bold"
          >
            {getRankStatus(rankInfo)}
          </Text>{" "}
          <SearchShortcut
            color={theme.colors.primary}
            fontSize={[3]}
            query={`receiver:producerjson auth:${account.account_name}`}
          >
            <Text color={theme.colors.primary} fontSize={[3]}>
              {t("account.summary.block_producer")}
            </Text>
          </SearchShortcut>
        </TitleHeader>
        <DetailLine variant="auto" label={t("account.summary.creation_date")}>
          <Text>{formatDateFromString(account.created, false)}</Text>
        </DetailLine>
        {account.creator
          ? [
              <DetailLine key="0" variant="auto" label={t("account.summary.created_by")}>
                <MonospaceTextLink to={Links.viewAccount({ id: account.creator.creator })}>
                  {account.creator.creator}
                </MonospaceTextLink>
              </DetailLine>,
              <DetailLine variant="auto" key="1" label={t("account.summary.creation_trx_id")}>
                <TextLink to={Links.viewTransaction({ id: account.creator.trx_id })}>
                  {formatTransactionID(account.creator.trx_id)}
                </TextLink>
              </DetailLine>
            ]
          : null}
        <UiHrSpaced />
        <DetailLine variant="auto" label={t("account.summary.website")}>
          <ExternalTextLink to={account.block_producer_info!.org.website}>
            {account.block_producer_info!.org.website}
          </ExternalTextLink>
        </DetailLine>
        <DetailLine variant="auto" label={t("account.summary.email")}>
          <ExternalTextLink to={`emailto:${account.block_producer_info!.org.email}`}>
            {account.block_producer_info!.org.email}
          </ExternalTextLink>
        </DetailLine>
        <DetailLine mb={2} variant="auto" label={t("account.summary.location")}>
          {account.block_producer_info!.org.location.name},{" "}
          {account.block_producer_info!.org.location.country}
        </DetailLine>
        <SocialLinks
          socialNetworks={socialLinks}
          verifiedTitle={t("account.social_links.verified_by")}
        />
      </Cell>
    )
  }
}
