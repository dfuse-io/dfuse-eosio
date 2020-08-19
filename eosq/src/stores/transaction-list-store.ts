import { observable } from "mobx"
import { BlockRangeFilter, FilterCombinations, RangeOptions } from "../models/search-filters"
import { CursorCache } from "../services/cursor-store"
import { BLOCK_NUM_100B } from "../models/block"
import { TransactionLifecycle, ErrorData } from "@dfuse/client"

export class TransactionListStore {
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
  @observable loadingTransactions: boolean
  @observable searchError?: ErrorData
  @observable results: TransactionLifecycle[]
  @observable previousBlockRangeFilter?: BlockRangeFilter
  @observable filterCombination: FilterCombinations
  @observable sort: "asc" | "desc" = "desc"
  @observable limit: number

  constructor() {
    this.cursorCache = new CursorCache()
    this.loadingTransactions = false
    this.results = []
    this.hasNextPage = false
    this.rangeOption = RangeOptions.LAST_BLOCKS
    this.filterCombination = FilterCombinations.LAST_BLOCKS
    this.limit = 25
  }

  updateCursorCache(cursor: string) {
    if (this.cursorCache.currentCursor === cursor) {
      this.cursorCache.resetAll()
    } else {
      this.cursorCache.prepareNextCursor(cursor)
      this.hasNextPage = this.cursorCache.hasNextPage
    }
  }
}
