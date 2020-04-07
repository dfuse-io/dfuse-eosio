export default {
  filters: {
    title: "筛选条件",
    queryParams: "查询参数",
    apply: "确定",
    sections: {
      titles: {
        blockRange: "区块区间",
        blockStatus: "区块状态"
      },
      labels: {
        from: "从",
        to: "到",
        all: "全部",
        lastBlocks: "查询包含的区块数量",
        irreversible: "仅查询不可逆"
      }
    },
    currentFilter: {
      last_blocks: "搜索之前的 {{lastBlocks}} 个区块",
      last_blocks_irreversible: "搜索之前的 {{lastBlocks}} 个区块，并仅限不可逆区块",
      last_blocks_ascending_irreversible:
        "搜索之前的 {{lastBlocks}} 个区块，并仅限不可逆区块，升序排列",
      last_blocks_ascending: "搜索之前的 {{lastBlocks}} 个区块，升序排列",
      all: "搜索全部历史",
      all_irreversible: "搜索全部历史，但仅包含不可逆区块",
      block_range: "搜索区块区间：{{min}} 至 {{max}}",
      block_range_irreversible: "搜索区块区间：{{min}} 至 {{max}}，但仅包含不可逆区块"
    },
    rangeOptions: {
      lastBlocks: "最新区块",
      all: "全部",
      custom: "自定义"
    }
  }
}
