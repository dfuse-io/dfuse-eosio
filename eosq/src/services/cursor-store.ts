export class CursorCache {
  private cursorHistory: string[] = []
  private currentPageCursor = ""

  private nextPageCursor = ""

  get currentCursor() {
    return this.currentPageCursor
  }

  get hasNextPage() {
    return this.nextPageCursor.length > 0
  }

  get hasPreviousPage() {
    return this.cursorHistory.length > 1
  }

  setCurrentCursor(currentCursor: string | undefined) {
    this.cursorHistory.push("")
    this.currentPageCursor = currentCursor || ""
  }

  prepareNextCursor(nextCursor: string | undefined) {
    this.nextPageCursor = nextCursor || ""
    return this.nextPageCursor
  }

  shiftToPreviousCursor() {
    this.nextPageCursor = this.currentPageCursor

    this.currentPageCursor = this.cursorHistory.pop()!
    return this.currentPageCursor
  }

  shiftToNextCursor() {
    this.cursorHistory.push(this.currentPageCursor)
    this.currentPageCursor = this.nextPageCursor

    return this.nextPageCursor
  }

  resetAll(nextCursor?: string) {
    this.cursorHistory = []
    this.nextPageCursor = nextCursor || ""
    this.currentPageCursor = ""
  }
}
