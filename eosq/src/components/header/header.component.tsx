import * as React from "react"
import { withRouter } from "react-router"
import { Button } from "antd"
import { Cell, Grid } from "../../atoms/ui-grid/ui-grid.component"
import { SearchBar } from "../search-bar/search-bar"
import { UiDrawer } from "../../atoms/ui-drawer/ui-drawer"
import { Text } from "../../atoms/text/text.component"
import { HeaderMenu } from "../header-menu/header-menu"
import { HeaderLogo } from "../header-elements/header-elements"
import { theme, styled } from "../../theme"
import { faBars } from "@fortawesome/free-solid-svg-icons"
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome"
import { HeaderMenuMobile } from "../header-menu/header-menu-mobile"
import { menuStore } from "../../stores"
import { observer } from "mobx-react"
import { t } from "i18next"

const RoutedOmniSearch = withRouter(SearchBar)

interface State {
  height: number
}

const StyledButton: React.ComponentType<any> = styled(Button)`
  padding: 10px;
  border: 1px solid ${(props) => props.theme.colors.primary} !important;
  border-radius: 0 !important;
`

@observer
export class Header extends React.Component<any, State> {
  drawerOpened = false

  renderMenuOpener() {
    return (
      <StyledButton>
        <Text
          pl={[3]}
          pr={[3]}
          fontSize={[4]}
          display={["none", "inline-block"]}
          color={theme.colors.primary}
        >
          {t("core.menu.mainTitle")}
        </Text>
        <FontAwesomeIcon size="2x" icon={faBars as any} color={theme.colors.primary} />
      </StyledButton>
    )
  }

  renderMobileMenuOpener() {
    return (
      <Cell p="10px">
        <FontAwesomeIcon size="2x" icon={faBars as any} color={theme.colors.primary} />
      </Cell>
    )
  }

  renderMobileContent(onClose: () => void): JSX.Element {
    return <HeaderMenuMobile onClose={onClose} />
  }

  componentDidMount(): void {
    this.drawerOpened = menuStore.opened
  }

  componentDidUpdate(): void {
    // forcing drawer state, do not remove
    if (menuStore.opened !== this.drawerOpened) {
      this.drawerOpened = menuStore.opened
      this.forceUpdate()
    }
  }

  render() {
    return (
      <Grid
        gridTemplateColumns={["auto 1fr", "2fr 4fr 2fr", "2fr 5fr 2fr"]}
        pt={[0]}
        pb={[0]}
        px={[1, 0]}
        position="fixed"
      >
        <HeaderLogo variant="light" />
        <Cell
          gridRow={["2", "1", "1"]}
          gridColumn={["1 / span2", "2", "2"]}
          alignSelf="center"
          justifySelf={["inline", "inline"]}
          pt={[1, 4]}
          pb={[3, 4]}
          px={[0, 1]}
        >
          <RoutedOmniSearch />
        </Cell>
        <Cell
          gridRow={["1", "1", "1"]}
          gridColumn={["2 / span2", "3", "3"]}
          alignSelf={["start", "center"]}
          justifySelf={["end", "end"]}
          pr={[0, 3]}
        >
          <UiDrawer
            onOpen={() => menuStore.open()}
            onClose={() => menuStore.close()}
            opener={this.renderMenuOpener()}
            mobileOpener={this.renderMobileMenuOpener()}
            content={<HeaderMenu />}
            renderMobileContent={this.renderMobileContent}
            opened={menuStore.opened}
          />
        </Cell>
      </Grid>
    )
  }
}
