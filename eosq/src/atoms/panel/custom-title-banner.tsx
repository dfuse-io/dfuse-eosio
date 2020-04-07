import * as React from "react"
import { styled } from "../../theme"
import { Cell, Grid } from "../ui-grid/ui-grid.component"
import { Text } from "../text/text.component"

const Container: React.ComponentType<any> = styled(Cell)`
  background-color: ${(props) => props.theme.colors.banner};
  margin-bottom: 1px;
  border-style: solid;
  border-color: ${(props) => props.theme.colors.bleu6};
`

export interface BannerContainerProps {
  contentLeft: string | JSX.Element | JSX.Element[]
  children?: any
  contentRight?: string | JSX.Element | JSX.Element[]
  rest?: any
}

export const CustomTitleBanner: React.SFC<BannerContainerProps> = ({
  contentLeft,
  contentRight,
  ...rest
}) => (
  <Container
    borderTop={["0px"]}
    borderBottom={["0px"]}
    borderLeft={["0px", "1px", "1px"]}
    borderRight={["0px", "1px", "1px"]}
    px={[3, 4]}
    pt={[1, 2]}
    pb={[1, 2]}
    {...(rest || {})}
  >
    <Grid gridTemplateColumns={["auto", "auto 100px", "auto 100px"]}>
      <Cell alignSelf="center" gridColumn={["1", "1"]} gridRow={["1", "1"]} mr={[2, 4]}>
        {contentLeft}
      </Cell>
      {contentRight ? (
        <Cell
          mt={[2, 0]}
          gridColumn={["1", "2", "2"]}
          gridRow={["2", "1"]}
          wordBreak="break-all"
          pr={[2, 3]}
          alignSelf="center"
        >
          <Text fontSize={[4]} fontWeight="800" fontFamily="Roboto Condensed" color="primary">
            {contentRight}
          </Text>
        </Cell>
      ) : null}
    </Grid>
  </Container>
)
