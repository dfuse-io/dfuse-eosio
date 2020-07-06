import { t } from "i18next"
import { observer } from "mobx-react"
import * as React from "react"
import { formatPercentage, Spinner } from "@dfuse/explorer"
import { styled } from "../../theme"
import { computeTransactionTrustPercentage } from "../../models/transaction"
import { Text } from "../../atoms/text/text.component"
import { Cell } from "../../atoms/ui-grid/ui-grid.component"

const MiddleChild: React.ComponentType<any> = styled(Cell)`
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  text-align: center;
  fill: #eee;
`

const ProgressSpinner: React.ComponentType<any> = styled(Spinner)`
  color: rgba(0, 0, 0, 0.7) !important;
  transform: scale(0.7);
`

const ProgressText: React.ComponentType<any> = styled(Text)`
  font-family: Roboto Condensed;
  text-align: center;
`

const ProgressSvg: React.ComponentType<any> = styled.svg`
  circle {
    transition: stroke-dashoffset 0s linear;
    stroke: rgba(255, 255, 255, 0.2);
    stroke-width: 8px;
    fill: #eee;
  }

  .progress-bar {
    stroke: ${(props) => props.theme.colors.ternary};
  }

  .progress-completed {
    stroke: #2a2a36;
  }
`

interface Props {
  blockNum: number
  headBlockNum: number
  lastIrreversibleBlockNum: number
}

@observer
export class BlockProgressPie extends React.Component<Props> {
  renderCircle(
    className: string,
    radius: number,
    offsetX: number,
    offsetY: number,
    dashArray: number,
    dashOffset: number
  ) {
    return (
      <circle
        className={className}
        r={radius}
        cx={offsetX}
        cy={offsetY}
        fill="transparent"
        transform={`rotate(-90 ${offsetX} ${offsetY})`}
        strokeDasharray={dashArray}
        strokeDashoffset={dashOffset}
      />
    )
  }

  renderProgressCircle(completion: number) {
    const radius = 35
    const diameter = radius * 2
    const offsetX = 50
    const offsetY = 50
    const dashArray = Math.PI * diameter
    const dashBarOffset = (1.0 - completion) * dashArray

    let completedCircle = null
    if (completion >= 1.0) {
      completedCircle = this.renderCircle("progress-completed", 44, offsetX, offsetY, 0, 0)
    }

    return (
      <ProgressSvg
        viewBox="0 0 100 100"
        width="100%"
        height="100%"
        version="1.1"
        xmlns="http://www.w3.org/2000/svg"
      >
        {completedCircle}
        {this.renderCircle("progress-bg", radius, offsetX, offsetY, dashArray, 0)}
        {this.renderCircle("progress-bar", radius, offsetX, offsetY, dashArray, dashBarOffset)}
      </ProgressSvg>
    )
  }

  renderProgressLoading() {
    return <ProgressSpinner name="three-bounce" fadeIn="none" />
  }

  renderProgressTrustRate(completion: number) {
    return (
      <div>
        <ProgressText color="text" fontSize={[3, 2]}>
          {formatPercentage(completion, 2)}
        </ProgressText>
        <ProgressText color="text" fontSize={[2, 0]}>
          {t("transaction.progressCircle.confidence").toLocaleUpperCase()}
        </ProgressText>
      </div>
    )
  }

  renderIrreversibleImage() {
    return <img width="60%" src="/images/picto-irreversible-03.svg" alt="irreversible" />
  }

  isDataProvided() {
    return this.props.blockNum && this.props.headBlockNum && this.props.lastIrreversibleBlockNum
  }

  renderProgressInfo(percentage: number) {
    if (!this.isDataProvided()) {
      return this.renderProgressLoading()
    }

    if (percentage < 0.9999) {
      return this.renderProgressTrustRate(percentage)
    }

    return this.renderIrreversibleImage()
  }

  render() {
    const percentage = computeTransactionTrustPercentage(
      this.props.blockNum,
      this.props.headBlockNum,
      this.props.lastIrreversibleBlockNum
    )

    return (
      <Cell
        style={{ position: "relative" }}
        h="160px"
        w="160px"
        alignSelf="center"
        justifySelf="center"
      >
        <MiddleChild>{this.renderProgressInfo(percentage)}</MiddleChild>
        {this.renderProgressCircle(percentage)}
      </Cell>
    )
  }
}
