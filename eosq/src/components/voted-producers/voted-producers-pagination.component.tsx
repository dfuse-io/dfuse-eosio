import * as React from "react"
import { Cell, Grid } from "../../atoms/ui-grid/ui-grid.component"
import { styled } from "../../theme"
import { LinkStyledText, Text } from "../../atoms/text/text.component"
import { pagination } from "../../helpers/pagination-algorithm"
import { NavigationButton } from "../../atoms/navigation-buttons/navigation-buttons"

const PageNumberUl: React.ComponentType<any> = styled.ul`
  text-align: center;
  list-style: none;
  padding: 5px;
  padding-left: 30px;
  margin-top: 0px;
  margin-bottom: 0px;
`
const PageNumberLi: React.ComponentType<any> = styled.li`
  color: ${(props) => props.theme.colors.highlight};
  display: inline-block;
  margin-right: 20px;
`

const VotedProducerPaginationContainer: React.ComponentType<any> = styled(Grid)`
  width: 100%;
`

export const VotedProducerPagination: React.SFC<{
  currentPage: number
  numberOfPages: number
  showNext: boolean
  showPrev: boolean
  onClickPage: (offset: number) => void
  onPrevClick: () => void
  onNextClick: () => void
}> = ({
  currentPage,
  numberOfPages,
  showPrev,
  showNext,
  onPrevClick,
  onNextClick,
  onClickPage
}) => {
  function renderPageNumbers(pageCount: number) {
    const pages = pagination(currentPage, pageCount)
    return pages.map((pageNumber: string, index: number) => {
      if (pageNumber === "...") {
        return (
          <PageNumberLi key={index}>
            <Text>{pageNumber}</Text>
          </PageNumberLi>
        )
      }

      const pageNumberInt = parseInt(pageNumber, 10)
      return (
        <PageNumberLi key={index}>
          <LinkStyledText
            color={currentPage === pageNumberInt ? "linkHover" : "link"}
            onClick={() => onClickPage(pageNumberInt)}
          >
            {pageNumberInt + 1}
          </LinkStyledText>
        </PageNumberLi>
      )
    })
  }

  return (
    <VotedProducerPaginationContainer gridTemplateColumns={["1fr 6fr 1fr"]}>
      <Cell alignSelf="center" onClick={showPrev ? onPrevClick : null} justifySelf="left">
        {" "}
        {showPrev ? (
          <NavigationButton variant="light" direction="previous" onClick={() => onPrevClick()} />
        ) : null}
      </Cell>
      <Cell>
        <PageNumberUl>{renderPageNumbers(numberOfPages)}</PageNumberUl>
      </Cell>
      <Cell alignSelf="center" onClick={showPrev ? onNextClick : null} justifySelf="right">
        {showNext ? (
          <NavigationButton variant="light" direction="next" onClick={() => onNextClick()} />
        ) : null}
      </Cell>
    </VotedProducerPaginationContainer>
  )
}
