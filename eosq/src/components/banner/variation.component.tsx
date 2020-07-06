import * as React from "react"
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome"
import { formatVariation, Box } from "@dfuse/explorer"
import { Text } from "../../atoms/text/text.component"

import { faSortDown, faSortUp } from "@fortawesome/free-solid-svg-icons"

import { t } from "i18next"
import { UiToolTip } from "../../atoms/ui-tooltip/ui-tooltip"
import { theme, styled } from "../../theme"

const TrendIconMarginRight = "8px"

const TrendUpIcon: React.ComponentType<any> = styled(FontAwesomeIcon)`
  color: ${(props) => props.theme.colors.trendUp};
  margin-right: ${TrendIconMarginRight};
  transform: translate(0, +4px);
`

const TrendDownIcon: React.ComponentType<any> = styled(FontAwesomeIcon)`
  color: ${(props) => props.theme.colors.trendDown};
  margin-right: ${TrendIconMarginRight};
  transform: translate(0, -4px);
`

type Props = {
  textColor: string
  variation: number
}

export class AmountVariation extends React.Component<Props> {
  renderIcon(iconName: any, IconComponent: any) {
    return <IconComponent size="lg" icon={iconName} />
  }

  render() {
    const formattedVariation = formatVariation(Math.abs(this.props.variation))
    const isUp = this.props.variation >= 0
    const iconName = isUp ? faSortUp : faSortDown
    const IconComponent = isUp ? TrendUpIcon : TrendDownIcon
    return (
      <Box flexDirection="row">
        {this.renderIcon(iconName, IconComponent)}
        <UiToolTip>
          <Text
            fontSize={[3]}
            borderBottom={`2px dotted ${theme.colors.bleu9}`}
            fontWeight="400"
            color={this.props.textColor}
          >
            {formattedVariation}%
          </Text>
          <Text color="primary" p={[3]}>
            {t("banner.tooltip.last_24h_change")}
          </Text>
        </UiToolTip>
      </Box>
    )
  }
}
