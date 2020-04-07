import * as React from "react"
import { shallowWithTheme } from "../../../tests/renderers"
import { getMockRouterProps } from "../../../__mocks__/router"
import { AccountDetail } from "../account-detail.page"

describe("page account-detail", () => {
  it("renders error component if store has error", () => {
    expect(shallowWithTheme(render())).toMatchSnapshot()
  })
})

function render(transactionId: string = "1") {
  const routerProps = getMockRouterProps<{ id: string }>({ id: transactionId })

  return (
    <AccountDetail
      match={routerProps.match}
      location={routerProps.location}
      history={routerProps.history}
    />
  )
}
