import { configure } from "enzyme"
import Adapter from "enzyme-adapter-react-16"

import "jest-enzyme"
import "jest-localstorage-mock"

configure({ adapter: new Adapter() })

// Initialize correct i18n resources
withConsoleDisabled(() => {
  // eslint-disable-next-line global-require
  require("./i18n")
})

function withConsoleDisabled(worker: () => void) {
  const consoleLog = console.log

  console.log = () => {}

  worker()

  console.log = consoleLog
}
