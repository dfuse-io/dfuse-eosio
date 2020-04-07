export default {
  transaction: {
    displayedTree: {
      creationTree: "按创建顺序",
      executionTree: "按执行顺序"
    },
    showMoreActions: {
      title: "+ {{extraActions}} 操作",
      longTitle: "+ {{extraActions}} 操作 / + {{extraDeferred}} 延迟操作"
    },
    tableops: {
      operations: {
        INS: "创建 TABLE",
        REM: "移除 TABLE"
      },
      label: "<0>{{operation}}</0> | table:<1>{{table}}</1> | scope:<2>{{scope}}</2>"
    },
    dbops: {
      operations: {
        INS: "创建 ROW",
        UPD: "更新 ROW",
        REM: "移除 ROW"
      },
      label:
        "<0>{{operation}}</0> | table:<1>{{table}}</1> | scope:<2>{{scope}}</2> | primary key: <3>{{primaryKey}}</3>:"
    },
    ramUsage: {
      operations: {
        create_table: "创建 table",
        deferred_trx_add: "储存延迟交易",
        deferred_trx_cancel: "取消延迟交易",
        deferred_trx_pushed: "创建延迟交易",
        deferred_trx_removed: "执行延迟交易",
        deleteauth: "删除授权",
        linkauth: "连接授权",
        newaccount: "创建新账户",
        primary_index_add: "存储 row (primary)",
        primary_index_remove: "移除 row (primary)",
        primary_index_update: "更新 row (primary)",
        primary_index_update_add_new_payer: "存储 payer (primary)",
        primary_index_update_remove_old_payer: "移除 payer (primary)",
        remove_table: "移除一个 table",
        secondary_index_add: "存储 row (secondary)",
        secondary_index_remove: "移除 row (secondary)",
        secondary_index_update_add_new_payer: "存储 payer (secondary)",
        secondary_index_update_remove_old_payer: "移除 payer (secondary)",
        setabi: "为账户更新ABI",
        setcode: "为账户更新合约",
        unlinkauth: "取消连接授权",
        updateauth_create: "创建新权限",
        updateauth_update: "更新权限"
      },
      title: "Ram用量",
      consumed: "<0>{{accountName}}</0> 消耗了 <1>{{bytes}}</1> (现有 <2>{{totalBytes}}</2>)",
      released: "<0>{{accountName}}</0> 释放了 <1>{{bytes}}</1> (现有 <2>{{totalBytes}}</2>)",
      consumedDetail: "<0>{{accountName}}</0> 消耗了<1>{{bytes}}</1> — <2>{{operation}}</2>",
      releasedDetail: "<0>{{accountName}}</0> 释放了<1>{{bytes}}</1> — <2>{{operation}}</2>"
    },
    deferred: {
      delayedFor: "延迟",
      create: "<0>已创建</0> 延迟交易 <1>{{transactionId}}</1> 延迟 <2>{{delay}}</2>",
      cancel: "<0>已取消</0> 延迟交易 <1>{{transactionId}}</1>",

      triggeredBy: {
        label: "触发者",
        content: "<0>{{transactionId}}</0>  区块号：<1>{{blockNum}}</1>"
      },
      creationMethod: {
        label: "创建方式",
        PUSH_CREATE: "直接推送到链上，附有延迟",
        CREATE: "由智能合约创建",
        MODIFY_CREATE: "由智能合约更改"
      },

      createdBy: {
        label: "创建者为",
        content: "区块 <1>{{blockNum}}</1> 中的 <0>{{transactionId}}</0>"
      },
      canceledBy: {
        label: "取消者为",
        content: "区块 <1>{{blockNum}}</1> 中的 <0>{{transactionId}}</0>"
      }
    },
    status: {
      hard_fail: "失败 (hard fail)",
      soft_fail: "失败 (soft fail)",
      delayed: "已延迟",
      canceled: "已取消",
      executed: "已执行",
      expired: "已过期",
      pending: "等待"
    },
    pill: {
      dbOps: "DB/数据库行为",
      ramOps: "RAM操作",
      general: "常规数据",
      jsonData: "JSON 格式数据",
      hexData: "HEX 格式数据",
      cpu_usage: "CPU用量",
      total_cpu_usage: "CPU用量总计",
      receiver: "接收者",
      memo: "备注：",
      account: "合约账户",
      action_name: "操作名称",
      authorization: "授权权限"
    },
    loading: "交易读取中",
    traces: {
      title: "操作",
      empty: "无行为",
      raw: "原始",
      memo: "备注",
      pill: {
        names: {
          buyram: "购买Ram"
        }
      }
    },
    notSeenYet: {
      notFound: "交易未找到",
      watchingForNetwork: "监视进入到网络中的交易"
    },
    banner: {
      transaction_count: "交易计数",
      block_produced: "产出区块",
      total_value: "总值",
      title: "交易"
    },
    blockPanel: {
      title: "区块",
      block: "区块号",
      blockId: "区块ID",
      age: "寿命",
      status: "状态",
      producer: "出块节点",

      statuses: {
        notSeenYet: "还未看到",
        waiting: "等待",
        confidence: "置信度",
        irreversible: "已执行"
      }
    },
    detailPanel: {
      producer: {
        unknown: "未知"
      },
      fullTrace: "完整痕迹",
      title: "交易",
      hash: "哈希",
      status: "状态",
      expirationDate: "失效日期",
      cpuUsage: "CPU用量",
      networkUsage: "网络用量",
      authorizations: "授权权限",
      signedBy: "签属者",
      noUsage: "无",
      statuses: {
        executed: "已执行，",
        expired: "已过期",
        accepted: "等待不可逆",
        irreversible: "不可逆",
        blockDeep: "区块深度"
      }
    },
    progressCircle: {
      confidence: "置信度"
    },
    list: {
      extendSearch: "继续搜索",
      advancedOptions: "高级选项",
      noResultsExtend: "前 {{lastBlocks}} 区块中不包含查询结果",
      noMoreResultsExtend: "前 {{lastBlocks}} 区块中没有更多的查询结果",
      empty: "无交易",
      loading: "交易读取中……",
      title: "交易",
      header: {
        timestamp: "时间戳",
        id: "交易ID",
        blockId: "区块ID",
        timeCreated: "日期 - 时间",
        expiration: "过期时间",
        blockTime: "时间戳",
        account: "账户",
        contract: "合约",
        action: "操作",
        value: "值",
        summary: "交易/ 区块",
        moreActions: "更多操作"
      }
    }
  }
}
