import { styled } from "../../theme"
import { Cell } from "../ui-grid/ui-grid.component"
import * as React from "react"

export const Badge: React.ComponentType<any> = styled(Cell)`
  color: ${(props: any) => props.theme.colors.primary};
  font-family: "Roboto Condensed";
  text-align: center;
  line-height: 24px;
  font-size: 11px;
  width: 25px;
  height: 25px;
  border-radius: 50%;
  margin-right: 10px;
  display: inline-block;
  border: 1px solid ${(props) => props.theme.colors.primary};
`

export const BadgeContainer = styled.div`
  min-width: 100px;
  display: flex;
`
