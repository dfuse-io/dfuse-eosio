import * as React from "react"
import { Cell } from "../../atoms/ui-grid/ui-grid.component"
import { HeaderLogo } from "../header-elements/header-elements"
import { MainMenu } from "../main-menu/main-menu.component"
import { theme, styled } from "../../theme"
import { Box } from "@dfuse/explorer"
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome"
import { Text } from "../../atoms/text/text.component"
import { t } from "i18next"
import {
  getCurrentLanguageName,
  getCurrentLanguageValue,
} from "../settings-selectors/settings.helpers"
import {
  faGlobeAmericas,
  faChevronLeft,
  faTimes,
  faVectorSquare,
} from "@fortawesome/free-solid-svg-icons"
import { NetworkSelector } from "../settings-selectors/network-selector"
import { LanguageSelector } from "../settings-selectors/language-selector"
import { Config, EosqNetwork } from "../../models/config"

const HeaderWrapper: React.ComponentType<any> = styled(Cell)`
  width: 100%;
  background-color: white;
  height: 100%;
  &:focus {
    outline: none !important;
  }

  div:focus {
    outline: none !important;
  }
`

const SelectorWrapper: React.ComponentType<any> = styled(Cell)`
  padding: 8px;
  background-color: ${(props) => props.theme.colors.grey3};
  height: 100vh;
  display: flex;
  flex-direction: column;
`

const ChevronContainer: React.ComponentType<any> = styled.div`
  position: absolute;
  top: 12px;
  left: 10px;
  z-index: 1000000;
`

interface Props {
  onClose: () => void
}

interface State {
  displayedSection: "main" | "network" | "language"
}

const StyledIcon = styled(FontAwesomeIcon)`
  width: auto !important;
  height: 24px !important;
`

export class HeaderMenuMobile extends React.Component<Props, State> {
  state = { displayedSection: "main" } as State

  showSection = (section: "main" | "network" | "language") => {
    this.setState({ displayedSection: section })
  }

  renderNetworkTitle(color: string, pl: number, width?: string) {
    return [
      <Cell width="24px" textAlign="center" key="0">
        <StyledIcon color={color} icon={faVectorSquare as any} size="lg" />
      </Cell>,
      <Text key="1" color={color} pl={[pl]} fontWeight="600" fontSize={[3]} width={width}>
        {t("core.menu.titles.network")}
      </Text>,
    ]
  }

  renderLanguageTitle(color: string, pl: number, width?: string) {
    return [
      <Cell width="24px" textAlign="center" key="0">
        <StyledIcon color={color} icon={faGlobeAmericas as any} size="lg" />
      </Cell>,
      <Text key="1" color={color} pl={[pl]} fontWeight="600" fontSize={[3]} width={width}>
        {t("core.menu.titles.language")}
      </Text>,
    ]
  }

  renderNetworkSummary() {
    const network = Config.available_networks.find((ref: EosqNetwork) => {
      return ref.id === Config.network_id
    })

    return (
      <Box py={[2]} px={[3]} alignItems="center" onClick={() => this.showSection("network")}>
        {this.renderNetworkTitle(theme.colors.bleu8, 3, "calc(50% - 12px)")}
        <Text
          width="calc(50% - 12px)"
          color={theme.colors.grey5}
          textAlign="right"
          fontWeight="600"
          fontSize={[2]}
        >
          {t(`core.networkOptions.${Config.network_id.replace("-", "_")}`, {
            defaultValue: network ? network.name : Config.network_id,
          })}
        </Text>
      </Box>
    )
  }

  renderLanguageSummary() {
    const language = getCurrentLanguageName()

    return (
      <Box py={[2]} px={[3]} alignItems="center" onClick={() => this.showSection("language")}>
        {this.renderLanguageTitle(theme.colors.bleu8, 3, "calc(50% - 12px)")}
        <Text
          width="calc(50% - 12px)"
          color={theme.colors.grey5}
          textAlign="right"
          fontWeight="600"
          fontSize={[2]}
        >
          {language}
        </Text>
      </Box>
    )
  }

  renderNetworkContent() {
    return (
      <SelectorWrapper>
        <Cell position="relative" p={[3]} textAlign="center">
          <ChevronContainer onClick={() => this.showSection("main")}>
            <FontAwesomeIcon color={theme.colors.bleu8} size="2x" icon={faChevronLeft as any} />
          </ChevronContainer>
          <Box justifyContent="center" alignItems="center" width="100%">
            {this.renderNetworkTitle(theme.colors.grey6, 1)}
          </Box>
        </Cell>
        <Box height="100%" bg="white">
          <NetworkSelector variant="dark" />
        </Box>
      </SelectorWrapper>
    )
  }

  renderLanguageContent() {
    return (
      <SelectorWrapper>
        <Cell position="relative" p={[3]} textAlign="center">
          <ChevronContainer onClick={() => this.showSection("main")}>
            <FontAwesomeIcon color={theme.colors.bleu8} size="2x" icon={faChevronLeft as any} />
          </ChevronContainer>
          <Box justifyContent="center" alignItems="center" width="100%">
            {this.renderLanguageTitle(theme.colors.grey6, 1)}
          </Box>
        </Cell>
        <Box height="100%" bg="white">
          <LanguageSelector variant="dark" />
        </Box>
      </SelectorWrapper>
    )
  }

  onClose = () => {
    this.props.onClose()
  }

  renderMain() {
    return (
      <HeaderWrapper mx="auto">
        <Cell mx="auto" py={[0, 3]} height="100%">
          <Cell>
            <Box
              alignItems="center"
              height="100%"
              py={[2]}
              borderBottom={`2px dotted ${theme.colors.grey3}`}
            >
              <Cell display="inline-block" pl={[3]}>
                <HeaderLogo variant="dark" />
              </Cell>
              <Cell
                pr={[3]}
                width="100%"
                display="inline-block"
                onClick={this.onClose}
                textAlign="right"
              >
                <FontAwesomeIcon size="2x" icon={faTimes as any} color={theme.colors.bleu10} />
              </Cell>
            </Box>

            <Cell
              height="100%"
              borderBottom={`1px solid ${theme.colors.grey3}`}
              alignSelf="right"
              justifySelf={["inline", "inline"]}
              px={[0, 4]}
              py={[2]}
            >
              <MainMenu variant="dark" />
            </Cell>
            <Cell
              height="100%"
              borderBottom={`1px solid ${theme.colors.grey3}`}
              alignSelf="right"
              justifySelf={["inline", "inline"]}
              px={[0, 4]}
              py={[2]}
            >
              {this.renderNetworkSummary()}
            </Cell>
            <Cell
              height="100%"
              borderBottom={`1px solid ${theme.colors.grey3}`}
              alignSelf="right"
              justifySelf={["inline", "inline"]}
              px={[0, 4]}
              py={[2]}
            >
              {this.renderLanguageSummary()}
            </Cell>
          </Cell>
          <Cell
            pt={[4]}
            style={{ position: "absolute", bottom: "20px", textAlign: "center", width: "100%" }}
          >
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
      </HeaderWrapper>
    )
  }

  render() {
    switch (this.state.displayedSection) {
      case "main":
        return this.renderMain()
      case "network":
        return this.renderNetworkContent()
      case "language":
        return this.renderLanguageContent()
      default:
        return this.renderMain()
    }
  }
}
