import * as React from "react"
import * as Sentry from "@sentry/browser"
import { withRouter } from "react-router"
import AppContainer from "./components/app-container/app-container.component"
import { AppErrorBoundary } from "./components/app-error-boundary/app-error-boundary"
import withTheme from "./hocs/with-theme"
import { Config } from "./models/config"

// Let's initialize Sentry error handling, if not disabled
if (!Config.disable_sentry) {
  Sentry.init({
    dsn: "https://e268f409256b4df6b11d2fa584e734af@sentry.io/1339887"
  })
}

// @ts-ignore
const Container = withRouter(withTheme(AppContainer))

const App = (props: any) => (
  <AppErrorBoundary>
    <Container />
  </AppErrorBoundary>
)

export default App
