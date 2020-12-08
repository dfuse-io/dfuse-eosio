import {
  boxShadow,
  BoxShadowProps,
  color,
  ColorProps,
  compose,
  layout,
  LayoutProps,
  space,
  SpaceProps,
} from "styled-system"
import { styled } from "../theme"

export type BlockProps = BoxShadowProps | ColorProps | SpaceProps | LayoutProps

const blockStyle = compose(boxShadow, color, space, layout)

/**
 * A simple wrapper around `div` with those extra props available:
 * - spacing (margins & paddings)
 */
export const Block = styled.div<BlockProps>`
  ${blockStyle}
`
