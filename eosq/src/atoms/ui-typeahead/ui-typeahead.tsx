import * as React from "react"
import Downshift from "downshift"
import MenuItem from "@material-ui/core/MenuItem/MenuItem"
import { UiPaper, UiSearch } from "../ui-text-field/ui-text-field"
import { Suggestion, SuggestionSection } from "../../models/typeahead"
import { UiTypeaheadFetcher } from "./ui-typeahead-fetcher"
import { Cell, Grid } from "../ui-grid/ui-grid.component"
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome"
import { faUser, faTimes, faSearch, faCube } from "@fortawesome/free-solid-svg-icons"
import { theme, styled } from "../../theme"
import { EllipsisText, Text } from "../text/text.component"
import { UiHrDense } from "../ui-hr/ui-hr"
import { t } from "i18next"
import { Spinner } from "@dfuse/explorer"

type OnCloseListener = () => void

const PaperContainer: React.ComponentType<any> = styled.div`
  position: relative;
`

const SearchIcon: React.ComponentType<any> = styled(FontAwesomeIcon)`
  width: 18px;
`

const SearchWrapper: React.ComponentType<any> = styled.div`
  position: relative;
`

const RoundedSpinnerCube: React.ComponentType<any> = styled(Spinner)`
  width: 28px !important;
  height: 28px !important;
  margin-right: auto;
  margin-left: auto;
`

const SearchButton: React.ComponentType<any> = styled.div`
  position: absolute;
  border: none !important;

  cursor: pointer;
  color: #fff;
  border-radius: 0px !important;
  padding: 0 6px;

  &:disabled {
    cursor: inherit;
  }

  width: fit-content;

  top: 19px;
  right: 11px;
  font-size: 38px;

  @media (max-width: 767px) {
    top: 12px;
    left: 8px;
    font-size: 28px;
  }
`

const SyntaxBox: React.ComponentType<any> = styled.div`
  width: auto;
  font-size: 16px;
  border-radius: 3px;
  padding: 3px 1px;
  line-height: 16px;
  margin-bottom: 5px;
  height: auto;
  display: inline-block;
  background-color: rgba(101, 101, 111, 0.1);
`

const DeleteIcon: React.ComponentType<any> = styled(FontAwesomeIcon)`
  position: absolute;

  color: #fff;
  &:disabled {
    cursor: inherit;
  }
  &:hover {
    cursor: pointer;
  }
  top: 29px;
  right: 70px;

  @media (max-width: 767px) {
    top: 18px;
    right: 15px;
  }
`

const ResponsiveContainer: React.ComponentType<any> = styled(Cell)`
  @media (max-width: 767px) {
    div:nth-child(1n + 4) {
      display: none;
    }
  }
`

interface Props {
  placeholder: string
  defaultQuery?: string
  getItems: (inputValue: string | null) => Promise<SuggestionSection[]>
  help?: JSX.Element | string
  clickToFollowTypes?: string[]
  onSubmit: (query: string) => Promise<any>
}

interface State {
  value: string
  searching: boolean
}

function renderInput(inputProps: any) {
  const { InputProps, ref, ...other } = inputProps

  return (
    <UiSearch
      autoCapitalize="none"
      inputRef={ref}
      {...InputProps}
      {...other}
      width="100%"
      disableUnderline={true}
    />
  )
}

function formatBold(content: string) {
  const regex: RegExp = /(\S*:)/g
  return (
    <EllipsisText whiteSpace="pre-wrap !important" fontFamily="Roboto Condensed" fontSize={[3]}>
      {content.split(regex).map((value: string, index: number) => {
        if (regex.test(value)) {
          return <SyntaxBox key={index}>{value}</SyntaxBox>
        }

        return value
      })}
    </EllipsisText>
  )
}

function renderSummary(summary: string, accountName: string, isHighlighted: boolean) {
  return (
    <Text
      pt={[2, 0]}
      fontSize={[2]}
      lineHeight={[2]}
      color={isHighlighted ? "white" : theme.colors.grey5}
    >
      {t(`search.suggestions.summary.${summary}`, { accountName })}
    </Text>
  )
}

export class UiTypeahead extends React.Component<Props, State> {
  suggestions: SuggestionSection[] = []

  constructor(props: Props) {
    super(props)
    this.state = {
      value: props.defaultQuery ? props.defaultQuery : "",
      searching: false
    }
  }

  componentDidUpdate(prevProps: Props): void {
    if (this.props.defaultQuery && this.props.defaultQuery !== prevProps.defaultQuery) {
      // eslint-disable-next-line react/no-did-update-set-state
      this.setState({
        value: this.props.defaultQuery,
        searching: false
      })
    }
  }

  renderSuggestion(params: {
    groupId: string
    suggestion: { label: string; summary?: string }
    index: number
    itemProps: any
    highlightedIndex: number | null
    selectedItem: string
  }) {
    const isHighlighted =
      params.highlightedIndex !== null ? params.highlightedIndex === params.index : false
    const isSelected = (params.selectedItem || "").indexOf(params.suggestion.label) > -1
    let icon: any = faSearch

    if (params.groupId === "accounts") {
      icon = faUser
    } else if (params.groupId === "blocks") {
      icon = faCube
    }

    return (
      <MenuItem
        {...params.itemProps}
        key={params.suggestion.label}
        selected={isHighlighted}
        component="div"
        className={params.groupId}
        style={{
          fontWeight: isSelected ? 600 : 400,
          backgroundColor: isHighlighted ? theme.colors.green5 : "white",
          color: isHighlighted ? "white" : "black",
          height: "auto"
        }}
      >
        <Grid width="100%" gridTemplateColumns={["30px 1fr"]}>
          <Cell display="inline-block" mr={[3]}>
            <FontAwesomeIcon icon={icon} />
          </Cell>
          <Cell>{formatBold(params.suggestion.label)}</Cell>

          <Cell gridColumn={["2"]}>
            {params.suggestion.summary
              ? renderSummary(params.suggestion.summary, this.state.value, isHighlighted)
              : null}
          </Cell>
        </Grid>
      </MenuItem>
    )
  }

  getHighlightedItem(
    highlightedIndex: number
  ): { suggestion: Suggestion | undefined; id: string | undefined } {
    let index = 0
    let suggestion
    let id
    this.suggestions.forEach((suggestionGroup: SuggestionSection) => {
      suggestionGroup.suggestions.forEach((suggestionRef: Suggestion) => {
        if (index === highlightedIndex) {
          suggestion = suggestionRef
          id = suggestionGroup.id
        }
        index += 1
      })
    })

    return { suggestion, id }
  }

  getItemByValue(
    selectedItemValue: string
  ): { suggestion: Suggestion | undefined; id: string | undefined; index: number } {
    let suggestion
    let id
    let index = 0
    let i = 0
    this.suggestions.forEach((suggestionGroup: SuggestionSection) => {
      suggestionGroup.suggestions.forEach((suggestionRef: Suggestion) => {
        if (suggestionRef.key === selectedItemValue) {
          suggestion = suggestionRef
          id = suggestionGroup.id
          index = i
        }
        i += 1
      })
    })

    return { suggestion, id, index }
  }

  handleStateChange = (changes: any) => {
    if (changes.type === Downshift.stateChangeTypes.clickItem) {
      // eslint-disable-next-line no-prototype-builtins
      const ref = changes.hasOwnProperty("selectedItem")
        ? this.getItemByValue(changes.selectedItem)
        : this.getItemByValue(this.state.value)
      if (ref.id && (this.props.clickToFollowTypes || []).includes(ref.id)) {
        this.onSubmitInternal(() => {
          return undefined
        }, ref.index)
      } else {
        this.setState({ value: changes.selectedItem })
      }
    }

    // eslint-disable-next-line no-prototype-builtins
    if (changes.hasOwnProperty("selectedItem")) {
      this.setState({ value: changes.selectedItem })
      // eslint-disable-next-line no-prototype-builtins
    } else if (changes.hasOwnProperty("inputValue")) {
      this.setState({ value: changes.inputValue })
    }
  }

  handleKeyDown = (
    event: KeyboardEvent,
    closeMenu: (cb?: OnCloseListener) => any,
    selectHighlightedItem: (params: any, cb: any) => any,
    highlightedIndex: number | null
  ) => {
    if (event.keyCode === 13) {
      if (highlightedIndex !== null) {
        selectHighlightedItem({}, () => {
          const ref = this.getHighlightedItem(highlightedIndex)
          if (ref.id && (this.props.clickToFollowTypes || []).includes(ref.id)) {
            this.onSubmitInternal(closeMenu, highlightedIndex)
          }
        })
      } else {
        this.onSubmitInternal(closeMenu, null)
      }
    }
  }

  handleInputChange = (event: any) => {
    this.setState({ value: event.target.value })
  }

  resetField = () => {
    this.setState({
      value: ""
    })
  }

  renderItems(
    items: SuggestionSection[],
    getItemProps: any,
    highlightedIndex: number | null,
    selectedItem: any
  ) {
    let totalIndex = 0
    return (items || []).map((suggestionGroup: SuggestionSection, index: number) => {
      if (suggestionGroup.suggestions && suggestionGroup.suggestions.length > 0) {
        let groupItems: any = []

        const groupItemsContent = suggestionGroup.suggestions.map((suggestion: Suggestion) => {
          const render = this.renderSuggestion({
            groupId: suggestionGroup.id,
            suggestion,
            index: totalIndex,
            itemProps: getItemProps({ item: suggestion.label }),
            highlightedIndex,
            selectedItem
          })
          totalIndex += 1
          return render
        })

        groupItems = [
          ...groupItems,
          <ResponsiveContainer key={suggestionGroup.id}>{groupItemsContent}</ResponsiveContainer>
        ]

        if (index < items.length - 1) {
          groupItems = groupItems.concat([<UiHrDense key={`${index}-separator`} />])
        }

        return groupItems
      }
      return null
    })
  }

  onSubmitInternal = (
    closeMenu: (cb?: OnCloseListener) => any,
    highlightedIndex: number | null
  ) => {
    if (!this.state.searching) {
      closeMenu()
      let { value } = this.state
      let suggestionWithId
      if (highlightedIndex !== null) {
        suggestionWithId = this.getHighlightedItem(highlightedIndex)
        if (suggestionWithId.suggestion) {
          value = suggestionWithId.suggestion.label
        }
      }

      this.setState({ value, searching: true }, () => {
        this.props.onSubmit(value).then(
          () => {
            this.setState({ value, searching: false })
          },
          () => {
            this.setState({ value, searching: false })
          }
        )
      })
    }
  }

  renderSearchButton(closeMenu: (cb?: OnCloseListener) => any, highlightedIndex: number | null) {
    if (this.state.searching) {
      return (
        <SearchButton key="1" name="search" disabled={true}>
          <RoundedSpinnerCube color="white" fadeIn="none" name="double-bounce" />
        </SearchButton>
      )
    }

    return (
      <SearchButton
        key="2"
        onClick={() => this.onSubmitInternal(closeMenu, highlightedIndex)}
        name="search"
      >
        <SearchIcon icon={faSearch} />
      </SearchButton>
    )
  }

  renderDeleteIcon() {
    return this.state.value && this.state.value.length > 0 ? (
      <DeleteIcon onClick={this.resetField} icon={faTimes} size="lg" />
    ) : null
  }

  render() {
    let popperNode: any

    const { value } = this.state
    return (
      <Downshift selectedItem={value} onStateChange={this.handleStateChange}>
        {({
          getInputProps,
          getMenuProps,
          getItemProps,
          isOpen,
          selectedItem,
          inputValue,
          highlightedIndex,
          selectHighlightedItem,
          clearItems,
          setItemCount,
          getRootProps,
          closeMenu,
          setHighlightedIndex
        }) => (
          <SearchWrapper {...getRootProps()}>
            {renderInput({
              fullWidth: true,
              InputProps: getInputProps({
                onChange: this.handleInputChange,
                onKeyDown: (event: KeyboardEvent) =>
                  this.handleKeyDown(event, closeMenu, selectHighlightedItem, highlightedIndex),
                placeholder: this.props.placeholder
              }),
              ref: (node: any) => {
                popperNode = node
              }
            })}
            {this.renderSearchButton(closeMenu, highlightedIndex)}
            {this.renderDeleteIcon()}
            <PaperContainer {...getMenuProps()}>
              {isOpen ? (
                <UiPaper
                  square={true}
                  style={{ marginTop: 0, width: popperNode ? popperNode.clientWidth : null }}
                >
                  <UiTypeaheadFetcher
                    fetchData={this.props.getItems}
                    searchValue={inputValue}
                    onLoaded={(suggestions) => {
                      clearItems()
                      if (suggestions) {
                        // @ts-ignore
                        setHighlightedIndex(null)
                        setItemCount(
                          suggestions
                            .map(
                              (suggestionGroup: SuggestionSection) =>
                                suggestionGroup.suggestions.length
                            )
                            .reduce((sum: number, current) => sum + current, 0)
                        )
                        this.suggestions = suggestions
                      }
                    }}
                  >
                    {({ loading, suggestions, error }) => {
                      if (loading) {
                        return (
                          <Cell px={[3]} py={[3]}>
                            <Text fontSize={[3]}>{t("search.loading")}</Text>
                          </Cell>
                        )
                      }
                      if (error) {
                        return (
                          <Cell px={[3]} py={[3]}>
                            <Text fontSize={[3]}>{t("search.errorFetch")}</Text>
                          </Cell>
                        )
                      }

                      return this.renderItems(
                        suggestions,
                        getItemProps,
                        highlightedIndex,
                        selectedItem
                      )
                    }}
                  </UiTypeaheadFetcher>
                  {this.props.help}
                </UiPaper>
              ) : null}
            </PaperContainer>
          </SearchWrapper>
        )}
      </Downshift>
    )
  }
}
