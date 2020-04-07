import { observable } from "mobx"
import {
  BlockRangeFilter,
  FilterCombinations,
  FilterSection,
  FilterTypes,
  RangeOptions
} from "../models/search-filters"
import { CursorCache } from "../services/cursor-store"
import { Links } from "../routes"
import { stringify } from "query-string"
import { BLOCK_NUM_100B, BLOCK_NUM_5M } from "../models/block"
import { SearchTransactionRow, ErrorData } from "@dfuse/client"
import {
  SearchQueryParams,
  LegacySearchQueryParams,
  upgradeLegacySearchQueryParams
} from "../models/search"

export function blockRangeToBlockParams(sort: string, data: any) {
  if (data.option === RangeOptions.ALL) {
    return { startBlock: 0, blockCount: BLOCK_NUM_100B }
  }

  if (data.option === RangeOptions.LAST_BLOCKS) {
    return { startBlock: 0, blockCount: data.lastBlocks }
  }

  let startBlock = data.max > 0 ? data.max : 0
  let blockCount = data.min > 0 ? startBlock - data.min : 0

  if (sort === "asc") {
    startBlock = data.min > 0 ? data.min : 0
    blockCount = data.max > 0 ? data.max - startBlock : 0
  }
  return { startBlock, blockCount }
}

export function blockParamsToBlockRange(
  sort: string,
  startBlock: number,
  blockCount: number
): { lastBlocks?: number; min?: number; max?: number; option: RangeOptions } {
  if (startBlock > 0 && startBlock - blockCount > 0) {
    if (sort === "desc") {
      const min = startBlock - blockCount
      return { min, max: startBlock, option: RangeOptions.CUSTOM }
    }

    const min = startBlock
    const max = startBlock + blockCount
    return { min, max, option: RangeOptions.CUSTOM }
  }

  if (blockCount > 0) {
    return { lastBlocks: blockCount, option: RangeOptions.LAST_BLOCKS }
  }

  return { lastBlocks: BLOCK_NUM_5M, option: RangeOptions.LAST_BLOCKS }
}

export class SearchStore {
  DEFAULT_PARAMS = {
    limit: 25,
    sort: "desc",
    withReversible: true,
    startBlock: 0,
    blockCount: BLOCK_NUM_100B
  }

  public cursorCache: CursorCache
  @observable hasNextPage: boolean
  @observable rangeOption: RangeOptions
  @observable query = ""
  @observable filterSections: FilterSection[]
  @observable loadingTransactions: boolean
  @observable searchError?: ErrorData
  @observable results: SearchTransactionRow[]
  @observable previousBlockRangeFilter?: BlockRangeFilter
  @observable filterCombination: FilterCombinations
  @observable sort: "asc" | "desc" = "desc"
  @observable limit: number

  constructor() {
    this.cursorCache = new CursorCache()
    this.filterSections = this.defaultFilterSections
    this.loadingTransactions = false
    this.results = []
    this.hasNextPage = false
    this.rangeOption = RangeOptions.LAST_BLOCKS
    this.filterCombination = FilterCombinations.LAST_BLOCKS
    this.limit = 25
  }

  get defaultFilterSections(): FilterSection[] {
    return [
      {
        type: FilterTypes.BLOCK_RANGE,
        data: { lastBlocks: BLOCK_NUM_5M, option: RangeOptions.LAST_BLOCKS }
      },
      {
        type: FilterTypes.BLOCK_STATUS,
        data: { irreversibleOnly: false }
      }
    ]
  }

  get defaultLastBlocksFilterSection() {
    return {
      type: FilterTypes.BLOCK_RANGE,
      data: { lastBlocks: BLOCK_NUM_5M, option: RangeOptions.LAST_BLOCKS }
    }
  }

  get defaultCustomFilterSection() {
    return {
      type: FilterTypes.BLOCK_RANGE,
      data: { min: 1, max: 2 * BLOCK_NUM_5M, option: RangeOptions.CUSTOM }
    }
  }

  saveBlockRange() {
    const filter = this.filterSection(FilterTypes.BLOCK_RANGE)
    if (filter) {
      if (filter.data.option === RangeOptions.CUSTOM) {
        this.previousBlockRangeFilter = {
          min: filter.data.min,
          max: filter.data.max,
          option: filter.data.option
        }
      } else {
        this.previousBlockRangeFilter = {
          lastBlocks: filter.data.lastBlocks,
          option: filter.data.option
        }
      }
    }
    this.updateFilterCombinations()
  }

  updateFilterCombinations() {
    // This is used to display a localized text of the current filter status
    const filterRange = this.filterSection(FilterTypes.BLOCK_RANGE)!

    if (this.sort === "desc") {
      if (filterRange.data.option === RangeOptions.ALL && this.withReversible) {
        this.filterCombination = FilterCombinations.ALL
      } else if (filterRange.data.option === RangeOptions.ALL && !this.withReversible) {
        this.filterCombination = FilterCombinations.ALL_IRREVERSIBLE
      } else if (filterRange.data.option === RangeOptions.CUSTOM && this.withReversible) {
        this.filterCombination = FilterCombinations.BLOCK_RANGE
      } else if (filterRange.data.option === RangeOptions.CUSTOM && !this.withReversible) {
        this.filterCombination = FilterCombinations.BLOCK_RANGE_IRREVERSIBLE
      } else if (filterRange.data.option === RangeOptions.LAST_BLOCKS && !this.withReversible) {
        this.filterCombination = FilterCombinations.LAST_BLOCKS_IRREVERSIBLE
      } else if (filterRange.data.option === RangeOptions.LAST_BLOCKS && this.withReversible) {
        this.filterCombination = FilterCombinations.LAST_BLOCKS
      }
    } else if (filterRange.data.option === RangeOptions.ALL && this.withReversible) {
      this.filterCombination = FilterCombinations.ALL_ASCENDING
    } else if (filterRange.data.option === RangeOptions.ALL && !this.withReversible) {
      this.filterCombination = FilterCombinations.ALL_ASCENDING_IRREVERSIBLE
    } else if (filterRange.data.option === RangeOptions.CUSTOM && this.withReversible) {
      this.filterCombination = FilterCombinations.BLOCK_RANGE_ASCENDING
    } else if (filterRange.data.option === RangeOptions.CUSTOM && !this.withReversible) {
      this.filterCombination = FilterCombinations.BLOCK_RANGE_ASCENDING_IRREVERSIBLE
    } else if (filterRange.data.option === RangeOptions.LAST_BLOCKS && !this.withReversible) {
      this.filterCombination = FilterCombinations.LAST_BLOCKS_ASCENDING_IRREVERSIBLE
    } else if (filterRange.data.option === RangeOptions.LAST_BLOCKS && this.withReversible) {
      this.filterCombination = FilterCombinations.LAST_BLOCKS_ASCENDING
    }
  }

  didRangeFilterChange(): boolean {
    const currentRange = this.filterSection(FilterTypes.BLOCK_RANGE)!.data
    const reference = this.previousBlockRangeFilter
    if (!reference) {
      return false
    }
    if (reference.option === RangeOptions.ALL && currentRange.option === RangeOptions.ALL) {
      return false
    }

    if (reference.option === RangeOptions.CUSTOM && currentRange.option === RangeOptions.ALL) {
      return true
    }

    if (reference.option === RangeOptions.LAST_BLOCKS && currentRange.option === RangeOptions.ALL) {
      return true
    }

    if (reference.option === RangeOptions.ALL && currentRange.option === RangeOptions.LAST_BLOCKS) {
      return true
    }

    if (
      reference.option === RangeOptions.LAST_BLOCKS &&
      currentRange.option === RangeOptions.CUSTOM
    ) {
      return true
    }

    if (
      reference.option === RangeOptions.CUSTOM &&
      currentRange.option === RangeOptions.LAST_BLOCKS
    ) {
      return true
    }

    if (reference.min === currentRange.min && reference.max === currentRange.max) {
      return false
    }

    return true
  }

  get blockRangeParams(): { startBlock: number; blockCount: number } {
    const filter = this.filterSection(FilterTypes.BLOCK_RANGE)

    if (filter) {
      return blockRangeToBlockParams(this.sort, filter.data)
    }

    return { startBlock: 0, blockCount: 0 }
  }

  get blockRange(): BlockRangeFilter {
    return this.filterSection(FilterTypes.BLOCK_RANGE)!.data
  }

  get withReversible(): boolean {
    const filter = this.filterSection(FilterTypes.BLOCK_STATUS)

    if (filter && filter) {
      return !filter.data.irreversibleOnly
    }

    return false
  }

  updateCursorCache(cursor: string) {
    if (this.cursorCache.currentCursor === cursor) {
      this.cursorCache.resetAll()
    } else {
      this.cursorCache.prepareNextCursor(cursor)
      this.hasNextPage = this.cursorCache.hasNextPage
    }
  }

  toggleSort() {
    this.sort = this.sort === "desc" ? "asc" : "desc"
  }

  parseField(field: string, value: string | number) {
    if (field === "min" || field === "max" || field === "lastBlocks") {
      value = value.toString().replace(/\D/g, "")
      return parseInt(value, 10)
    }
    return value
  }

  updateFilter(type: FilterTypes, field: string, value: string | number | boolean) {
    const section = this.filterSection(type)

    if (section) {
      if (field === "option") {
        if (value === RangeOptions.LAST_BLOCKS) {
          this.sort = "desc"
          section.type = this.defaultLastBlocksFilterSection.type
          section.data = Object.assign(this.defaultLastBlocksFilterSection.data)
        } else if (value === RangeOptions.CUSTOM) {
          section.type = this.defaultCustomFilterSection.type
          section.data = Object.assign(this.defaultCustomFilterSection.data)
        }
      }
      section.data[field] = value

      if (field === "option" && type === FilterTypes.BLOCK_RANGE) {
        this.rangeOption = value as RangeOptions
      }
    }
  }

  toParams(cursor?: string): SearchQueryParams {
    return {
      ...this.DEFAULT_PARAMS,
      ...this.toParamsForUrl(cursor),
      limit: this.limit
    }
  }

  updateFromUrlParams(rawParams: SearchQueryParams & LegacySearchQueryParams) {
    const newSections = this.defaultFilterSections
    const params = upgradeLegacySearchQueryParams(rawParams)

    this.query = params.q ? decodeURIComponent(params.q) : ""
    this.sort = params.sort ? params.sort : this.sort

    if (params.startBlock && params.blockCount) {
      const section = newSections.find((ref: FilterSection) => ref.type === FilterTypes.BLOCK_RANGE)
      if (section) {
        section.data = blockParamsToBlockRange(this.sort, params.startBlock, params.blockCount)
        this.rangeOption = section.data.option
      }
    } else {
      this.rangeOption = RangeOptions.LAST_BLOCKS
    }

    const statusSection = newSections.find(
      (ref: FilterSection) => ref.type === FilterTypes.BLOCK_STATUS
    )
    if (statusSection) {
      if (params.withReversible !== undefined) {
        statusSection.data.irreversibleOnly = !params.withReversible
      } else {
        statusSection.data.irreversibleOnly = false
      }
    }

    this.filterSections = newSections
    this.updateFilterCombinations()
  }

  cursoredUrl(cursor: string): string {
    let queryParams = {}
    if (this.query && this.query.length > 0) {
      queryParams = this.toParamsForUrl(cursor)
    }

    return `${Links.viewTransactionSearch()}?${stringify(queryParams)}`
  }

  private filterSection(type: FilterTypes) {
    return this.filterSections.find((ref: FilterSection) => ref.type === type)
  }

  private toParamsForUrl(cursor?: string): SearchQueryParams {
    return {
      q: this.query,
      startBlock: this.blockRangeParams.startBlock,
      blockCount: this.blockRangeParams.blockCount,
      withReversible: this.withReversible,
      sort: this.sort,
      cursor
    }
  }
}
