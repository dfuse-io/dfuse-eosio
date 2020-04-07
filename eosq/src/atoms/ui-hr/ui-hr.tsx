import { styled } from "../../theme"
import * as React from "react"

export const UiHrDotted: React.ComponentType<any> = styled.hr`
  border: none;
  border-bottom: 1px dotted ${(props) => props.theme.colors.grey4};
`

export const UiHrSpaced: React.ComponentType<any> = styled.hr`
  border: none;
  border-bottom: 1px solid ${(props) => props.theme.colors.grey4};
  margin-top: 30px;
  margin-bottom: 30px;
`

export const UiHrDense: React.ComponentType<any> = styled.hr`
  margin: 0;
  border: none;
  border-bottom: 1px dotted ${(props) => props.theme.colors.grey6};
`
