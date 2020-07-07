import * as React from "react"
import { truncateString } from "@dfuse/explorer"
import { styled } from "../../theme"
import { Text } from "../text/text.component"
import { Cell } from "../ui-grid/ui-grid.component"

const InfoText: React.ComponentType<any> = styled(Text)`
  font-family: "'Roboto Condensed', sans-serif";
`

interface Props {
  info: string
}

export const PillInfo: React.FC<Props> = ({ info }) => {
  return (
    <Cell alignSelf={["center"]} borderLeft="1px dotted #aaa">
      <InfoText alignSelf="center" pl={[2]} pr={[3]} color="traceMemoText" fontSize={[2]}>
        {truncateString(info, 50)}
      </InfoText>
    </Cell>
  )
}
