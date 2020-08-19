import { FilterCombinations, FilterTypes, RangeOptions } from "../../models/search-filters"
import { BLOCK_NUM_5M } from "../../models/block"
import { SearchStore } from "../search-store"
import { SearchQueryParams, LegacySearchQueryParams } from "../../models/search"

const expectedParams: SearchQueryParams = {
  blockCount: 2000000,
  cursor: "cursor",
  limit: 25,
  q: "query",
  sort: "desc",
  startBlock: 32000000,
  withReversible: true
}

const expectedLegacyParams: SearchQueryParams & LegacySearchQueryParams = {
  block_count: 2000000,
  cursor: "cursor",
  limit: 25,
  q: "query",
  sort: "desc",
  start_block: 32000000,
  with_reversible: true
}

describe("SearchStore", () => {
  describe("defaultFilterSections", () => {
    it("should return the current defaults", () => {
      const searchStore = new SearchStore()
      expect(searchStore.defaultFilterSections).toEqual([
        {
          type: FilterTypes.BLOCK_RANGE,
          data: { lastBlocks: BLOCK_NUM_5M, option: RangeOptions.LAST_BLOCKS }
        },
        {
          type: FilterTypes.BLOCK_STATUS,
          data: { irreversibleOnly: false }
        }
      ])
    })
  })

  describe("saveBlockRange", () => {
    it("should save the current block range in the attribute previousBlockRangeFilter", () => {
      const min = 20000000
      const max = 30000000
      const searchStore = new SearchStore()
      searchStore.updateFilter(FilterTypes.BLOCK_RANGE, "option", RangeOptions.CUSTOM)
      searchStore.updateFilter(FilterTypes.BLOCK_RANGE, "min", min)
      searchStore.updateFilter(FilterTypes.BLOCK_RANGE, "max", max)

      searchStore.saveBlockRange()
      expect(searchStore.blockRange).toEqual({ min, max, option: RangeOptions.CUSTOM })
    })
  })

  describe("updateFilterCombinations", () => {
    it("should set the filter combination to BLOCK_RANGE", () => {
      const min = 20000000
      const max = 30000000
      const searchStore = new SearchStore()
      searchStore.updateFilter(FilterTypes.BLOCK_RANGE, "option", RangeOptions.CUSTOM)
      searchStore.updateFilter(FilterTypes.BLOCK_RANGE, "min", min)
      searchStore.updateFilter(FilterTypes.BLOCK_RANGE, "max", max)
      searchStore.updateFilterCombinations()
      expect(searchStore.filterCombination).toEqual(FilterCombinations.BLOCK_RANGE)
    })
  })

  describe("updateFilterCombinations", () => {
    it("should set the filter combination to LAST_BLOCKS_IRREVERSIBLE AND BLOCK_RANGE_IRREVERSIBLE", () => {
      const min = 20000000
      const max = 30000000
      const searchStore = new SearchStore()
      expect(searchStore.filterCombination).toEqual(FilterCombinations.LAST_BLOCKS)
      searchStore.updateFilter(FilterTypes.BLOCK_STATUS, "irreversibleOnly", true)
      searchStore.updateFilterCombinations()

      expect(searchStore.filterCombination).toEqual(FilterCombinations.LAST_BLOCKS_IRREVERSIBLE)

      searchStore.updateFilter(FilterTypes.BLOCK_RANGE, "option", RangeOptions.CUSTOM)
      searchStore.updateFilter(FilterTypes.BLOCK_RANGE, "min", min)
      searchStore.updateFilter(FilterTypes.BLOCK_RANGE, "max", max)
      searchStore.updateFilterCombinations()
      expect(searchStore.filterCombination).toEqual(FilterCombinations.BLOCK_RANGE_IRREVERSIBLE)
    })
  })

  describe("blockRangeParams", () => {
    it("should return a valid block range (0, 0) is default", () => {
      const searchStore = new SearchStore()
      expect(searchStore.blockRangeParams).toEqual({ startBlock: 0, blockCount: BLOCK_NUM_5M })
    })
  })

  describe("withReversible", () => {
    it("should return true by default", () => {
      const searchStore = new SearchStore()
      expect(searchStore.withReversible).toEqual(true)
    })
  })

  describe("updateFilter", () => {
    it("should update a the block status filter", () => {
      const searchStore = new SearchStore()
      expect(searchStore.withReversible).toEqual(true)
      searchStore.updateFilter(FilterTypes.BLOCK_STATUS, "irreversibleOnly", true)
      expect(searchStore.withReversible).toEqual(false)
    })
  })

  describe("updateFromUrlParams", () => {
    it("should convert url params into filters", () => {
      const searchStore = new SearchStore()
      searchStore.updateFromUrlParams(expectedParams)
      expect(searchStore.query).toEqual("query")
      expect(searchStore.blockRangeParams).toEqual({ blockCount: 2000000, startBlock: 32000000 })
      expect(searchStore.withReversible).toBe(true)
    })

    it("should convert legacy url params into filters", () => {
      const searchStore = new SearchStore()
      searchStore.updateFromUrlParams(expectedLegacyParams)
      expect(searchStore.query).toEqual("query")
      expect(searchStore.blockRangeParams).toEqual({ blockCount: 2000000, startBlock: 32000000 })
      expect(searchStore.withReversible).toBe(true)
    })
  })
})
