import { observer } from "mobx-react"
import * as React from "react"
import { Button } from "@material-ui/core"
import { Cell, Grid } from "../../atoms/ui-grid/ui-grid.component"
import { HoverableTextNoHighlight, Text } from "../../atoms/text/text.component"
import { theme, styled } from "../../theme"
import { UiInput } from "../../atoms/ui-text-field/ui-text-field"
import { searchStore } from "../../stores"
import { FilterSection, FilterTypes, RangeOptions } from "../../models/search-filters"
import { t } from "i18next"
import { RouteComponentProps } from "react-router"
import queryString from "query-string"
import { performStructuredSearch } from "../../services/search"
import { Box, formatNumber } from "@dfuse/explorer"
import { UiDropDown } from "../../atoms/ui-dropdown/ui-dropdown.component"
import Checkbox from "@material-ui/core/Checkbox/Checkbox"

import { UiSwitch } from "../../atoms/ui-switch/switch"

interface Props extends RouteComponentProps<any> {
  onApply: () => void
}

const StyledInput: React.ComponentType<any> = styled(UiInput)`
  height: 32px;
  width: auto !important;
  min-width: 160px !important;
  input {
    color: ${(props) => props.theme.colors.text};
  }
`

const StyledButton: React.ComponentType<any> = styled(Button)`
  padding-left: 40px !important;
  padding-right: 40px !important;
  background-color: ${(props) => props.theme.colors.ternary} !important;
  border: none !important;
  span {
    font-weight: 600 !important;
  }
  border-radius: 0px !important;
  height: 30px !important;
  min-height: 35px !important;
  color: ${(props) => props.theme.colors.primary} !important;
`

const StyledCheckbox: React.ComponentType<any> = styled(Checkbox)`
  padding: 0 !important;
`

const HoverableBox: React.ComponentType<any> = styled(Box)`
  &:hover {
    cursor: pointer;
  }
`

@observer
export class SearchFilters extends React.Component<Props> {
  BLOCK_RANGE_OPTIONS = [
    { label: t("filters.rangeOptions.custom"), value: RangeOptions.CUSTOM },
    { label: t("filters.rangeOptions.all"), value: RangeOptions.ALL },
    { label: t("filters.rangeOptions.lastBlocks"), value: RangeOptions.LAST_BLOCKS }
  ]
  get parsed() {
    return queryString.parse(this.props.location.search)
  }

  renderSectionTitle(type: string, color: string) {
    return (
      <Text
        fontFamily="'Roboto Condensed', sans-serif;"
        color={color}
        display="inline-block"
        fontSize={[2]}
        key="title"
      >
        {t(`filters.sections.titles.${type}`)}
      </Text>
    )
  }

  renderBlockInput(
    field: "min" | "max" | "lastBlocks",
    value: number,
    type: FilterTypes,
    label: string
  ) {
    return (
      <Grid justifySelf="left" gridTemplateRows={["1fr 1fr"]}>
        <Text
          justifySelf="left"
          display="inline-block"
          fontSize={[2]}
          color={theme.colors.bleu8}
          mr={[2]}
        >
          {label}
        </Text>
        <StyledInput
          inputProps={{ lang: "en-150", step: "1000000" }}
          onChange={(event: React.ChangeEvent<HTMLInputElement>) =>
            this.onChangeField(type, field, event)
          }
          disableUnderline={true}
          mr={[2]}
          value={value >= 0 ? formatNumber(value) : 0}
        />
      </Grid>
    )
  }

  onSelectBlockRangeOption = (value: string) => {
    searchStore.updateFilter(FilterTypes.BLOCK_RANGE, "option", value as RangeOptions)
  }

  renderBlockRangeDropDown(label: string) {
    return (
      <Grid justifySelf="left" gridTemplateRows={["1fr 1fr"]}>
        <Text
          justifySelf="left"
          display="inline-block"
          fontSize={[2]}
          color={theme.colors.bleu8}
          mr={[2]}
        >
          {label}
        </Text>
        <UiDropDown
          defaultValue={searchStore.rangeOption}
          options={this.BLOCK_RANGE_OPTIONS}
          onSelect={this.onSelectBlockRangeOption}
        />
      </Grid>
    )
  }

  displayBlockRange(filterSection: FilterSection) {
    const { data } = filterSection

    const content = (
      <Grid gridTemplateColumns={["1fr", "auto auto auto"]} gridRowGap={[2, 0]} key="block-range">
        <Cell pr={[4]} alignItems="center">
          {this.renderBlockRangeDropDown(t("filters.sections.titles.blockRange"))}
        </Cell>
        {searchStore.rangeOption === RangeOptions.CUSTOM
          ? [
              <Cell key="from" pr={[4]} alignItems="center">
                {this.renderBlockInput(
                  "min",
                  data.min,
                  filterSection.type,
                  t("filters.sections.labels.from")
                )}
              </Cell>,
              <Cell key="to" pr={[2]} alignItems="center">
                {this.renderBlockInput(
                  "max",
                  data.max,
                  filterSection.type,
                  t("filters.sections.labels.to")
                )}
              </Cell>
            ]
          : null}
        {searchStore.rangeOption === RangeOptions.LAST_BLOCKS
          ? [
              <Cell key="from" pr={[4]} alignItems="center">
                {this.renderBlockInput(
                  "lastBlocks",
                  data.lastBlocks,
                  filterSection.type,
                  t("filters.sections.labels.lastBlocks")
                )}
              </Cell>
            ]
          : null}
      </Grid>
    )

    return this.displayLine(content, filterSection.type)
  }

  onChangeField(type: FilterTypes, field: string, e: React.ChangeEvent<HTMLInputElement>) {
    const value: string = e.target.value.toString()
    searchStore.updateFilter(type, field, searchStore.parseField(field, value))
  }

  toggleIrreversibleOnly = () => {
    searchStore.updateFilter(
      FilterTypes.BLOCK_STATUS,
      "irreversibleOnly",
      searchStore.withReversible
    )
  }

  displayBlockStatus(filterSection: FilterSection) {
    const { data } = filterSection
    const content = (
      <HoverableBox
        onClick={() => this.toggleIrreversibleOnly()}
        key="block-status"
        alignItems="center"
        py={[4]}
      >
        <Cell display="inline-block">
          <StyledCheckbox
            checked={data.irreversibleOnly}
            color="default"
            onChange={(event: any) => {
              searchStore.updateFilter(
                FilterTypes.BLOCK_STATUS,
                "irreversibleOnly",
                event.target.checked
              )
            }}
          />
        </Cell>
        <Text display="inline-block" fontSize={[2]} color={theme.colors.bleu8} ml={[1]}>
          {t("filters.sections.labels.irreversible")}
        </Text>
        <Cell />
      </HoverableBox>
    )
    return this.displayLine(content, filterSection.type)
  }

  displayLine(content: JSX.Element, type: FilterTypes) {
    return (
      <Cell alignItems={["center"]} height={["auto", "auto"]} py={[2, 0]} key={type}>
        {content}
      </Cell>
    )
  }

  renderSections() {
    return searchStore.filterSections.map((section: FilterSection) => {
      let content: JSX.Element
      switch (section.type) {
        case FilterTypes.BLOCK_RANGE:
          content = this.displayBlockRange(section)
          break
        case FilterTypes.BLOCK_STATUS:
          content = this.displayBlockStatus(section)
          break
        default:
          throw new Error(`unknown FilterTypes (${section.type}), this shouldn't happen`)
      }

      return content
    })
  }

  onSubmit = () => {
    const cursor = performStructuredSearch(this.parsed.cursor as string)
    this.props.history.push(this.cursoredUrl(cursor as string))
    this.props.onApply()
  }

  cursoredUrl = (cursor: string) => {
    return searchStore.cursoredUrl(cursor)
  }

  renderButton() {
    return (
      <Cell textAlign={["center", "left"]} py={[3]}>
        <StyledButton onClick={this.onSubmit}>{t("filters.apply")}</StyledButton>
      </Cell>
    )
  }

  renderSort() {
    if (searchStore.blockRange.option === RangeOptions.LAST_BLOCKS) {
      return null
    }
    return (
      <Cell>
        <Cell>
          <Text color={theme.colors.bleu8}>SORT</Text>
        </Cell>
        <HoverableTextNoHighlight
          onClick={() => searchStore.toggleSort()}
          display="inline-block"
          fontSize={[2]}
          color={theme.colors.bleu8}
          mr={[1]}
        >
          Ascending
        </HoverableTextNoHighlight>
        <UiSwitch
          key={searchStore.sort}
          checked={searchStore.sort === "desc"}
          onChange={() => searchStore.toggleSort()}
        />
        <HoverableTextNoHighlight
          onClick={() => searchStore.toggleSort()}
          display="inline-block"
          fontSize={[2]}
          color={theme.colors.bleu8}
          ml={[1]}
        >
          Descending
        </HoverableTextNoHighlight>
      </Cell>
    )
  }

  render() {
    return (
      <Cell maxWidth="1800px" mx="auto">
        <Grid px={[2, 3]} gridTemplateRows={["1fr auto 1fr"]} key="b">
          {this.renderSections()}
          {this.renderSort()}
          {this.renderButton()}
        </Grid>
      </Cell>
    )
  }
}
