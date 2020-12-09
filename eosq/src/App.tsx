import * as React from "react"
import * as Sentry from "@sentry/browser"
import { withRouter } from "react-router"
import AppContainer from "./components/app-container/app-container.component"
import { AppErrorBoundary } from "./components/app-error-boundary/app-error-boundary"
import withTheme from "./hocs/with-theme"
import { Config } from "./models/config"
import { Helmet } from "react-helmet"

// Let's initialize Sentry error handling, if not disabled
if (!Config.disable_sentry && !Config.isLocalhost) {
  console.log("Initializing Sentry!")
  Sentry.init({
    dsn: "https://e268f409256b4df6b11d2fa584e734af@sentry.io/1339887",
  })
}

// @ts-ignore
const Container = withRouter(withTheme(AppContainer))

export const App = (props: any) => (
  <AppErrorBoundary>
    <DocumentMeta />
    <Container />
  </AppErrorBoundary>
)

const DocumentMeta: React.FC = () => {
  const { network } = Config
  if (network == null) {
    return null
  }

  const baseURL = new URL("", document.baseURI).href.replace(/\/+$/, "")
  const faviconURL = `${baseURL}${network.favicon_template}.png`

  return (
    <Helmet>
      {network?.page_title ? <title>{network?.page_title}</title> : null}

      {/* The meta tags must always use full-url */}
      {network?.favicon_template ? <meta property="og:image" content={faviconURL} /> : null}
      {network?.favicon_template ? <meta name="twitter:image" content={faviconURL} /> : null}
      {network?.favicon_template ? (
        <link rel="shortcut icon" href={`${network.favicon_template}.png`} />
      ) : null}
    </Helmet>
  )
}
