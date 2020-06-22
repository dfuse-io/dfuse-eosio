import { createBrowserHistory } from "history"
import { Provider } from "mobx-react"
import { RouterStore, syncHistoryWithStore } from "mobx-react-router"
import * as React from "react"
import { render } from "react-dom"
import { Router } from "react-router"
import "sanitize.css/sanitize.css"
import App from "./App"
import "./i18n"
import "./index.css"
// eslint-disable-next-line import/no-extraneous-dependencies
import { initializeDfuseClientFromConfig } from "@dfuse/explore"
import { Config } from "./models/config"

const browserHistory = createBrowserHistory()
const routingStore = new RouterStore()

const history = syncHistoryWithStore(browserHistory, routingStore)

const stores = {
  routing: routingStore
}

const renderApp = (NextApp: any) =>
  render(
    <Provider {...stores}>
      <Router history={history}>
        <NextApp />
      </Router>
    </Provider>,
    document.querySelector("#root")
  )

// @ts-ignore
const hotModule = module.hot

/* Hot module reload enabled (if available through `module.hot`) */
if (hotModule) {
  hotModule.accept("./App", () => {
    // eslint-disable-next-line global-require
    const NextApp = require("./App").default
    renderApp(NextApp)
  })
}

initializeDfuseClientFromConfig({
  apiKey: Config.dfuse_io_api_key,
  network: Config.dfuse_io_endpoint,
  authUrl: Config.dfuse_auth_endpoint,
  secure: Config.secure !== undefined && Config.secure
})

renderApp(App)
