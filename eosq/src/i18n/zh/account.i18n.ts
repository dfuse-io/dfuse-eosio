export default {
  account: {
    transactions: {
      title: "近期交易",
      subTitle: "(截止到当前区块）"
    },
    tokens: {
      title: "其它代币",
      table: {
        token: "代币",
        quantity: "余额",
        contract: "账户"
      }
    },
    social_links: {
      verified_by: "由 EOS Canada 验证"
    },
    status: {
      used: "已用"
    },
    summary: {
      block_producer: "BP节点",
      creation_date: "创建日期",
      created_by: "创建者",
      creation_trx_id: "创建交易",
      owner: "所有者：",
      website: "网站",
      email: "电子邮箱",
      location: "地点",
      staked_by: "抵押者",
      self: "账户本身",
      tooltip: {
        other: "其他",
        networkTitle: "网络带宽总计",
        cpuTitle: "CPU带宽总计"
      },
      voter_info: {
        noVotes: "此帐户目前没有对任何BP节点投票。",
        title: "BP节点投票",
        labels: {
          latest_vote: "最近投票：",
          strength: "衰退后强度：",
          vote_for: "目前投票：",
          nextDecay: "下次衰退：下周六 00:00 UTC",
          vote_weight: "投票权重",
          decayed_vote_weight: "投票衰退强度",
          vote_for_producers: "为BP节点投的票：",
          vote_for_proxy: "通过代理为BP节点投的票："
        }
      }
    },
    permissions: {
      title: "权限",
      labels: {
        weight: "权重：",
        account: "账户：",
        wait: "等待：",
        seconds: "秒",
        key: "密钥：",
        name: "权限名称：",
        parent_permission: "母权限",
        threshold: "授权，阈值："
      }
    },
    badges: {
      gn: "Gn",
      px: "Px",
      pv: "Pv",
      co: "Co",
      bp: "Bp",
      my: {
        name: "My",
        title: "账户通过 MYKEY 创建"
      }
    },
    pie_chart: {
      legendTitle: "总余额",
      labels: {
        staked_cpu: "CPU抵押数量",
        staked_network: "网络带宽抵押数量",
        delegated_cpu: "CPU委托数量",
        delegated_network: "网络带宽委托数量",
        pending_refund: "待退款",
        available_funds: "可用资金",
        rex: "REX",
        rex_funds: "REX FUNDS"
      }
    },
    banner: {
      labels: {
        transactions: "交易",
        votes_staked: "投票抵押",
        transactions_value: "交易价值"
      }
    },
    status_bar: {
      units: {
        kb: "Kb",
        mb: "Mb",
        seconds: "s"
      },
      titles: {
        available: "目前可用",
        memory: "RAM",
        cpu_bandwidth: "CPU",
        network_bandwidth: "网络带宽"
      }
    },
    tabs: {
      vote_title: "投票",
      transactions: "交易",
      tables: "表格"
    },
    loading: "读取账户中",
    tables: {
      formatted: "表格化"
    }
  }
}
