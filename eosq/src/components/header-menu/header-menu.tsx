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

  background: #1E1F23; /* Old browsers */
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
