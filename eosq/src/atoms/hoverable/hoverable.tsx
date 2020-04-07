import * as React from "react"
import { styled } from "../../theme"
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome"

export const HoverableIcon: React.ComponentType<any> = styled(FontAwesomeIcon)`
  &:hover {
    cursor: pointer;
    color: ${(props) => props.theme.colors.linkHover};
  }
`
