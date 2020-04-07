import { observable, ObservableMap } from "mobx"
import { BlockSummary } from "../models/block"

const MAX_LIVE_BLOCKS = 500
const MAX_SNAPSHOT_BLOCKS = 5

export class BlockStore {
  liveBlocks = observable.map<string, BlockSummary>()

  /**
   * The list of block currently displayed in the dashboard.
   * First accumulated until reaching MAX_SNAPSHOT_BLOCKS then
   * refreshed only when requested.
   */
  snapshotBlocks = observable.map<string, BlockSummary>()

  /**
   * The amount of new blocks accumulated since last snapshot
   * of blocks taken.
   */
  @observable unseenBlockCount = 0

  @observable searchResult: BlockSummary | null = null

  addIncomingBlock(block: BlockSummary) {
    // TODO: update the LIB in the store..
    if (this.snapshotBlocks.size < MAX_SNAPSHOT_BLOCKS) {
      setBlockInMap(this.snapshotBlocks, block)
    } else {
      this.unseenBlockCount += 1
    }

    if (this.liveBlocks.size >= MAX_LIVE_BLOCKS) {
      this.liveBlocks.delete(this.liveBlocks.keys().next().value)
    }

    setBlockInMap(this.liveBlocks, block)
  }

  updateSnapshot() {
    this.unseenBlockCount = 0
    this.snapshotBlocks.clear()

    const updatedBlocks = Array.from(this.liveBlocks.values()).slice(-5)

    updatedBlocks.forEach((block) => setBlockInMap(this.snapshotBlocks, block))
  }

  setSearchResult(block: BlockSummary) {
    this.searchResult = block
  }

  findById(blockId: string): BlockSummary | undefined {
    if (this.searchResult != null && blockMatchesIdentifier(this.searchResult, blockId)) {
      return this.searchResult
    }

    // TODO: Search by block hash (block_mroot) before using the key to return the value
    //       previous block navigation uses the block_mroot, has such, to avoid unecessary
    //       API calls, searching with block hash might return some results while with key
    //       (being usually block_num) won't
    return this.liveBlocks.get(blockId)
  }
}

function setBlockInMap(map: ObservableMap<string, BlockSummary>, block: BlockSummary) {
  map.set(block.block_num.toString(), block)
}

function blockMatchesIdentifier(block: BlockSummary, blockId: string) {
  return block.block_num.toString() === blockId || block.id === blockId
}
