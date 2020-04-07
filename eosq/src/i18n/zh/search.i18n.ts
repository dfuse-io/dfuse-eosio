export default {
  search: {
    placeholder: "搜索账户、区块、交易、时间戳…",
    result: {
      noResultFoundFor: "找不到结果",
      blockFound: "区块已找到但是没有被处理，请稍后再查询！",
      error: "搜索时服务器上发生错误",
      nothingFound: "什么都没找到",
      searchQuery: "搜索查询",
      unregisteredLabel: "未注册的帐户",
      unregisteredValue: "你可以以后再申请",
      errors: {
        label: "Error:",
        request_validation_error: "your query was malformed",
        generic_error: "The search failed"
      }
    },
    suggestions: {
      summary: {
        account_history: "{{accountName}}的账户历史",
        signed_by: "{{accountName}}所签名的交易",
        eos_token_transfer: "{{accountName}}的EOS转账",
        fuzzy_token_search: "{{accountName}}泛代币查询"
      }
    },
    syntax: "句法:",
    irreversibleOnly: "仅包含不可逆区块",
    searchResultsFor: "搜索结果",
    sqeDocumentation: "SQE 语言文档",
    loading: "读取中……",
    errorFetch: "无相关搜索建议"
  },
  transactionSearch: {
    buttons: {
      signedBy: "SIGNED BY {{accountName}}",
      notifications: "NOTIFICATIONS"
    },
    search: "SEARCH",
    title: "搜索交易",
    results: {
      title: "结果：",
      subTitle: "仅包括不可逆状态的区块"
    },
    buttonLabels: {
      account: "账户",
      tokens: "代币",
      system: "系统操作"
    },
    dropdowns: {
      tokens: {
        allTokens: "所有代币",
        eos: "EOS",
        popularTokens: "人气代币"
      },
      system: {
        claimRewards: "认领奖励",
        delegateBandwidth: "委派带宽",
        undelegateBandwidth: "取消带宽委派",
        regProducer: "REG PRODUCER",
        setCode: "SET CODE"
      }
    }
  }
}
