import * as React from "react"
import { UiDropDown } from "../ui-dropdown.component"
import { shallowWithTheme } from "../../../tests/renderers"

describe("UiDropdown", () => {
  describe("default render", () => {
    it("should render properly", () => {
      function onSelect(e: any) {
        console.log(e)
      }

      expect(shallowWithTheme(render(onSelect))).toMatchSnapshot()
    })
  })
})

function render(onSelect: (e: any) => void) {
  return (
    <UiDropDown
      id={"id"}
      onSelect={onSelect}
      options={[
        { label: "label 1", value: "value 1" },
        { label: "label 2", value: "value 2" }
      ]}
    />
  )
}
