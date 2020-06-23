import * as React from "react"
import { Cell, Grid } from "../../atoms/ui-grid/ui-grid.component"
import { Text } from "../../atoms/text/text.component"
import { HeaderLogo } from "../header-elements/header-elements"
import { MainMenu } from "../main-menu/main-menu.component"
import { theme, styled } from "../../theme"
import { NetworkSelector } from "../settings-selectors/network-selector"
import { LanguageSelector } from "../settings-selectors/language-selector"
import { t } from "i18next"
import { getCurrentLanguageValue } from "../settings-selectors/settings.helpers"

const HeaderWrapper: React.ComponentType<any> = styled(Cell)`
  width: 100%;

  background: #474793; /* Old browsers */
  background: -moz-linear-gradient(left, #474793 8%, #5e5ec2 93%); /* FF3.6-15 */
  background: -webkit-linear-gradient(left, #474793 8%, #5e5ec2 93%); /* Chrome10-25,Safari5.1-6 */
  background: linear-gradient(
    to right,
    #474793 8%,
    #5e5ec2 93%
  ); /* W3C, IE10+, FF16+, Chrome26+, Opera12+, Safari7+ */
  filter: progid:DXImageTransform.Microsoft.gradient( startColorstr='#474793', endColorstr='#5e5ec2',GradientType=1 ); /* IE6-9 */
`

export class HeaderMenu extends React.Component {
  renderSectionTitle(text: string) {
    return (
      <Text pb={[2]} pl={[3]} fontWeight="600" fontSize={[3]} color={theme.colors.bleu6}>
        {text}
      </Text>
    )
  }

  render() {
    return (
      <HeaderWrapper mx="auto">
        <Cell mx="auto" px={[2, 3, 4]} py={[0, 3]}>
          <Grid
            gridTemplateColumns={["1fr", "1fr 1fr 1fr 1fr", "1fr 1fr 1fr 1fr"]}
            pt={[0]}
            pb={[0]}
            px={[1, 0]}
            gridColumnGap={[0, 1, 2]}
          >
            <Cell height="100%" py={[2]}>
              <HeaderLogo variant="light" />
              <Cell pt={[4]}>
                <a
                  href={`https://dfuse.io/${getCurrentLanguageValue()}`}
                  title="The dfuse Blockchain Data Platform"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  <img
                    src={`/images/built-with-dfuse${
                      getCurrentLanguageValue() === "zh" ? "-CN" : ""
                    }-01.png`}
                    title="The dfuse Blockchain Data Platform"
                    alt="built-with-dfuse"
                    width="210"
                    height="auto"
                  />
                </a>
              </Cell>
            </Cell>

            <Cell
              height="100%"
              borderLeft={`2px solid ${theme.colors.bleu6}`}
              alignSelf="right"
              justifySelf={["inline", "inline"]}
              px={[0, 4]}
              py={[2]}
            >
              {this.renderSectionTitle(t("core.menu.titles.navigation"))}
              <MainMenu variant="light" />
            </Cell>
            <Cell
              height="100%"
              borderLeft={`2px solid ${theme.colors.bleu6}`}
              alignSelf="right"
              justifySelf={["inline", "inline"]}
              px={[0, 4]}
              py={[2]}
            >
              {this.renderSectionTitle(t("core.menu.titles.network"))}
              <NetworkSelector variant="light" />
            </Cell>
            <Cell
              height="100%"
              borderLeft={`2px solid ${theme.colors.bleu6}`}
              alignSelf="right"
              justifySelf={["inline", "inline"]}
              px={[0, 4]}
              py={[2]}
            >
              {this.renderSectionTitle(t("core.menu.titles.language"))}
              <LanguageSelector variant="light" />
            </Cell>
          </Grid>
        </Cell>
      </HeaderWrapper>
    )
  }
}
