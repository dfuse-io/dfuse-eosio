import * as React from "react"
import { styled } from "../../theme"
import { Text } from "../text/text.component"
import { Cell, Grid } from "../ui-grid/ui-grid.component"

const Wrapper: React.ComponentType<any> = styled(Grid)`
  border: 1px solid ${(props: any) => props.theme.colors.border};
  grid-auto-flow: row;
  min-width: 0px;
`

const BorderLessWrapper: React.ComponentType<any> = styled(Grid)`
  min-width: 0px;
`

type Props = {
  title?: string | JSX.Element
  subtitle?: string
  overflowX?: string
  renderSideTitle?: () => JSX.Element
}

export const Panel: React.SFC<Props> = ({
  title,
  subtitle,
  overflowX,
  renderSideTitle,
  children
}) => {
  return (
    <Wrapper bg="panelBackground" overflowX={overflowX}>
      <Grid
        gridTemplateColumns={title && renderSideTitle ? ["1fr", "1fr auto", "1fr auto"] : ["1fr"]}
      >
        {title ? (
          <Cell gridRow={["1", "1", "1"]} gridColumn={["1", "1", "1"]}>
            <Text
              alignSelf="left"
              px={[3, 3, 4]}
              py={[3, 3, 4]}
              color="header"
              fontSize={[5]}
              fontWeight="500"
            >
              {title}
            </Text>
          </Cell>
        ) : null}
        {renderSideTitle ? (
          <Cell
            gridRow={["2", "1", "1"]}
            gridColumn={["1", "2", "2"]}
            alignSelf={["end"]}
            justifySelf={["right", "right", "right"]}
            px={[2, 3, 4]}
            py={[2, 3, 4]}
          >
            {renderSideTitle()}
          </Cell>
        ) : null}
      </Grid>
      <Cell overflow="hidden">{children}</Cell>
    </Wrapper>
  )
}

export const BorderLessPanel: React.SFC<Props> = ({
  title,
  subtitle,
  overflowX,
  renderSideTitle,
  children
}) => {
  return (
    <BorderLessWrapper bg="panelBackground" overflowX={overflowX}>
      <Grid
        gridTemplateColumns={title && renderSideTitle ? ["1fr", "1fr 1fr", "1fr 1fr"] : ["1fr"]}
      >
        {title ? (
          <Cell gridRow={["1", "1", "1"]} gridColumn={["1", "1", "1"]}>
            <Text
              textTransform="capitalize"
              alignSelf="left"
              px={[3, 3, 4]}
              py={[3, 3, 4]}
              color="neutral"
              fontSize={[5]}
              fontWeight="700"
            >
              {title}
            </Text>
          </Cell>
        ) : null}
        {renderSideTitle ? (
          <Cell
            gridRow={["2", "1", "1"]}
            gridColumn={["1", "2", "2"]}
            alignSelf="right"
            justifySelf="right"
            px={[2, 3, 4]}
            py={[2, 3, 4]}
          >
            {renderSideTitle ? renderSideTitle() : null}
          </Cell>
        ) : null}
      </Grid>
      <Cell overflow="hidden">{children}</Cell>
    </BorderLessWrapper>
  )
}
