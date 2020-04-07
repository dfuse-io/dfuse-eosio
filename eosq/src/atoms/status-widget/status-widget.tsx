import * as React from "react"
import { Cell, Grid } from "../ui-grid/ui-grid.component"
import { Text } from "../text/text.component"
import { theme, styled } from "../../theme"

const Container: React.ComponentType<any> = styled(Grid)`
  margin-bottom: 16px;
`

interface Props {
  title: string
  description: string | JSX.Element
  amount: JSX.Element
}

export const StatusWidget: React.SFC<Props> = ({ title, description, amount }) => {
  return (
    <Container gridTemplateColumns={["1fr 100px"]}>
      <Cell>
        <Text fontSize={[3]} fontWeight="700">
          {title}
        </Text>
        <Text fontSize={[2]} color={theme.colors.grey5}>
          {description}
        </Text>
      </Cell>
      <Cell alignSelf="right" justifySelf="right" textAlign="right">
        {amount}
      </Cell>
    </Container>
  )
}
