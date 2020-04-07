import * as React from "react"
import { shallowWithTheme } from "../../../tests/renderers"
import { NavigationButtons } from "../navigation-buttons"

describe("NavigationButtons", () => {
  describe("default render", () => {
    it("should render properly", () => {
      expect(shallowWithTheme(render())).toMatchSnapshot()
    })
  })
})

function render() {
  const props = {
    onFirst: () => {
      console.log("first")
    },
    onNext: () => {
      console.log("next")
    },
    onPrev: () => {
      console.log("previous")
    },
    showNext: true,
    showPrev: true,
    showFirst: true
  }

  return <NavigationButtons {...props} />
}
