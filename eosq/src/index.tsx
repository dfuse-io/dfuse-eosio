import { createBrowserHistory } from "history"
import { Provider } from "mobx-react"
import { RouterStore, syncHistoryWithStore } from "mobx-react-router"
import { render } from "react-dom"
import { Router } from "react-router"
import { initializeDfuseClientFromConfig } from "./clients/dfuse"
import "sanitize.css/sanitize.css"
import { App } from "./App"
import "./i18n"
import "./index.css"

const browserHistory = createBrowserHistory()
const routingStore = new RouterStore()

const history = syncHistoryWithStore(browserHistory, routingStore)

const stores = {
  routing: routingStore,
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

initializeDfuseClientFromConfig()

renderApp(App)
