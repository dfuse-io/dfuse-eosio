/**
 * Copyright 2019 dfuse Platform Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import styled from '@emotion/styled';
import React from 'react';
import {
  alignItems,
  AlignItemsProps,
  alignSelf,
  border,
  borderBottom,
  borderColor,
  borderRadius,
  borders,
  BordersProps,
  color,
  ColorProps,
  display,
  DisplayProps,
  fontFamily,
  fontSize,
  FontSizeProps,
  fontWeight,
  FontWeightProps,
  gridAutoFlow,
  gridColumn,
  gridColumnGap,
  GridColumnProps,
  gridGap,
  gridRow,
  gridRowGap,
  GridRowProps,
  gridTemplateColumns,
  gridTemplateRows,
  height,
  justifyContent,
  JustifyContentProps,
  justifyItems,
  JustifyItemsProps,
  justifySelf,
  JustifySelfProps,
  lineHeight,
  LineHeightProps,
  maxHeight,
  maxWidth,
  minHeight,
  minWidth,
  opacity,
  OpacityProps,
  overflow,
  OverflowProps,
  space,
  SpaceProps,
  textAlign,
  TextAlignProps,
  width
} from 'styled-system';
import {
  ChildPositioningProps,
  ColoringProps,
  GridLayoutProps,
  SelfPositioningProps,
  SizingProps
} from '../theme/system';
import { theme } from '../theme';

export interface GridProps
  extends React.HTMLAttributes<HTMLDivElement>,
    BordersProps,
    ColoringProps,
    GridLayoutProps,
    JustifyContentProps,
    JustifyItemsProps,
    JustifySelfProps,
    SizingProps,
    SpaceProps,
    ChildPositioningProps,
    SelfPositioningProps,
    BordersProps {
  color?: string;
}

export const Grid: React.FC<GridProps> = styled.div`
    display: grid;
    position: relative;
    ${border}
    ${alignSelf}
    ${alignItems}
    ${justifySelf}
    ${justifyItems}
    ${gridGap}
    ${gridColumnGap}
    ${gridRowGap}
    ${gridRow}
    ${gridColumn}
    ${gridTemplateColumns}
    ${gridTemplateRows}
    ${gridAutoFlow}
    ${height}
    ${width}
    ${minHeight}
    ${minWidth}
    ${space}
    ${color}
    ${textAlign}
    ${borders}
    ${borderColor}
    ${borderRadius}
    ${maxWidth}
    ${maxHeight}
  `;

export interface CellProps
  extends React.HTMLAttributes<HTMLDivElement>,
    AlignItemsProps,
    DisplayProps,
    ColorProps,
    JustifyItemsProps,
    GridRowProps,
    OpacityProps,
    GridColumnProps,
    LineHeightProps,
    SizingProps,
    JustifyContentProps,
    BordersProps,
    SpaceProps,
    SelfPositioningProps,
    OverflowProps,
    FontWeightProps,
    TextAlignProps,
    FontSizeProps {
  color?: string;
}

export const Cell: React.FC<CellProps> = styled.div`
    position: relative;
    ${alignItems}
    ${justifyContent}
    ${opacity}
    ${display}
    ${borders}
    ${borderColor}
    ${borderRadius}
    ${borderBottom}
    ${justifyItems}
    ${fontSize}
    ${fontFamily}
    ${alignSelf}
    ${justifySelf}
    ${gridColumn}
    ${gridRow}
    ${height}
    ${width}
    ${minHeight}
    ${minWidth}
    ${space}
    ${color}
    ${textAlign}
    ${lineHeight}
    ${alignItems}
    ${maxWidth}
    ${maxHeight}
    ${overflow}
    ${fontWeight}
  `;

export const HoverableCell: React.FC<CellProps> = styled.div`
  position: relative;
  ${alignItems}
  ${justifyContent}
  ${opacity}
  ${display}
  ${borders}
  ${borderColor}
  ${borderRadius}
  ${borderBottom}
  ${justifyItems}
  ${fontSize}
  ${fontFamily}
  ${alignSelf}
  ${justifySelf}
  ${gridColumn}
  ${gridRow}
  ${height}
  ${width}
  ${minHeight}
  ${minWidth}
  ${space}
  ${color}
  ${textAlign}
  ${lineHeight}
  ${alignItems}
  ${maxWidth}
  ${maxHeight}
  ${overflow}
  ${fontWeight}
  &:hover {
    cursor: pointer;
    div,
    svg {
      color: ${theme.colors.link500};
      d {
        fill: ${theme.colors.link500} !important;
      }
    }
  }
`;
