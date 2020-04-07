import { shallow } from "enzyme"
import * as React from "react"
import { FormattedContractElement } from "../formatted-contract-element"

describe("formatted-contract-element", () => {
  it("correctly renders date when type is time_point", () => {
    expect(
      shallow(
        <FormattedContractElement type="time_point" value="2018-07-30T12:52:54.000" label="" />
      )
    ).toMatchSnapshot()
  })

  it("correctly renders date when type is time_point_sec", () => {
    expect(
      shallow(
        <FormattedContractElement type="time_point_sec" value="2018-07-30T12:52:54" label="" />
      )
    ).toMatchSnapshot()
  })

  it("correctly renders date when type is string as number in seconds", () => {
    expect(
      shallow(<FormattedContractElement type="string" value="1536262055" label="created_at" />)
    ).toMatchSnapshot()
  })

  it("correctly renders date when type is string as number in milliseconds", () => {
    expect(
      shallow(<FormattedContractElement type="string" value="1536262086756" label="created_at" />)
    ).toMatchSnapshot()
  })
})
