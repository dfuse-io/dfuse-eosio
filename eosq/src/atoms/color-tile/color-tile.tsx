import { styled } from "../../theme"
import { Text } from "../text/text.component"
import * as React from "react"

interface ColorTileStyleProps {
  size?: number
}

export const ColorTile: React.ComponentType<any> = styled(Text)`
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  min-width: ${(props: ColorTileStyleProps) => {
    return props.size ? `${props.size}px` : "14px;"
  }};
  width: ${(props: ColorTileStyleProps) => {
    return props.size ? `${props.size}px` : "14px;"
  }};
  height: ${(props: ColorTileStyleProps) => {
    return props.size ? `${props.size}px` : "14px;"
  }};
`
