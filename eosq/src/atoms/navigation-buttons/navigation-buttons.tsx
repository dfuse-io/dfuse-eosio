import * as React from "react"
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome"
import { faAngleLeft, faAngleDoubleLeft, faAngleRight } from "@fortawesome/free-solid-svg-icons"

import { Cell } from "../ui-grid/ui-grid.component"
import { borders, color as color_ } from "styled-system"
import { theme, styled } from "../../theme"

interface NavigationProps {
  direction: "next" | "previous" | "first"
  onClick?: () => void
  variant?: string
}

const ColoredTile: React.ComponentType<any> = styled(Cell)`
  ${borders}
  ${color_}

  width: 30px;
  height: 30px;
  text-align: center;
  cursor: pointer;
  display: flex;
  justify-items: center;
  align-items: center;
  justify-content: center;
  &:hover {
    background-color: ${(props) => props.hoverBg};
    color: ${(props) => props.hoverColor};
  }
`

export const NavigationButton: React.SFC<NavigationProps> = ({ direction, onClick, variant }) => {
  variant = variant || "dark"
  const color = variant === "dark" ? "white" : "ternary"
  const hoverColor = variant === "dark" ? color : "white"
  const hoverBg = variant === "dark" ? theme.colors.bleu10 : theme.colors.ternary
  const ChevronLeft = <FontAwesomeIcon size="lg" color={color} icon={faAngleLeft} />
  const ChevronRight = <FontAwesomeIcon size="lg" color={color} icon={faAngleRight} />
  const DoubleChevronLeft = <FontAwesomeIcon size="lg" color={color} icon={faAngleDoubleLeft} />

  let Chevron = ChevronLeft

  if (direction === "next") {
    Chevron = ChevronRight
  } else if (direction === "first") {
    Chevron = DoubleChevronLeft
  }

  return (
    <ColoredTile
      hoverColor={hoverColor}
      color={color}
      hoverBg={hoverBg}
      bg={variant === "light" ? "white" : theme.colors.bleu9}
      border={variant === "light" ? `1px solid ${theme.colors.grey2}` : "none"}
      title={direction}
      size="30"
      onClick={onClick}
    >
      {Chevron}
    </ColoredTile>
  )
}

interface NavigationButtonsProps {
  onNext: () => void
  onPrev: () => void
  onFirst?: () => void
  showFirst: boolean
  showNext: boolean
  showPrev: boolean
  variant?: string
}

export class NavigationButtons extends React.Component<NavigationButtonsProps> {
  renderNavigationButton(direction: "next" | "previous" | "first", display: boolean) {
    let onClick: () => void
    if (direction === "next") {
      onClick = this.props.onNext
    } else if (direction === "previous") {
      onClick = this.props.onPrev
    } else if (direction === "first" && this.props.onFirst) {
      onClick = this.props.onFirst
    }

    if (display) {
      return (
        <NavigationButton
          variant={this.props.variant}
          direction={direction}
          onClick={() => onClick()}
        />
      )
    }

    return null
  }
  render() {
    return (
      <Cell>
        {this.props.onFirst ? (
          <Cell display="inline-block" p={1}>
            {this.renderNavigationButton("first", this.props.showFirst)}
          </Cell>
        ) : null}
        <Cell display="inline-block" p={1}>
          {this.renderNavigationButton("previous", this.props.showPrev)}
        </Cell>
        <Cell display="inline-block" p={1}>
          {this.renderNavigationButton("next", this.props.showNext)}
        </Cell>
      </Cell>
    )
  }
}
