import emotionStyled, { CreateStyled } from "@emotion/styled"
import { injectThemedStyled } from "@dfuse/explorer"
import { colors } from "./colors"
import { breakPoints, mediaQueries, fontSizes, lineHeights, space } from "./scales"
import { Scale, get as resolveValueWithDefault } from "styled-system"

export const theme = {
  breakPoints,
  mediaQueries,
  fontSizes,
  lineHeights,
  space,
  colors,
  fontFamily: {
    roboto: "Roboto Condensed",
    opensans: "Open Sans",
    iceland: "Iceland",
    lato: "Lato"
  }
}

export function resolveValue(n: number | string, scale?: Scale) {
  return resolveValueWithDefault(scale, n, n)
}

export type ThemeInterface = typeof theme

export const styled = emotionStyled as CreateStyled<ThemeInterface>

injectThemedStyled(styled)
