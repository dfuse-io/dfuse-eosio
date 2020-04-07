import { observable } from "mobx"
import { VotedProducer, Votes, VoteTally } from "../models/vote"

export class VoteStore {
  @observable votesCast: number
  @observable votes: Votes

  constructor() {
    this.votesCast = -1
    this.votes = []
  }

  update(voteTally: VoteTally) {
    this.votesCast = voteTally.total_activated_stake / 10000

    this.votes = (voteTally.producers || []).map((producer: VotedProducer) => {
      return {
        producer: producer.owner,
        votePercent:
          voteTally.total_votes > 0 ? (producer.total_votes / voteTally.total_votes) * 100 : 0,
        decayedVote: producer.total_votes / voteTally.decay_weight / 10000.0,
        website: producer.url
      }
    })
  }
}
