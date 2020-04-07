import * as React from "react"
import { Cell } from "../../atoms/ui-grid/ui-grid.component"
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome"
import { faSearch } from "@fortawesome/free-solid-svg-icons"
import { Links } from "../../routes"
import { theme, styled } from "../../theme"
import { Text } from "../../atoms/text/text.component"

interface Props {
  query: string
  position?: "left" | "right"
  color?: string
  fontSize?: number[] | string
  lineHeight?: number[] | string
  fontWeight?: string
  fixed?: boolean
}

const WrapperLeft: React.ComponentType<any> = styled(Text)`
  padding-left: 15px;

  display: inline-block;
  &.hide-on-hover .svg svg {
    display: none;
  }
  &.hide-on-hover:hover .svg svg {
    display: block;
  }
  &:hover .svg svg:hover {
    display: block;
    cursor: pointer;
  }
`

const WrapperRight: React.ComponentType<any> = styled(Text)`
  padding-right: 15px;

  display: inline-block;
  &.hide-on-hover .svg svg {
    display: none;
  }
  &.hide-on-hover:hover .svg svg {
    display: block;
  }
  &:hover .svg svg:hover {
    display: block;
    cursor: pointer;
  }
`

const MagnifierWrapper: React.ComponentType<any> = styled(Cell)`
  display: inline-block;
  position: absolute;
  font-size: 10px;
  top: calc(50% - 5px);
`

export class SearchShortcut extends React.Component<Props> {
  onClick = () => {
    window.location.href = `${Links.viewTransactionSearch()}?q=${this.props.query}`
  }

  renderLeft() {
    return (
      <WrapperLeft
        className={this.props.fixed ? null : "hide-on-hover"}
        fontWeight={this.props.fontWeight ? this.props.fontWeight : null}
        lineHeight={this.props.lineHeight ? this.props.lineHeight : null}
        fontSize={this.props.fontSize ? this.props.fontSize : null}
      >
        <Cell
          fontWeight={this.props.fontWeight ? this.props.fontWeight : null}
          fontSize={this.props.fontSize ? this.props.fontSize : null}
          display="inline-block"
          verticalAlign="middle"
        >
          {this.props.children}
        </Cell>
        <MagnifierWrapper
          left="0px"
          className="svg"
          color={theme.colors.grey6}
          onClick={this.onClick}
        >
          <FontAwesomeIcon icon={faSearch as any} />
        </MagnifierWrapper>
      </WrapperLeft>
    )
  }

  renderRight() {
    return (
      <WrapperRight
        className={this.props.fixed ? null : "hide-on-hover"}
        fontWeight={this.props.fontWeight ? this.props.fontWeight : null}
        lineHeight={this.props.lineHeight ? this.props.lineHeight : null}
        fontSize={this.props.fontSize ? this.props.fontSize : null}
      >
        <Cell
          fontWeight={this.props.fontWeight ? this.props.fontWeight : null}
          fontSize={this.props.fontSize ? this.props.fontSize : null}
          display="inline-block"
          verticalAlign="middle"
        >
          {this.props.children}
        </Cell>
        <MagnifierWrapper
          title={`Search - ${this.props.query}`}
          right="0px"
          className="svg"
          color={this.props.color ? this.props.color : theme.colors.grey6}
          onClick={this.onClick}
        >
          <FontAwesomeIcon icon={faSearch as any} />
        </MagnifierWrapper>
      </WrapperRight>
    )
  }

  render() {
    return this.props.position === "left" ? this.renderLeft() : this.renderRight()
  }
}
