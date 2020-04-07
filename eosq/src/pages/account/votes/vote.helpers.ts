import moment from "moment"
import { Account } from "../../../models/account"

const BLOCK_EPOCH = 946684800
const SECONDS_PER_WEEK = 3600 * 24 * 7

export function effectivePersonalVoteWeight(account: Account) {
  if (account.voter_info.is_proxy) {
    return account.voter_info.last_vote_weight - account.voter_info.proxied_vote_weight
  }

  return account.voter_info.last_vote_weight
}

export function calculateVoteStrength(account: Account, staked: number): number {
  const timeDifference = parseInt(moment.utc().format("X"), 10) - BLOCK_EPOCH
  const weight = Math.floor(timeDifference / SECONDS_PER_WEEK) / 52.0
  return effectivePersonalVoteWeight(account) / (weight ** 2 * staked)
}
