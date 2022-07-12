import { Account } from "../models/account"

export function getAccountMock(): Account {
  return {
    creator: {
      creator: "test",
      created: "",
      block_id: "abc",
      block_num: 123,
      block_time: "block_time",
      trx_id: "abc"
    },
    account_name: "eoscanadacom",
    last_code_update: new Date(),
    core_liquid_balance: "16.000 EOS",
    linked_permissions: [],
    created: new Date(),
    permissions: [],
    privileged: false,
    ram_quota: 100,
    ram_usage: 10,
    net_limit: {
      used: 12,
      available: 23,
      max: 34
    },
    cpu_limit: {
      used: 12,
      available: 23,
      max: 34
    },
    self_delegated_bandwidth: {
      from: "from",
      to: "to",
//ultra-andrey-bezrukov --- BLOCK-80 Integrate ultra power into dfuse and remove rex related tables
//      net_weight: "2.2000 EOS",
//      cpu_weight: "1.3000 EOS"
      power_weight: "3.5000 EOS"
    },
//    cpu_weight: 12,
//    net_weight: 13,
    power_weight: 25,
    total_resources: {
//      net_weight: "4.2000 EOS",
//      cpu_weight: "5.3000 EOS",
      power_weight: "9.5000 EOS",
      owner: "eoscanadacom",
      ram_bytes: 123
    },
    voter_info: {
      owner: "eoscanadacom",
      proxy: "",
      producers: [],
      staked: 12,
      last_vote_weight: 12,
      proxied_vote_weight: 12,
      is_proxy: false,
      deferred_trx_id: 12,
      last_unstake_time: new Date(),
      unstaking: "unstaking"
    }
  } as Account
}
