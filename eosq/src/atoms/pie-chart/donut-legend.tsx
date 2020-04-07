import { DonutData, DonutParams } from "./donut-chart"
import * as React from "react"
import { Cell, Grid } from "../ui-grid/ui-grid.component"
import { styled } from "../../theme"
import { Text } from "../text/text.component"
import numeral from "numeral"
import { ColorTile } from "../color-tile/color-tile"

const DonutChartLegendContainer: React.ComponentType<any> = styled.div`
  margin-bottom: 5px;
`

const renderEntry = (entry: DonutData, units: string) => {
  if (entry.renderWrapper) {
    return entry.renderWrapper(entry.value)
  }

  return (
    <Text fontSize={[3, 2]} alignSelf="center" lineHeight="20px" fontWeight="bold">
      {numeral(entry.value).format("0,0.0000")} {units}
    </Text>
  )
}

const renderContent = (data: DonutData[], legendColors: string[], units: string) => {
  return data.map((entry: DonutData, index: number) => {
    return (
      <Grid key={index} gridColumnGap={[10]} gridTemplateColumns={["20px 1fr"]} pb="0px">
        <Cell alignSelf="center" lineHeight="20px">
          <ColorTile bg={legendColors[index]} />
        </Cell>
        <Grid gridTemplateColumns={["1fr", "1fr 1fr"]}>
          <Cell alignSelf="center">
            <Text fontSize={[1, 2]} alignSelf="center" lineHeight={["20px", "20px"]}>
              {entry.label}
            </Text>
          </Cell>
          <Cell alignSelf={["left", "right"]} justifySelf={["left", "right"]} lineHeight="30px">
            {renderEntry(entry, units)}
          </Cell>
        </Grid>
      </Grid>
    )
  })
}

export const DonutChartLegend: React.SFC<{ id: string; params: DonutParams; units: string }> = ({
  id,
  params,
  units
}) => {
  return (
    <DonutChartLegendContainer>
      {renderContent(params.data, params.colors, units)}
    </DonutChartLegendContainer>
  )
}
