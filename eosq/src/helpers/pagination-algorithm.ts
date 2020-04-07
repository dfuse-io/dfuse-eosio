export function pagination(current: number, total: number) {
  const leftGap = current > 4
  const rightGap = total - current > 4

  const items: string[] = []

  for (let i = 0; i < total; i++) {
    if (leftGap && i > 1 && i < current - 1) {
      if (items[items.length - 1] !== "...") {
        items.push("...")
      }
    } else if (rightGap && i > current + 1 && i < total - 2) {
      if (items[items.length - 1] !== "...") {
        items.push("...")
      }
    } else {
      items.push(i.toString())
    }
  }

  return Object.assign([], items)
}
