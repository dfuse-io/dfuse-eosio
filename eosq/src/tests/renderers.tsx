// eslint-disable-next-line import/no-extraneous-dependencies
import { shallow } from "enzyme"
import * as React from "react"
import { ThemeProvider } from "emotion-theming"
import { theme as defaultTheme } from "../theme"

export const shallowWithTheme = (tree: any, theme: any = undefined) => {
  const context = shallow(<ThemeProvider theme={theme || defaultTheme} />).instance()

  return shallow(tree, { context })
}
