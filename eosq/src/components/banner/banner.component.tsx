import { t } from "i18next"
import { observer } from "mobx-react"
import * as React from "react"
import { formatAmount, formatNumber } from "../../helpers/formatters"
import { styled } from "../../theme"
import { metricsStore } from "../../stores"
import { Text } from "../../atoms/text/text.component"
import { Box } from "@dfuse/explorer"
import { Cell, Grid } from "../../atoms/ui-grid/ui-grid.component"
import { BannerContainer, BannerItem, BannerTitle } from "./banner-item.component"
import { AmountVariation } from "./variation.component"
import { Link } from "react-router-dom"
import { Links } from "../../routes"
import { Config } from "../../models/config"

const BannerWrapper: React.ComponentType<any> = styled(Grid)`
  grid-column-gap: 0px;
  border-style: solid;
  border-color: ${(props) => props.theme.colors.bleu6};
`

const Price: React.ComponentType<any> = styled(Text)`
  color: ${(props) => props.theme.colors.bannerValue};
  line-height: 1;
  font-weight: 700;
  font-family: "Roboto Condensed", sans-serif;
`

const BannerMarketPrice: React.SFC<{ price: number; variation: number }> = ({
  price,
  variation
}) => {
  let formattedPrice = formatAmount(price)
  if (price < 0) {
    formattedPrice = ""
  }

  return (
    <BannerContainer>
      <Box flexDirection="row">
        <Box align={["center", "center", "left"]}>
          <Price fontSize={[8]}>{formattedPrice}</Price>
          <Box
            align="left"
            ml={[2]}
            flexDirection="column"
            justifyContent="center"
            alignItems="center"
          >
            <Cell>
              <BannerTitle fontSize={[1]}>{t("banner.eos_usd")}</BannerTitle>
              <AmountVariation variation={variation} textColor="bannerValue" />
            </Cell>
          </Box>
        </Box>
      </Box>
    </BannerContainer>
  )
}

@observer
export class Banner extends React.Component {
  renderProducerLink(account: string) {
    if (!account || account.length === 0) {
      return (
        <Text
          fontFamily="'Roboto Condensed', sans-serif;"
          fontWeight="bold"
          color="white"
          fontSize={[4, 5, 5]}
        >
          {account}
        </Text>
      )
    }
    return (
      <Link to={Links.viewAccount({ id: account })}>
        <Text
          fontFamily="'Roboto Condensed', sans-serif;"
          fontWeight="bold"
          color="white"
          fontSize={[4, 5, 5]}
        >
          {account}
        </Text>
      </Link>
    )
  }

  renderBlockLink(blockNum: string, blockId: string) {
    if (!blockId || blockId.length === 0) {
      return (
        <Text
          fontFamily="'Roboto Condensed', sans-serif;"
          fontWeight="bold"
          color="white"
          fontSize={[4, 5, 5]}
        >
          {blockNum}
        </Text>
      )
    }
    return (
      <Link to={Links.viewBlock({ id: blockId })}>
        <Text
          fontFamily="'Roboto Condensed', sans-serif;"
          fontWeight="bold"
          color="white"
          fontSize={[4, 5, 5]}
        >
          {blockNum}
        </Text>
      </Link>
    )
  }

  renderBannerPrice() {
    if (!Config.display_price) {
      return <BannerContainer />
    }
    return (
      <BannerMarketPrice price={metricsStore.priceUSD} variation={metricsStore.priceVariation} />
    )
  }

  render() {
    return (
      <Cell>
        <BannerWrapper
          borderLeft={["0px", "1px"]}
          borderRight={["0px"]}
          borderBottom={["0px"]}
          borderTop={["0px"]}
          py="0px"
          gridTemplateColumns={["3fr 2fr", "3fr 2fr 2fr 2fr"]}
        >
          {this.renderBannerPrice()}
          <BannerItem
            title={t("banner.head_block")}
            details={formatNumber(metricsStore.headBlockNum)}
          />
          <BannerItem
            title={t("banner.irreversible_block")}
            titleTip={t("banner.irreversible_block_tooltip")}
            details={this.renderBlockLink(
              formatNumber(metricsStore.lastIrreversibleBlockNum),
              metricsStore.lastIrreversibleBlockId
            )}
          />
          <BannerItem
            title={t("banner.head_block_producer")}
            titleTip={t("banner.head_block_producer_tooltip")}
            details={this.renderProducerLink(metricsStore.headBlockProducer)}
          />
        </BannerWrapper>
      </Cell>
    )
  }
}
