import { FontAwesomeIcon } from "@fortawesome/react-fontawesome"
import { Text, TextLink, TextLinkProps } from "../text/text.component"
import { styled } from "../../theme"
import * as React from "react"

export const MonospaceTextLink: React.SFC<TextLinkProps> = ({ ...rest }) => (
  <TextLink {...rest} fontFamily="'Roboto Mono', monospace;" lineHeight="1em" />
)

export const ArrowTo: React.ComponentType<any> = styled(FontAwesomeIcon)`
  height: auto;
  margin: 1px 4px 0 4px;
  vertical-align: middle;
  color: ${(props) => props.theme.colors.text};
`

export const MonospaceText: React.ComponentType<any> = styled(Text)`
  font-family: "Roboto Mono", monospace;
  white-space: nowrap;
`

export const MonospaceTextWrap: React.ComponentType<any> = styled(Text)`
  font-family: "Roboto Mono", monospace;
`

export const WrappingText: React.ComponentType<any> = styled(Text)`
  overflow: hidden;
  word-wrap: break-word;
`

export const WrappingMonospaceText: React.ComponentType<any> = styled(Text)`
  font-family: "Roboto Mono", monospace;
  overflow: hidden;
  word-wrap: break-word;
`
