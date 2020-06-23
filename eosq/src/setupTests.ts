import { configure } from "enzyme"
import Adapter from "enzyme-adapter-react-16"
import { createSerializer as createEmotionSerializer } from "jest-emotion"
import emotion from "@emotion/core"

import "jest-enzyme"
import "jest-localstorage-mock"

configure({ adapter: new Adapter() })

expect.addSnapshotSerializer(createEmotionSerializer(emotion as any))

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
