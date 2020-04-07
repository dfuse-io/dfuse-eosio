export interface Vote {
  producer: string
  votePercent: number
  decayedVote: number
  website: string
}

export interface Votes extends Array<Vote> {}

export interface VotedProducer {
  owner: string
  total_votes: number
  producer_key: string
  is_active: number
  url: string
  unpaid_blocks: number
  last_claim_time: number
  location: number
}

export interface VotedProducers extends Array<VotedProducer> {}

export interface VoteTally {
  total_activated_stake: number
  total_votes: number
  decay_weight: number
  producers?: VotedProducer[]
}
