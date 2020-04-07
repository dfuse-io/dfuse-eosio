import * as React from "react"
import { Text } from "../text/text.component"
import { Cell, Grid } from "../ui-grid/ui-grid.component"
import { styled } from "../../theme"

interface Props {
  label: string
  compact?: boolean
  mb?: number
  color?: string
}

export const CellValue: React.ComponentType<any> = styled(Cell)`
  word-break: break-word;
  white-space: normal;
`

export const DetailLine: React.SFC<Props> = ({ label, compact, color, mb, children }) => {
  let templateColumns = ["1fr", "2fr 3fr"]
  if (compact === true) {
    templateColumns = ["2fr 3fr", "2fr 6fr"]
  }

  return (
    <Grid mb={[mb !== undefined ? mb : 2]} gridTemplateColumns={templateColumns}>
      <Text color={color || "text"} fontWeight="bold">
        {label}&nbsp;
      </Text>
      <CellValue>{children}</CellValue>
    </Grid>
  )
}

export const DetailLineAuto: React.SFC<Props> = ({ label, color, mb, children }) => {
  const templateColumns = ["1fr", "auto 3fr"]

  return (
    <Grid mb={[mb !== undefined ? mb : 2]} gridTemplateColumns={templateColumns}>
      <Text color={color || "text"} fontWeight="bold">
        {label}&nbsp;
      </Text>
      <CellValue>{children}</CellValue>
    </Grid>
  )
}
