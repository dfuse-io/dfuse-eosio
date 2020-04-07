let hidden = "hidden"
let visibilityChange = "visibilitychange"
if (typeof document.hidden !== "undefined") {
  // Opera 12.10 and Firefox 18 and later support
  hidden = "hidden"
  visibilityChange = "visibilitychange"
  // @ts-ignore
} else if (typeof document.msHidden !== "undefined") {
  hidden = "msHidden"
  visibilityChange = "msvisibilitychange"
  // @ts-ignore
} else if (typeof document.webkitHidden !== "undefined") {
  // @ts-ignore
  hidden = "webkitHidden"
  visibilityChange = "webkitvisibilitychange"
}

export const HIDDEN = hidden

export const VISIBILITYCHANGE = visibilityChange

export const handleVisibilityChange = (
  visibleCallback?: () => void,
  hiddenCallback?: () => void
) => {
  return () => {
    if (document[HIDDEN!]) {
      if (hiddenCallback) {
        hiddenCallback()
      }
    } else if (visibleCallback) {
      visibleCallback()
    }
  }
}
