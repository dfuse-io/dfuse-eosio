export default {
  account: {
    transactions: {
      title: "Recent Transactions",
      subTitle: " (Up to head block)"
    },
    tokens: {
      title: "OTHER TOKENS",
      table: {
        token: "Token",
        quantity: "Balance",
        contract: "Account"
      }
    },
    social_links: {
      verified_by: "Verified by EOS Canada"
    },
    status: {
      used: "used"
    },
    summary: {
      block_producer: "Block Producer",
      creation_date: "Creation Date",
      created_by: "Created by",
      creation_trx_id: "Created in tx",
      owner: "Owner:",
      website: "Website",
      email: "Email",
      location: "Location",
      staked_by: "Staked by",
      self: "Self",
      tooltip: {
        other: "Others",
//ultra-andrey-bezrukov --- BLOCK-80 Integrate ultra power into dfuse and remove rex related tables
//        networkTitle: "Total",
//        cpuTitle: "Total",
        powerTitle: "Total",
      },
      verified_website: "Verified website (owned by account)",
      voter_info: {
        noVotes: "This account is currently not voting for any Block Producers.",
        title: "Block Producer Votes",
        labels: {
          latest_vote: "LATEST VOTE: ",
          strength: "Decayed: ",
          vote_for: "CURRENTLY VOTING FOR:",
          nextDecay: "next decay Saturday 00:00 UTC",
          vote_weight: "VOTE WEIGHT",
          decayed_vote_weight: "DECAYED VOTE WEIGHT",
          vote_for_producers: "Votes cast for block producer(s):",
          vote_for_proxy: "Block producer(s) voted by proxy:"
        }
      }
    },
    permissions: {
      title: "PERMISSIONS",
      labels: {
        weight: "Weight:",
        account: "Account:",
        wait: "Wait:",
        seconds: "seconds",
        key: "Key:",
        name: "Permission Name:",
        parent_permission: "Parent Permission",
        threshold: "Authorization, threshold:"
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
        title: "Account created by MYKEY"
      }
    },
    pie_chart: {
      legendTitle: "TOTAL BALANCE",
      labels: {
//        staked_cpu: "STAKED FOR CPU",
//        staked_network: "STAKED FOR NETWORK",
        staked_power: "STAKED FOR POWER",
        delegated_cpu: "DELEGATED FOR CPU",
        delegated_network: "DELEGATED FOR NETWORK",
        pending_refund: "PENDING REFUND",
        available_funds: "AVAILABLE FUNDS",
//       rex: "REX",
//       rex_funds: "REX FUNDS"
      }
    },
    banner: {
      labels: {
        transactions: "transactions",
        votes_staked: "votes staked",
        transactions_value: "transactions value"
      }
    },
    status_bar: {
      units: {
        kb: "Kb",
        mb: "Mb",
        seconds: "s"
      },
      titles: {
        available: "available",
        memory: "RAM",
        cpu_bandwidth: "POWER@CPU",
        network_bandwidth: "POWER@NETWORK",
        power_bandwidth: "POWER"
      }
    },
    tabs: {
      vote_title: "Votes",
      transactions: "Transactions",
      tables: "Tables"
    },
    loading: "Loading account",
    tables: {
      formatted: "Formatted"
    }
  }
}
