import * as React from "react"
import { styled } from "../../theme"
import { Cell } from "../ui-grid/ui-grid.component"

const StatusBarContainer: React.ComponentType<any> = styled(Cell)`
  height: 15px;
`

export const LargeStatusBarContainer: React.ComponentType<any> = styled(Cell)`
  height: 34px !important;
  // border: 1px solid ${(props) => props.theme.colors.neutral};
`

const StatusBarElement = styled(Cell)`
  height: 100%;
  display: inline-block;
`

export const StatusBar: React.SFC<{
  content: number[]
  color?: string
  total: number
  large?: boolean
  bg?: string
  children?: any
}> = ({ content, color, total, large, bg, children }) => {
  if (!total) {
    return <StatusBarContainer bg="barDataValue" />
  }
  const bgData = color || "barDataValue"
  let firstWidth = (content[0] * 100.0) / total
  if (firstWidth > 100 || content[0] === -1 || total === -1) {
    firstWidth = 100
    bg = "trendDown"
  }

  // const secondWidth = content.length > 1 ? content[1] * 100.0 / total : 0
  const Container = large ? LargeStatusBarContainer : StatusBarContainer
  return (
    <Container bg={bg !== undefined ? bg : "barBackground"}>
      <StatusBarElement bg={bgData} width={`${firstWidth}%`}>
        {children}
      </StatusBarElement>
    </Container>
  )
}
