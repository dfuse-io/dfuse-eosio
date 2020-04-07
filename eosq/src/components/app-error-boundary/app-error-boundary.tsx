import * as React from "react"
import Sentry from "@sentry/browser"
import { serviceWorkerStore } from "../../stores"
import { ServiceWorkerStates } from "../../stores/service-worker-store"
import { styled } from "../../theme"
import { log } from "../../services/logger"

interface State {
  error: any
  installing: boolean
}

const Spinner = styled.div`
  margin: 40px auto;
  width: 250px;
  height: 40px;
  text-align: center;
  font-size: 10px;

  > div {
    width: 20px;
    margin: 0 3px 0 0;
    background-color: #333;
    height: 100%;
    display: inline-block;
    background-color: #6d6ae8;
    -webkit-animation: sk-stretchdelay 1.5s infinite ease-in-out;
    animation: sk-stretchdelay 1.5s infinite ease-in-out;
  }

  .rect2 {
    -webkit-animation-delay: -1.2s;
    animation-delay: -1.2s;
  }

  .rect3 {
    -webkit-animation-delay: -0.9s;
    animation-delay: -0.9s;
  }

  .rect4 {
    -webkit-animation-delay: -0.6s;
    animation-delay: -0.6s;
  }

  .rect5 {
    -webkit-animation-delay: -0.3s;
    animation-delay: -0.3s;
  }

  @-webkit-keyframes sk-stretchdelay {
    0%,
    40%,
    100% {
      -webkit-transform: scaleY(0.4);
    }
    20% {
      -webkit-transform: scaleY(1);
    }
  }

  @keyframes sk-stretchdelay {
    0%,
    40%,
    100% {
      transform: scaleY(0.4);
      -webkit-transform: scaleY(0.4);
    }
    20% {
      transform: scaleY(1);
      -webkit-transform: scaleY(1);
    }
  }
`

export class AppErrorBoundary extends React.Component<any, State> {
  state = { error: null, installing: false }

  handleError(error: any) {
    let interval: number | undefined = setInterval(() => {
      if (serviceWorkerStore.state === ServiceWorkerStates.INSTALLED) {
        clearInterval(interval!)
        interval = undefined
        window.location.reload()
        return
      }

      if (serviceWorkerStore.state !== ServiceWorkerStates.INSTALLING) {
        this.setState({ error, installing: false })
      } else {
        this.setState({ installing: true, error: null })
      }
    }, 250) as any

    setTimeout(() => {
      clearInterval(interval!)
      interval = undefined
    }, 15000)
  }

  componentDidCatch(error: any, errorInfo: any) {
    this.handleError(error)
    if (Sentry) {
      Sentry.withScope((scope) => {
        Object.keys(errorInfo).forEach((key) => {
          scope.setExtra(key, errorInfo[key])
        })

        log.error("Captured an error", error)
        Sentry.captureException(error)
      })
    }
  }

  onReportErrorClicked = () => {
    if (Sentry) {
      Sentry.showReportDialog()
    }
  }

  render() {
    if (this.state.installing) {
      return (
        <div
          style={{
            width: "100%",
            height: "100vh",
            display: "flex",
            alignItems: "center",
            padding: 30,
            textAlign: "center",
            marginTop: 40
          }}
        >
          <div style={{ fontSize: 34, color: "#6d6ae8", width: "100%" }}>
            Welcome to&nbsp;
            <b style={{ fontSize: 34, color: "#6d6ae8" }}>eosq</b>
            <br />
            <br />
            <span style={{ fontSize: 20, color: "#6d6ae8" }}>
              Please wait, a new version of the application is being loaded.
            </span>
            <Spinner>
              <div className="rect1" />
              <div className="rect2" />
              <div className="rect3" />
              <div className="rect4" />
              <div className="rect5" />
            </Spinner>
          </div>
        </div>
      )
    }

    if (this.state.error) {
      // Render fallback UI
      return (
        <div style={{ padding: 30 }}>
          <p>
            We&apos;re sorry
            <span role="img" aria-label="emoji">
              😳
            </span>
            — something&apos;s gone wrong.
          </p>
          <p>
            <a onClick={this.onReportErrorClicked}>Report feedback</a>. Once reported, try going
            back to the home page and reload the page.
          </p>
        </div>
      )
    }

    // When there's not an error, render the children untouched
    return this.props.children
  }
}
