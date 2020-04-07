import { CursorCache } from "../cursor-store"

describe("CursorCache", () => {
  describe("setCurrentCursor", () => {
    it("should set the current cursor to the given value", () => {
      const cursorCache = new CursorCache()
      cursorCache.setCurrentCursor("abc")
      expect(cursorCache.currentCursor).toEqual("abc")
    })
  })

  describe("shiftToNextCursor", () => {
    it("should set the next cursor to the given value", () => {
      const cursorCache = new CursorCache()
      cursorCache.prepareNextCursor("abcd")
      expect(cursorCache.hasNextPage).toEqual(true)
      expect(cursorCache.shiftToNextCursor()).toEqual("abcd")
    })
  })

  describe("shiftToPreviousCursor", () => {
    it("should return back to first cursor", () => {
      const cursorCache = new CursorCache()
      cursorCache.setCurrentCursor("first")
      cursorCache.prepareNextCursor("second")
      expect(cursorCache.shiftToNextCursor()).toEqual("second")
      expect(cursorCache.hasPreviousPage).toEqual(true)

      cursorCache.prepareNextCursor("third")
      expect(cursorCache.shiftToPreviousCursor()).toEqual("first")
    })
  })

  describe("resetAll", () => {
    it("should resetAll values", () => {
      const cursorCache = new CursorCache()
      cursorCache.setCurrentCursor("first")
      cursorCache.prepareNextCursor("second")
      expect(cursorCache.shiftToNextCursor()).toEqual("second")
      cursorCache.prepareNextCursor("third")
      cursorCache.resetAll()
      expect(cursorCache.currentCursor).toEqual("")
      expect(cursorCache.hasNextPage).toEqual(false)
      expect(cursorCache.hasPreviousPage).toEqual(false)
    })
  })
})
