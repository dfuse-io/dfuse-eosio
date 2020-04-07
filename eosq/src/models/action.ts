export interface EosioNewAccount {
  active: Permission
  creator: string
  name: string
  owner: Permission
  payer: string
  quant: string
  receiver: string
  from: string
  stake_cpu_quantity: string
  stake_net_quantity: string
  transfer?: number
}

export interface EosioTokenIssueData {
  to: string
  quantity: string
  memo: string
}

export interface EosioTokenTransferData {
  from: string
  to: string
  quantity: string
  memo: string
}

export interface Permission {
  accounts: any[]
  keys: Key[]
  threshold: number
  waits: any[]
}

export interface Key {
  key: string
  weight: number
}
