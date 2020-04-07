export default {
  filters: {
    title: "FILTER",
    queryParams: "Query Parameters",
    apply: "APPLY",
    sections: {
      titles: {
        blockRange: "BLOCK RANGE",
        blockStatus: "BLOCK STATUS"
      },
      labels: {
        from: "From",
        to: "To",
        all: "ALL",
        lastBlocks: "Number of blocks",
        irreversible: "Irreversible Blocks Only"
      }
    },
    currentFilter: {
      last_blocks: "Searching the last {{lastBlocks}} blocks",
      last_blocks_irreversible:
        "Searching the last {{lastBlocks}} blocks, irreversible blocks only",
      last_blocks_ascending_irreversible:
        "Searching the last {{lastBlocks}} blocks, irreversible blocks only, ascending order",
      last_blocks_ascending: "Searching the last {{lastBlocks}} blocks, ascending order",

      all: "Filter Search Results",
      all_irreversible: "Searching all history, irreversible blocks only",
      block_range: "Searching from block {{min}} to {{max}}",
      block_range_irreversible: "Searching from block {{min}} to {{max}}, irreversible blocks only",

      all_ascending: "Searching all history, ascending order",
      all_ascending_irreversible:
        "Searching all history, irreversible blocks only, ascending order",
      block_range_ascending: "Searching from block {{min}} to {{max}}, ascending order",
      block_range_ascending_irreversible:
        "Searching from block {{min}} to {{max}}, irreversible blocks only, ascending order"
    },
    rangeOptions: {
      lastBlocks: "RECENT BLOCKS",
      all: "ALL",
      custom: "CUSTOM"
    }
  }
}
