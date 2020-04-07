import { observable } from "mobx"
import { MarketPrice } from "../models/marketprice"
import { HeadInfoData } from "@dfuse/client"

export class MetricsStore {
  @observable priceVariation: number
  @observable priceUSD: number

  @observable headBlockId: string
  @observable headBlockNum: number
  @observable headBlockProducer: string

  @observable lastIrreversibleBlockNum: number
  @observable lastIrreversibleBlockId: string

  constructor() {
    this.priceUSD = -1
    this.priceVariation = 0

    this.headBlockId = ""
    this.headBlockNum = 0
    this.headBlockProducer = ""

    this.lastIrreversibleBlockNum = 0
    this.lastIrreversibleBlockId = ""
  }

  setBlockHeight(data: HeadInfoData) {
    if (data.head_block_num > this.headBlockNum) {
      this.headBlockId = data.head_block_id
      this.headBlockNum = data.head_block_num
    }

    if (data.head_block_producer !== this.headBlockProducer) {
      this.headBlockProducer = data.head_block_producer
    }

    if (data.last_irreversible_block_num > this.lastIrreversibleBlockNum) {
      this.lastIrreversibleBlockNum = data.last_irreversible_block_num
      this.lastIrreversibleBlockId = data.last_irreversible_block_id
    }
  }

  setPrice(price: MarketPrice) {
    if (price.price !== this.priceUSD) {
      this.priceUSD = price.price
      this.priceVariation = price.variation
    }
  }
}
