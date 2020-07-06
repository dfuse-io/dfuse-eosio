import * as React from "react"
import { Box } from "@dfuse/explorer"
import { Cell } from "../ui-grid/ui-grid.component"
import { MonospaceText } from "../text-elements/misc"
import {
  PillWrapper,
  PillContainer,
  PillContainerDetails,
  PillExpandedContainer,
  PillHeaderText,
  PillInfoContainer,
  PillOverviewRow,
  PillClickable,
  HoverablePillContainer,
  AnimatedPillContainer,
  PillLogoContainer,
  PillLogo
} from "./pill-elements"
import { faMinus, faPlus } from "@fortawesome/free-solid-svg-icons"
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome"

interface Props {
  headerHoverTitle: string
  renderInfo: () => JSX.Element[] | JSX.Element | null
  disabled?: boolean
  colorVariant: string
  colorVariantHeader: string
  headerText: JSX.Element | string
  title?: JSX.Element | string
  info?: string
  content: JSX.Element[] | JSX.Element | string
  renderExpandedContent: () => JSX.Element[] | JSX.Element | null
  highlighted?: boolean
  logo?: PillLogoProps
}

export interface PillLogoProps {
  path: string
  website: string
}

interface State {
  isOpen: boolean
}

export class Pill extends React.Component<Props, State> {
  state: State = {
    isOpen: false
  }

  toggleIsOpen = () => {
    if (this.props.disabled) {
      return
    }

    this.setState((prevState: State) => ({ isOpen: !prevState.isOpen }))
  }

  renderTitle = () => {
    if (!this.props.title) {
      return (
        <Box px="2px" bg={this.props.colorVariant}>
          &nbsp;
        </Box>
      )
    }

    let WrapperComponent = PillClickable
    if (this.props.disabled) {
      WrapperComponent = Box
    }

    return (
      <WrapperComponent onClick={this.toggleIsOpen} bg={this.props.colorVariant}>
        <MonospaceText alignSelf="center" px={[2]} color="text" fontSize={[1]}>
          {this.props.title}
        </MonospaceText>
      </WrapperComponent>
    )
  }

  renderOverviewRow() {
    return (
      <PillOverviewRow bg={this.props.highlighted ? "lightyellow" : "#ffffff"} minHeight="26px">
        {this.renderTitle()}
        {this.props.content}
        {this.props.disabled ? null : (
          <PillClickable
            onClick={this.toggleIsOpen}
            bg="grey5"
            color="white"
            px="12px"
            alignItems={["center"]}
            display={["flex"]}
          >
            <FontAwesomeIcon size="sm" icon={this.state.isOpen ? faMinus : faPlus} />
          </PillClickable>
        )}
      </PillOverviewRow>
    )
  }

  renderInfoRow() {
    return (
      <PillInfoContainer>
        <Cell p={[3]}>{this.props.renderInfo()}</Cell>
      </PillInfoContainer>
    )
  }

  renderRawRow() {
    return (
      <PillExpandedContainer bg="#FFFFFF">
        {this.props.renderExpandedContent()}
      </PillExpandedContainer>
    )
  }

  renderHeader(text: JSX.Element | string, color: string, title: string) {
    let WrapperComponent = PillClickable
    if (this.props.disabled) {
      WrapperComponent = Box
    }

    return (
      <WrapperComponent
        onClick={this.toggleIsOpen}
        bg={color}
        alignItems="center"
        justifyConten="center"
      >
        <PillHeaderText
          title={title}
          pl={this.props.logo ? "35px" : "10px"}
          pr="7px"
          color="traceAccountText"
          fontSize={[1]}
        >
          {text}
        </PillHeaderText>
      </WrapperComponent>
    )
  }

  openWebsiteLink() {
    window.open(this.props.logo!.website, "_blank")
  }

  renderLogo() {
    if (this.props.logo) {
      return (
        <PillLogoContainer>
          <PillLogo onClick={() => this.openWebsiteLink()}>
            <img width="100%" src={this.props.logo.path} alt="" />
          </PillLogo>
        </PillLogoContainer>
      )
    }
    return null
  }

  render() {
    let PillContainerComponent = HoverablePillContainer
    if (this.props.disabled) {
      PillContainerComponent = PillContainer
    }

    return (
      <PillWrapper width="100%" display="block" clear="both" my={["5px"]}>
        {this.renderLogo()}
        <PillContainerComponent
          cursor={this.props.disabled ? "default" : "pointer"}
          overflow="hidden"
          gridTemplateColumns="auto 1fr"
        >
          {this.renderHeader(
            this.props.headerText,
            this.props.colorVariantHeader,
            this.props.headerHoverTitle
          )}
          {this.renderOverviewRow()}
        </PillContainerComponent>
        <AnimatedPillContainer pl="31px" pr="35px" maxHeight={this.state.isOpen ? "3000px" : "0px"}>
          <PillContainerDetails>
            {this.renderInfoRow()}
            {this.renderRawRow()}
          </PillContainerDetails>
        </AnimatedPillContainer>
      </PillWrapper>
    )
  }
}
