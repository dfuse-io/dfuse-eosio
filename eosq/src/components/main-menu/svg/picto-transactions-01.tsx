import * as React from "react"

export const PictoTransactions = (props: any) => (
  <svg width="1.7em" height="1.7em" viewBox="0 0 200 200" {...props}>
    <path
      fill="none"
      stroke={props.color}
      strokeWidth={25.345}
      strokeMiterlimit={10}
      d="M24.5 98.812h150v60.216h-150z"
    />
    <path fill={props.color} d="M157.186 23.463L99.495 90.378 41.808 23.463z" />
  </svg>
)
