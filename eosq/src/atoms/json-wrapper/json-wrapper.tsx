import { styled } from "../../theme"
import * as React from "react"
import { fontSize as _fontSize } from "styled-system"

const JsonContainerCode: React.ComponentType<any> = styled.code`
  ${_fontSize};
  font-family: "Roboto Mono", monospace;
  // padding: 0 16px 0 16px;
  white-space: pre-wrap;
  word-wrap: break-word;
  display: block;
  overflow-x: hidden;
`

const JsonContainerPre: React.ComponentType<any> = styled.pre`
  word-wrap: break-all;
  white-space: pre-wrap;
  display: block;
  overflow-x: hidden;
`

export const JsonWrapper: React.FC<any> = ({ fontSize, children }) => (
  <JsonContainerPre>
    <JsonContainerCode fontSize={fontSize || "12px"}>{children}</JsonContainerCode>
  </JsonContainerPre>
)
