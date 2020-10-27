import {
  alignItems,
  alignSelf,
  borderColor,
  borderRadius,
  borders,
  color,
  display,
  flex,
  flexDirection,
  flexWrap,
  height,
  justifyContent,
  justifySelf,
  space,
  textAlign,
  width,
  fontSize,
  minWidth,
  maxWidth,
  minHeight,
  compose
} from "styled-system"
import { styled } from "../../theme"
import * as React from "react"

const boxStyle = compose(
  fontSize,
  display,
  width,
  minWidth,
  maxWidth,
  minHeight,
  space,
  flexDirection,
  alignSelf,
  justifySelf,
  justifyContent,
  alignItems,
  flex,
  color,
  flexWrap,
  textAlign,
  height,
  borders,
  borderRadius,
  borderColor
)

const Box: React.ComponentType<any> = styled.div`
  white-space: ${(props) => (props.whiteSpace ? props.whiteSpace : "")};
  word-break: ${(props) => (props.wordBreak ? props.wordBreak : "")};
  display: flex;
  position: relative;
  ${boxStyle}

  b {
    ${fontSize}
  }
`

export default Box

export const HomeWrapper: React.ComponentType<any> = styled(Box)`
  min-height: 100vh;
  background-color: ${(props) => props.theme.colors.primary};
`
