import * as React from "react"
import { styled } from "../../theme"
import { Cell, Grid } from "../ui-grid/ui-grid.component"
import { Text } from "../text/text.component"

const Container: React.ComponentType<any> = styled(Cell)`
  background-color: ${(props) => props.theme.colors.banner};
  margin-bottom: 1px;
  // border-left: 1px solid ${(props) => props.theme.colors.bleu6}
  // border-right: 1px solid ${(props) => props.theme.colors.bleu6}
  border-style:solid;
  border-color: ${(props) => props.theme.colors.bleu6};
`

export interface BannerContainerProps {
  title: string
  children?: any
  content?: string | JSX.Element
  rest?: any
}

export const PannelTitleBanner: React.SFC<BannerContainerProps> = ({
  title,
  content,
  children,
  ...rest
}) => (
  <Container
    borderTop={["0px"]}
    borderBottom={["0px"]}
    borderLeft={["0px", "1px", "1 px"]}
    borderRight={["0px", "1px", "1px"]}
    px={[3, 4]}
    pt={[2, 3]}
    pb={[2, 3]}
    {...(rest || {})}
  >
    <Grid gridTemplateColumns={["auto 6fr 100px"]}>
      <Cell alignSelf="center" gridColumn={["1 / span 2", "1"]} gridRow={["1", "1"]} mr={[2, 4]}>
        <Text fontSize={[4]} color="bannerSubtitle">
          {title}
        </Text>
      </Cell>
      <Cell
        mt={[2, 0]}
        gridColumn={["1 / span 3", "2"]}
        gridRow={["2", "1"]}
        wordBreak="break-all"
        pr={[2, 3]}
        alignSelf="center"
      >
        <Text fontSize={[4]} fontWeight="800" fontFamily="Roboto Condensed" color="primary">
          {content}
        </Text>
      </Cell>
      <Cell alignSelf="center" justifySelf="right" gridColumn={["3", "3"]} gridRow={["1", "1"]}>
        {children}
      </Cell>
    </Grid>
  </Container>
)
