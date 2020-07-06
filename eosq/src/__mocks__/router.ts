import { UnregisterCallback } from "history"
import { RouteComponentProps } from "react-router"

// tslint:disable-next-line no-empty
const noop = () => {}

export function getMockRouterProps<P = {}>(data: P = {} as P) {
  const location = {
    hash: "",
    key: "",
    pathname: "",
    search: "",
    state: {}
  }

  return {
    match: {
      isExact: true,
      params: data,
      path: "",
      url: ""
    },
    location,
    history: {
      length: 2,
      action: "POP",
      location,
      push: noop,
      replace: noop,
      go: noop,
      goBack: noop,
      goForward: noop,
      block: () => noop as UnregisterCallback,
      createHref: () => "",
      listen: () => noop as UnregisterCallback
    },
    staticContext: {}
  } as RouteComponentProps<P>
}
