// Typographic Scale (numbers are converted to px values)
export const fontSizes = [10, 12, 14, 16, 20, 24, 32, 40, 48, 64, 72]
export const lineHeights = ["18px", "22px", "26px", "34px", "40px", "50px"]

// Spacing Scale (used for margin and padding)
export const space = [0, 4, 8, 16, 32, 64, 128, 256, 512]

export const breakPoints = {
  small: 768,
  medium: 1280,
  large: 1440
}

export const mediaQueries = {
  smallOnly: `@media (max-width: ${breakPoints.small - 1}px)`,
  small: `@media (min-width: ${breakPoints.small}px)`,
  medium: `@media (min-width: ${breakPoints.medium}px)`,
  large: `@media (min-width: ${breakPoints.large}px)`
}
