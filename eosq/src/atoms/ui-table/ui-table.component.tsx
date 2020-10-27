import TableCell from "@material-ui/core/TableCell"
import TableHead from "@material-ui/core/TableHead"
import TableRow from "@material-ui/core/TableRow"
import Table from "@material-ui/core/Table"
import TableBody from "@material-ui/core/TableBody"
import { system, Scale } from "styled-system"
import { Box, Cell } from "@dfuse/explorer"
import { theme, styled, resolveValue } from "../../theme"
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome"
import * as React from "react"

type TableCaptionItemProps = {
  alignSelf?: any
  justifySelf?: any
  pl?: any
  textAlign?: any
  mr?: any
}

export const TableCaptionItem: React.SFC<TableCaptionItemProps> = ({ children, ...rest }) => (
  <Box mr={[4]} fontWeight="300" fontSize={[1]} color="#404041" {...rest}>
    {children}
  </Box>
)

type TableCaptionColorProps = {
  height?: any
  width?: any
  mr?: any
  bg?: any
}

export const TableCaptionColor: React.SFC<TableCaptionColorProps> = ({ children, ...rest }) => (
  <Cell mr={[2]} width="12px" height={["12px"]} {...rest}>
    {children}
  </Cell>
)

export const TableIcon: React.ComponentType<any> = styled(FontAwesomeIcon)`
  color: #aaa;
  margin-left: 5px;
`

export const TableIconLight: React.ComponentType<any> = styled(FontAwesomeIcon)`
  color: ${(props) => props.theme.colors.grey4};
  margin-left: 5px;
`

export const UiTable: React.ComponentType<any> = styled(Table)`
  grid-auto-flow: row;
`

export const UiTableHead: React.ComponentType<any> = styled(TableHead)`
  border-bottom: 1px solid ${(props) => props.theme.colors.border};
`

export const UiTableBody: React.ComponentType<any> = styled(TableBody)``

export const UiTableRow: React.ComponentType<any> = styled(TableRow)`
  min-height: 30px !important;
  height: auto !important;
`

export const UiTableRowAlternated: React.ComponentType<any> = styled(TableRow)`
  min-height: 30px !important;
  height: auto !important;
  &:nth-of-type(even) {
    background-color: ${(props) => props.theme.colors.tableEvenRowBackground};
  }
`

const fontSize = system({
  fontSize: {
    property: "fontSize",
    scale: "fontSizes",
    transform: (n: string | number, scale?: Scale) => `${resolveValue(n, scale)}px !important;`
  }
})

const textAlign = system({
  textAlign: {
    property: "textAlign",
    transform: (n: string | number) => `${n} !important;`
  }
})

export const UiTableCell: React.ComponentType<any> = styled(TableCell)`
  ${fontSize};
  ${textAlign};
  border-bottom: none !important;
  padding-top: 8px !important;
  padding-right: 20px !important;
  padding-left: 20px !important;
  padding-bottom: 8px !important;
  white-space: nowrap;
  position: relative;
  color: ${(props) => (props.color ? props.color : theme.colors.text)} !important;
`

export const UiTableCellNarrow: React.ComponentType<any> = styled(TableCell)`
  ${fontSize};
  ${textAlign};
  border-bottom: none !important;
  padding-top: 8px !important;
  padding-right: 5px !important;
  padding-left: 5px !important;
  padding-bottom: 8px !important;
  white-space: nowrap;
  position: relative;
  color: ${(props: any) => (props.color ? props.color : theme.colors.text)} !important;
`

export const UiTableCellTop: React.ComponentType<any> = styled(UiTableCell)`
  ${textAlign};
  vertical-align: top !important;
  padding-top: 20px !important;
`

export const UiTableCellPill: React.ComponentType<any> = styled(UiTableCell)``
