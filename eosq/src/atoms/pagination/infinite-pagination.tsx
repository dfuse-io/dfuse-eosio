import * as React from "react"
import { styled } from "../../theme"
import { LinkStyledText } from "../text/text.component"
import { Grid } from "../ui-grid/ui-grid.component"

const InfinitePaginationContainer: React.ComponentType<any> = styled(Grid)`
  width: 100%;
  height: 80px;
  border-top: 1px solid #eee;
`

const NavigationLink: React.SFC<{
  onClick?: () => void
  textAlign: string
  justifySelf: string
}> = ({ onClick, children, ...rest }) => (
  <LinkStyledText p={[3]} fontSize={[3]} color="link" onClick={onClick} {...rest}>
    {children}
  </LinkStyledText>
)

export const InfinitePagination: React.SFC<{
  prevText: string
  nextText: string
  onPrevClick: () => void
  onNextClick: () => void
}> = ({ prevText, nextText, onPrevClick, onNextClick }) => {
  return (
    <InfinitePaginationContainer gridTemplateColumns={["1fr 1fr"]}>
      <NavigationLink
        onClick={prevText.length > 0 ? onPrevClick : undefined}
        justifySelf="left"
        textAlign="left"
      >
        {prevText}
      </NavigationLink>
      <NavigationLink onClick={onNextClick} justifySelf="right" textAlign="right">
        {nextText}
      </NavigationLink>
    </InfinitePaginationContainer>
  )
}
