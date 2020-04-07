import { Text } from "../text/text.component"
import { Cell, Grid } from "../ui-grid/ui-grid.component"
import { styled } from "../../theme"
import Box from "../ui-box/ui-box.component"
import * as React from "react"

export const PillClickable: React.ComponentType<any> = styled(Box)`
  &:hover {
    cursor: pointer;
  }
`

export const PillWrapper: React.ComponentType<any> = styled(Cell)`
  min-width: 680px;
  height: auto;
  overflow: hidden;
`

export const PillContainer: React.ComponentType<any> = styled(Grid)`
  &:hover {
    border: 1px solid ${(props) => props.theme.colors.bleu10};
  }
  box-shadow: 0 0 1px 0px white inset, 0 0 1px 0px white;
  border-radius: 28px;
  border: 1px solid #d0d2d3;
`

export const HoverablePillContainer: React.ComponentType<any> = styled(PillContainer)``

export const PillContainerDetails: React.ComponentType<any> = styled(Cell)`
  border-left: 1px solid #d0d2d3;
  border-right: 1px solid #d0d2d3;
  border-bottom: 1px solid #d0d2d3;
  word-break: break-all;
  white-space: normal;
`

export const PillOverviewRow: React.ComponentType<any> = styled(Grid)`
  grid-auto-flow: column;
  display: grid;
  grid-template-columns: auto 1fr auto;
  grid-auto-columns: max-content auto;
`

export const PillInfoContainer: React.ComponentType<any> = styled(Cell)`
  background-color: white;
  font-size: 14px;
  font-family: "Roboto Mono", monospace;
`

export const PillExpandedContainer: React.ComponentType<any> = styled(Grid)`
  border-top: 1px solid #d0d2d3;
  grid-template-columns: max-content auto;
  width: 100%;

  overflow-x: auto;
`

export const AnimatedPillContainer: React.ComponentType<any> = styled(Cell)`
  transition: max-height 0.3s;
  transition-timing-function: ease-in-out;
`

export const PillExpandButton: React.ComponentType<any> = styled.button`
  background-color: ${(props) => props.theme.colors.traceRawButtonBackground};
  border: none;
  outline: none;
  cursor: pointer;
  color: ${(props) => props.theme.colors.traceRawButtonText};
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  font-size: 10px;
  font-family: "Roboto Mono", monospace;
  font-weight: bold;
  padding-top: 10px;
  text-transform: uppercase;
`

export const PillHeaderText: React.ComponentType<any> = styled(Text)`
  font-family: "Roboto Mono", monospace;
`

export const PillLogoContainer: React.ComponentType<any> = styled(Cell)`
  border: 1px solid #8d939a;
  width: 28px;
  height: 28px;
  position: absolute;
  left: 0px;
  top: 0px;
  box-sizing: border-box;
  z-index: 999;
  border-radius: 50%;
  background: #fff;
`

export const PillLogo: React.ComponentType<any> = styled(Cell)`
  &:hover {
    cursor: pointer;
  }
`
