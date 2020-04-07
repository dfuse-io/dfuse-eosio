import * as React from "react"
import moment from "moment"
import { Text } from "../text/text.component"
import { formatDateFromString } from "../../helpers/moment.helpers"
import { Cell } from "../ui-grid/ui-grid.component"

interface AgeProps {
  date: string | Date
  color?: string
}

export const Age: React.SFC<AgeProps> = ({ date, color }) => (
  <Cell>
    <Text display="inline-block">{formatDateFromString(date, false)}</Text>
    &nbsp;
    <Text display="inline-block" fontSize={[1]}>
      ({moment.utc(date, "YYYY-MM-DD hh:mm:ss Z").fromNow()})
    </Text>
  </Cell>
)
