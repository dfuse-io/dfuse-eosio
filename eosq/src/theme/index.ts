import emotionStyled, { CreateStyled } from "@emotion/styled"
import { colors } from "./colors"
import { breakpoints, fontSizes, lineHeights, space } from "./scales"

export const theme = {
  breakpoints,
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
