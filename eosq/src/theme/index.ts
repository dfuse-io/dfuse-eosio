import emotionStyled, { CreateStyled } from "@emotion/styled"
import { injectThemedStyled } from "@dfuse/explorer"
import { colors } from "./colors"
import { breakPoints, mediaQueries, fontSizes, lineHeights, space } from "./scales"

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

export type ThemeInterface = typeof theme

export const styled = emotionStyled as CreateStyled<ThemeInterface>

injectThemedStyled(styled)
