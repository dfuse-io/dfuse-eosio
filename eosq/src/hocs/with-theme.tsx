import * as React from "react"
import { ThemeProvider as EmotionThemeProvider, ThemeProviderProps } from "emotion-theming"
import { theme, ThemeInterface } from "../theme"

const ThemeProvider: React.FC<ThemeProviderProps<ThemeInterface>> = EmotionThemeProvider as any

export default (ComposedComponent: any) => {
  return class WrapperComponent extends React.PureComponent {
    state = {
      currentTheme: localStorage.getItem("@theme:current")
    }

    switchTheme = () => {
      const currentTheme = localStorage.getItem("@theme:current")
      if (currentTheme === "darkTheme") {
        this.setState({ currentTheme: "lightTheme" })
        localStorage.setItem("@theme:current", "lightTheme")
      } else {
        this.setState({ currentTheme: "darkTheme" })
        localStorage.setItem("@theme:current", "darkTheme")
      }
    }

    render() {
      return (
        <ThemeProvider theme={theme}>
          <ComposedComponent
            switchTheme={this.switchTheme}
            currentTheme={this.state.currentTheme}
            {...this.props}
          />
        </ThemeProvider>
      )
    }
  }
}
