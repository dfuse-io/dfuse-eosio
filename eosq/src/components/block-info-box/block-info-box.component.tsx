import * as React from "react"
import { TextLink } from "../../atoms/text/text.component"
import { Links } from "../../routes"
import { Cell, Grid } from "../../atoms/ui-grid/ui-grid.component"
import { faLock, faLockOpen } from "@fortawesome/free-solid-svg-icons"
import { formatNumber } from "@dfuse/explorer"
import { TableIcon, TableIconLight } from "../../atoms/ui-table/ui-table.component"
import { formatDateFromString } from "../../helpers/moment.helpers"
import { UiToolTip } from "../../atoms/ui-tooltip/ui-tooltip"
import { styled, theme } from "../../theme"

interface Props {
  blockTime?: string
  blockNum: number
  blockId: string
  irreversible: boolean
}

const UnderlinedTextLink: React.ComponentType<any> = styled(TextLink)`
  border-bottom: 2px dotted ${theme.colors.grey4};
`

export class BlockInfoBox extends React.Component<Props> {
  renderLock() {
    return this.props.irreversible ? (
      <TableIcon icon={faLock} />
    ) : (
      <TableIconLight icon={faLockOpen} />
    )
  }

  render() {
    return (
      <Grid display="inline-block" gridTemplateColumns={["1fr auto 1fr"]} height="20px">
        <Cell>
          <UiToolTip>
            <UnderlinedTextLink to={Links.viewBlock({ id: this.props.blockId })}>
              {formatNumber(this.props.blockNum)}
            </UnderlinedTextLink>

            {this.props.blockTime ? formatDateFromString(this.props.blockTime, false) : null}
          </UiToolTip>
        </Cell>
        <Cell> {this.renderLock()} </Cell>
      </Grid>
    )
  }
}
