export interface FilterSection {
  type: FilterTypes
  data: any
}

export enum FilterTypes {
  BLOCK_STATUS = "blockStatus",
  BLOCK_RANGE = "blockRange"
}

export enum RangeOptions {
  ALL = "all",
  CUSTOM = "custom",
  LAST_BLOCKS = "lastBlocks"
}

export interface BlockRangeFilter {
  min?: number
  max?: number
  lastBlocks?: number
  option: RangeOptions
}

export enum FilterCombinations {
  ALL = "all",
  ALL_IRREVERSIBLE = "all_irreversible",
  BLOCK_RANGE = "block_range",
  BLOCK_RANGE_IRREVERSIBLE = "block_range_irreversible",

  ALL_ASCENDING = "all_ascending",
  ALL_ASCENDING_IRREVERSIBLE = "all_ascending_irreversible",
  BLOCK_RANGE_ASCENDING = "block_range_ascending",
  BLOCK_RANGE_ASCENDING_IRREVERSIBLE = "block_range_ascending_irreversible",

  LAST_BLOCKS = "last_blocks",
  LAST_BLOCKS_IRREVERSIBLE = "last_blocks_irreversible",

  LAST_BLOCKS_ASCENDING = "last_blocks_ascending",
  LAST_BLOCKS_ASCENDING_IRREVERSIBLE = "last_blocks_irreversible"
}
