import { AlignItemsProps, compose, layout, LayoutProps, space, SpaceProps } from "styled-system"
import { styled } from "../theme"

export type ImgProps = SpaceProps | LayoutProps | AlignItemsProps

const imgStyle = compose(space, layout)

/**
 * A simple wrapper around `img` with those extra props available:
 * - spacing (margins & paddings)
 */
export const Img = styled.img<ImgProps>`
  ${imgStyle}
`
