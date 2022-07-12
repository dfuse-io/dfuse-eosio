export interface Account {
  creator?: AccountCreator
  account_name: string
  last_code_update: Date
  core_liquid_balance: string
  linked_permissions?: LinkedPermission[]
  account_verifications?: AccountVerifications
  created: Date
  permissions: Permission[]
  privileged: boolean
  ram_quota: number
  ram_usage: number
  net_limit: AccountResourceLimit
  cpu_limit: AccountResourceLimit
  self_delegated_bandwidth: SelfDelegatedBandwidth
  refund_request?: RefundRequest
  cpu_weight?: number
//  net_weight: number
  power_weight: number
  total_resources: AccountTotalResources
  voter_info: VoterInfo
  block_producer_info?: BlockProducerInfo
//ultra-andrey-bezrukov --- BLOCK-80 Integrate ultra power into dfuse and remove rex related tables
//  rex_balance?: RexBalance
//  rex_funds?: RexFunds
//  net_loans?: number
//  cpu_loans?: number
  power_loans?: number
}

export interface LinkedPermission {
  action: string
  permission_name: string
  contract: string
}

export interface Verifiable {
  handle: string
  claim: string
  verified: boolean
  last_check: string
}

export interface AccountVerifications {
  email?: Verifiable
  website?: Verifiable
  twitter?: Verifiable
  github?: Verifiable
  telegram?: Verifiable
  facebook?: Verifiable
  reddit?: Verifiable
}

export interface AccountCreator {
  created: string
  creator: string
  block_id: string
  block_num: number
  block_time: string
  trx_id: string
}

//ultra-andrey-bezrukov --- BLOCK-80 Integrate ultra power into dfuse and remove rex related tables
//export interface RexBalance {
//  matured_rex: number
//  owner: string
//  rex_balance: string
//  rex_maturities: { first: string; second: string }[]
//  version: number
//  vote_stake: string
//}
//
//export interface RexFunds {
//  version: number
//  owner: string
//  balance: string
//}
//
//export interface RexLoan {
//  balance: string
//  from: string
//  loan_num: number
//  payment: string
//  receiver: string
//  total_staked: string
//  version: 0
//}

export interface BlockProducerInfo {
  org: {
    email: string
    branding: BrandingLogos
    candidate_name: string
    location: Location
    social: Record<string, string>
    website: string
    ownership_disclosure: string
  }
  producer_account_name: string
}

export interface Location {
  country: string
  latitude: number
  longitude: number
  name: number
}

export interface BrandingLogos {
  logo_256: string
  logo_1024: string
  logo_svg: string
}

export interface AccountTotalResources {
//  cpu_weight: string
//  net_weight: string
  power_weight: string
  owner: string
  ram_bytes: number
}

export interface RefundRequest {
  owner: string
  request_time: Date
//  net_amount: string
//  cpu_amount: string
  power_amount: string
}

export interface SelfDelegatedBandwidth {
  from: string
  to: string
//  net_weight: string
//  cpu_weight: string
  power_weight: string
}

export interface AccountResourceLimit {
  used: number
  available: number
  max: number
}

export interface Permission {
  perm_name: string
  parent: string
  required_auth: Authority
}

export interface Authority {
  threshold: number
  keys: KeyWeight[]
  accounts: PermissionLevelWeight[]
  waits: WaitWeight[]
}

export interface PermissionLevelWeight {
  permission: PermissionLevel
  weight: number
}

export interface PermissionLevel {
  actor: string
  permission: string
}

export interface KeyWeight {
  key: string
  weight: number
}

export interface WaitWeight {
  wait_sec: number
  weight: number
}

export interface VoterInfo {
  owner: string
  proxy: string
  producers: string[]
  staked: number
  last_vote_weight: number
  proxied_vote_weight: number
  is_proxy: boolean
  deferred_trx_id: number
  last_unstake_time: Date
  unstaking: string
}

export interface DelegatedBandwidth {
  from: string
  to: string
//ultra-andrey-bezrukov --- BLOCK-80 Integrate ultra power into dfuse and remove rex related tables
//  net_weight: string
//  cpu_weight: string
  power_weight: string
}
