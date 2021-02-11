import { observer } from "mobx-react"
import { ContentLoaderComponent } from "../../../components/content-loader/content-loader.component"
import { Cell, Grid } from "../../../atoms/ui-grid/ui-grid.component"
import * as React from "react"
import { Badge, BadgeContainer } from "../../../atoms/badge/badge"
import { t } from "i18next"
import { theme, styled } from "../../../theme"
import { getRankBgColor, getRankInfo } from "../../../helpers/account.helpers"
import { voteStore } from "../../../stores"
import { ColorTile } from "../../../atoms/color-tile/color-tile"
import { SubTitle, ExternalTextLink } from "../../../atoms/text/text.component"
import { Account, BlockProducerInfo } from "../../../models/account"
import { MonospaceText } from "../../../atoms/text-elements/misc"
import { Config } from "../../../models/config"

interface Props {
  account: Account
}

const AccountImg: React.ComponentType<any> = styled.img`
  width: 70px;
`

const ClickableBadge = styled(Badge)`
  cursor: pointer;
`

const StyledObject: React.ComponentType<any> = styled.object`
  background-color: #fff;

  width: 70px;
  height: 70px;
`

@observer
export class AccountTitle extends ContentLoaderComponent<Props, any> {
  renderRank = (account: Account) => {
    const rankInfo = getRankInfo(account, voteStore.votes)
    const bgColor = getRankBgColor(rankInfo)

    if (rankInfo.rank > 0) {
      return (
        <ColorTile
          border={`1px solid ${theme.colors.primary}`}
          bg={bgColor}
          color="primary"
          fontWeight="bold"
          size={44}
          fontSize="18px"
          mr={[4]}
        >
          {rankInfo.rank !== 0 ? rankInfo.rank : "-"}
        </ColorTile>
      )
    }

    return <span />
  }

  renderPrivileged(account: Account) {
    return account.privileged ? (
      <Badge title="Privileged" bg="neutral">
        {t("account.badges.pv")}
      </Badge>
    ) : null
  }

  renderCo(account: Account) {
    return !this.isLastCodeUpdateEpoch(account.last_code_update) ? (
      <Badge title="Contract" bg={theme.colors.neutral}>
        {t("account.badges.co")}
      </Badge>
    ) : null
  }

  renderBp(account: Account) {
    return getRankInfo(account, voteStore.votes).rank > 0 ? (
      <Badge title="Block producer" bg={theme.colors.ternary}>
        {t("account.badges.bp")}
      </Badge>
    ) : null
  }

  renderMykey(account: Account) {
    return isAccountCreatedByMykey(account) ? (
      <ExternalTextLink to="https://mykey.org">
        <ClickableBadge title={t("account.badges.my.title")} bg={theme.colors.logo1}>
          <img src="/images/mykey.svg" alt="mykey logo" />
        </ClickableBadge>
      </ExternalTextLink>
    ) : null
  }

  isLastCodeUpdateEpoch(lastCodeUpdate: Date) {
    return lastCodeUpdate && new Date(lastCodeUpdate).getFullYear() === 1970
  }

  renderBadges(account: Account) {
    return (
      <BadgeContainer>
        {this.renderBp(account)}
        {this.renderCo(account)}
        {this.renderPrivileged(account)}
        {this.renderMykey(account)}
      </BadgeContainer>
    )
  }

  renderAccountImage(src: string) {
    return (
      <StyledObject data={src} type="image/jpg">
        <AccountImg alt={this.props.account.account_name} width="70px" height="70px" src={src} />
      </StyledObject>
    )
  }

  renderProducerAvatar(account: Account, src: string) {
    return (
      <Grid gridTemplateColumns={["auto 1fr", "auto 1fr"]} gridTemplateRows={["auto"]}>
        <Cell gridColumn={["1", "1"]} gridRow={["1", "1"]}>
          {this.renderRank(account)}
        </Cell>
        <Cell
          gridColumn={["2", "2"]}
          gridRow={["1", "1"]}
          alignSelf="start"
          height="72px"
          width="72px"
          border="1px solid"
          borderColor="grey3"
          bg="white"
          mr={[4]}
        >
          {this.renderAccountImage(src)}
        </Cell>
      </Grid>
    )
  }

  renderProducer(account: Account, blockProducerInfo: BlockProducerInfo) {
    return (
      <Cell py={[0]} gridColumn={["1", "1"]} gridRow={["1", "1"]}>
        <Grid gridTemplateColumns={["1fr", "auto auto 1fr"]} gridTemplateRows={["auto"]}>
          <Cell gridColumn={["1", "1"]} gridRow={["1", "1"]}>
            {this.renderProducerAvatar(account, blockProducerInfo.org.branding.logo_256)}
          </Cell>

          <Cell mt={[2, 0]} gridColumn={["1", "2"]} gridRow={["2", "1"]} mr={[3]}>
            <MonospaceText color="primary" fontWeight="bold" fontSize={[6]}>
              {account.account_name}
            </MonospaceText>
            <SubTitle fontWeight="bold" color="primary" fontSize={[2]}>
              {blockProducerInfo.org.candidate_name}
            </SubTitle>
          </Cell>
          <Cell gridColumn={["1", "3"]} gridRow={["3", "1"]} py="12px">
            {this.renderBadges(account)}
          </Cell>
        </Grid>
      </Cell>
    )
  }

  renderProducerNoJson(account: Account) {
    return (
      <Cell py={[0]} gridColumn={["1", "1"]} gridRow={["1", "1"]}>
        <Grid gridTemplateColumns={["1fr", "auto auto 1fr"]} gridTemplateRows={["auto"]}>
          <Cell gridColumn={["1", "1"]} gridRow={["1", "1"]}>
            {this.renderRank(account)}
          </Cell>

          <Cell mt={[2, 0]} gridColumn={["1", "2"]} gridRow={["2", "1"]} mr={[3]}>
            <MonospaceText color="primary" fontWeight="bold" fontSize={[6]}>
              {account.account_name}
            </MonospaceText>
          </Cell>
          <Cell gridColumn={["1", "3"]} gridRow={["3", "1"]} py="12px">
            {this.renderBadges(account)}
          </Cell>
        </Grid>
      </Cell>
    )
  }

  renderDefault(account: Account) {
    return (
      <Cell py={[0]} gridColumn={["1", "1"]} gridRow={["1", "1"]}>
        <Grid
          gridTemplateColumns={["1fr", "auto 1fr"]}
          gridTemplateRows={["auto"]}
          alignItems="center"
        >
          <Cell mt={[2, 0]} gridColumn={["1", "1"]} gridRow={["1", "1"]} mr={[3]}>
            <MonospaceText color="primary" fontWeight="bold" fontSize={[6]}>
              {account.account_name}
            </MonospaceText>
          </Cell>
          <Cell gridColumn={["1", "2"]} gridRow={["2", "1"]}>
            {this.renderBadges(account)}
          </Cell>
        </Grid>
      </Cell>
    )
  }

  render() {
    const { account } = this.props

    if (account.block_producer_info) {
      return this.renderProducer(account, account.block_producer_info)
    }

    if (getRankInfo(account, voteStore.votes).rank > 0) {
      return this.renderProducerNoJson(account)
    }

    if (!this.isLastCodeUpdateEpoch(account.last_code_update)) {
      return this.renderDefault(account)
    }

    return this.renderDefault(account)
  }
}

function isAccountCreatedByMykey(account: Account) {
  return (
    Config.network_id === "eos-mainnet" &&
    (account.account_name === "mykeymanager" ||
      (account.creator && account.creator.creator === "mykeymanager"))
  )
}
