import { styled } from "../../theme"
import * as React from "react"

export const LegendColor: React.ComponentType<any> = styled.div`
  background-color: ${(props: any) => {
    return props["background-color"]
  }};
  width: 14px;
  height: 14px;
`
