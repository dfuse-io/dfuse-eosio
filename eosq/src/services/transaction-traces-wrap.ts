import { ActionTrace, CreationNode } from "@dfuse/client"
import { groupBy } from "ramda"
import { DeferredOperation, TraceLevel } from "../models/transaction"

interface TraceLevelGroups {
  [key: number]: ActionTrace<any>[]
}

interface ParentDepthMap {
  [index: number]: number
}

export function groupTracesByTraceLevel(
  traces: ActionTrace<any>[],
  creationTree?: CreationNode[],
  displayCreationTraces?: boolean
): TraceLevelGroups {
  const traceLevelsExecution = computeTraceWithLevel(traces)[0]
  let displayedTraces: TraceLevel[] = traceLevelsExecution
  if (creationTree && creationTree.length > 0 && displayCreationTraces) {
    displayedTraces = computeCreationTraceWithLevel(creationTree, traceLevelsExecution)
  }
  // @ts-ignore The group key can be a number but definitions are accepting string only
  return groupBy((traceLevel: TraceLevel) => {
    return traceLevel.group
  }, displayedTraces)
}

export function computeTraceWithLevel(
  actionTraces: ActionTrace<any>[],
  currentLevel: number = 0,
  currentGroup: number = 0,
  results: TraceLevel[] = [],
  index: number = 0
): [TraceLevel[], number] {
  actionTraces.forEach((actionTrace: ActionTrace<any>) => {
    results.push({ actionTrace, level: currentLevel, group: currentGroup, index })
    index += 1

    if (actionTrace.inline_traces && actionTrace.inline_traces.length > 0) {
      // eslint-disable-next-line prefer-destructuring
      index = computeTraceWithLevel(
        actionTrace.inline_traces,
        currentLevel + 1,
        currentGroup,
        results,
        index
      )[1]
    }

    if (currentLevel === 0) {
      currentGroup++
    }
  })

  return [results, index]
}

export function computeCreationTraceWithLevel(
  creationTree: CreationNode[],
  traceLevels: TraceLevel[]
): TraceLevel[] {
  const results: TraceLevel[] = []
  let groupIndex = -1
  let depth = 0
  const parentDepth: ParentDepthMap = {}

  creationTree.forEach((creationNode: CreationNode) => {
    const currentIndex = creationNode[0]
    const parentIndex = creationNode[1]
    const traceLevelIndex = creationNode[2]
    const currentTraceLevel = traceLevels[traceLevelIndex]

    if (parentIndex === -1) {
      groupIndex += 1
      depth = 0
      parentDepth[currentIndex] = depth
    } else {
      const referenceDepth = parentDepth[parentIndex]
      depth = referenceDepth + 1
      parentDepth[currentIndex] = depth
    }

    results.push({
      index: currentIndex,
      actionTrace: currentTraceLevel.actionTrace,
      group: groupIndex,
      level: depth
    })
  })

  return results
}

export class TransactionTracesWrap {
  traces: ActionTrace<any>[] = []
  traceLevels: TraceLevelGroups
  actionIndexes: number[] = []
  deferredOperations: DeferredOperation[]

  constructor(
    actionTraces: ActionTrace<any>[],
    deferredOperations?: DeferredOperation[],
    actionIndexes?: number[],
    creationTree?: CreationNode[],
    displayCreationTraces?: boolean
  ) {
    this.traces = actionTraces || []

    this.traceLevels = this.traces
      ? groupTracesByTraceLevel(this.traces, creationTree, displayCreationTraces)
      : {}
    this.actionIndexes = actionIndexes || []
    this.deferredOperations = deferredOperations || []
  }

  collapsedTraces(traceLevels: TraceLevel[]) {
    if (this.actionIndexes.length === 0) {
      return traceLevels.filter((traceWithLevel: TraceLevel) => {
        return traceWithLevel.index === 0
      })
    }
    return traceLevels.filter((traceWithLevel: TraceLevel) => {
      return (
        this.actionIndexes.length === 0 ||
        this.actionIndexes.includes(traceWithLevel.index) || traceWithLevel.index === 0
      )
    })
  }

  mapGroups(callback: (traceLevels: TraceLevel[], key: string) => any) {
    return Object.keys(this.traceLevels).map((key: string) => {
      const group = this.traceLevels[key]
      return callback(group, key)
    })
  }

  hiddenActionsCount() {
    let extraActions = 0
    this.mapGroups((traceLevels: TraceLevel[]) => {
      extraActions += traceLevels.length - this.collapsedTraces(traceLevels).length
    })

    return extraActions
  }
}
