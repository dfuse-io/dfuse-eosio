export interface Post {
  account: string
  uuid: string
  title: string
  content: string
  replyToAccount: string
  replyToAccountUUID: string
  certify: boolean
  jsonMetadata: string
}

export interface Posts extends Array<Post> {
}

export interface Vote {

  voter: string
  proposition: string
  propositionHash: string
  voteValue: string
}

export interface Votes extends Array<Vote> {
}

export interface Proposition {
  title: string
  hash: string
  votes: Votes
}

export interface Propositions extends Array<Proposition> {
}
