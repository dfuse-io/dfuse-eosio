import * as React from "react"
import { Cell } from "../../atoms/ui-grid/ui-grid.component"
import { styled } from "../../theme"
import { width, space } from "styled-system"

const ContentWrapper: React.ComponentType<any> = styled.div`
  margin: 0 auto;
  ${width};
`
const OuterBannerWrapper: React.ComponentType<any> = styled.div`
  ${space};
  width: 100%;

  background-color: ${(props) => props.theme.colors.banner};
`

export const PageContainer: React.SFC<{}> = ({ children }) => {
  const childrenArray = React.Children.toArray(children)

  if (childrenArray.length === 1) {
    return (
      <Cell mt={[3]} mx="auto" maxWidth={["1800px"]} px={[2, 3, 4]}>
        <ContentWrapper>{childrenArray[0]}</ContentWrapper>
      </Cell>
    )
  }

  return (
    <Cell mx="auto">
      <OuterBannerWrapper py={[2, 3]} mb={[3]}>
        <Cell mx="auto" maxWidth={["1800px"]} px={[2, 3, 4]}>
          {childrenArray[0]}
        </Cell>
      </OuterBannerWrapper>
      <Cell mx="auto" maxWidth={["1800px"]} px={[2, 3, 4]}>
        <ContentWrapper>{childrenArray[1]}</ContentWrapper>
      </Cell>
    </Cell>
  )
}
