import { addGraph, models } from "nvd3"
import { select } from "d3"
import * as React from "react"
import { theme, styled } from "../../theme"
import { Cell } from "../ui-grid/ui-grid.component"

const StyledSvg: React.ComponentType<any> = styled.svg`
  height: 150px;
  width: 150px;
`
export interface DonutParams {
  data: DonutData[]
  colors: string[]
}

export interface DonutData {
  label: string
  value: number
  renderToolTip?: () => JSX.Element
  renderWrapper?: (value: any) => JSX.Element
}

export const DonutChartContainer: React.ComponentType<any> = styled.div`
  width: 100%;
  height: 100%;
  // padding: 10px;
  position: relative;
  box-sizing: border-box;
`

export const DonutChartCenterWrapper: React.ComponentType<any> = styled.div`
  position: absolute;
  left: 0px;
  top: 0px;
  height: 100%;
  width: 100%;
  box-sizing: border-box;
  display: flex;
  justify-content: center;
  align-items: center;
`

const DonutChartCenter: React.ComponentType<any> = styled(Cell)`
  position: relative;
  max-width: 400px;
  white-space: normal;
  display: flex;
  width: 160px;
  height: 160px;
  line-height: 18px;
  align-items: center;
  justify-content: center;
  text-align: center;
`

export const DonutChart: React.SFC<{ id: string; params: DonutParams; centerContent: string }> = ({
  id,
  params,
  centerContent
}) => {
  addGraph(() => {
    const chart = models
      .pieChart()
      .x((d: DonutData) => d.label)
      .y((d: DonutData) => d.value)
      .showLabels(false)
      .labelThreshold(0.05)
      .labelType("percent")
      .donut(true)
      .donutRatio(0.7)
      .color(params.colors)
      .showLegend(false)
      .width(160)
      .height(160)
      .margin({ top: 10, left: 10 })

    chart.tooltip.contentGenerator((d) => {
      return `<h2 style="font-weight:bold;color: ${theme.colors.primary};">${d.data.value}</h2>`
    })

    select(`#${id} svg`)
      .datum(params.data)
      .transition()
      .duration(350)
      .call(chart)

    return chart
  })

  return (
    <DonutChartContainer id={id}>
      <DonutChartCenterWrapper>
        <DonutChartCenter fontSize={[1]}>{centerContent}</DonutChartCenter>
      </DonutChartCenterWrapper>
      <StyledSvg />
    </DonutChartContainer>
  )
}
