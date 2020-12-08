import { createDfuseClient } from '@dfuse/client'
import { initializeDfuseClient } from '@dfuse/explorer'
import { configure } from "enzyme"
import Adapter from "enzyme-adapter-react-16"

import "jest-enzyme"
import "jest-localstorage-mock"

configure({ adapter: new Adapter() })
initializeDfuseClient(createDfuseClient({
  apiKey: "web_123456789",
  network: "localhost",
  requestIdGenerator: jest.fn(() => "dc-123"),
}))

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
