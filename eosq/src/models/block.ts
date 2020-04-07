export const BLOCK_NUM_100B = 100000000000
export const BLOCK_NUM_5M = 5000000

export interface BlockSummary {
  id: string
  irreversible: boolean
  header: BlockHeader
  block_num: number
  dpos_lib_num?: number
  transaction_count: number
  sibling_blocks?: BlockSummary[]
  active_schedule: ProducerSchedule
}

export interface BlockHeader {
  timestamp: string
  producer: string
  confirmed: number
  previous: string
  transaction_mroot: string
  action_mroot: string
  schedule_version: string
  new_producers?: ProducerSchedule
}

export interface ProducerSchedule {
  version: number
  producers: ProducerKey[]
}

export interface ProducerKey {
  producer_name: string
  block_signing_key: string
}
