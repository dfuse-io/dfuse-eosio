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

import {
  AlignContentProps,
  AlignItemsProps,
  AlignSelfProps,
  BackgroundProps,
  ColorProps,
  FlexBasisProps,
  FlexDirectionProps,
  FlexProps,
  FlexWrapProps,
  FontFamilyProps,
  FontSizeProps,
  FontWeightProps,
  GridAutoColumnsProps,
  GridAutoFlowProps,
  GridAutoRowsProps,
  GridColumnGapProps,
  GridGapProps,
  GridRowGapProps,
  GridTemplateColumnsProps,
  GridTemplateRowsProps,
  HeightProps,
  JustifyContentProps,
  JustifySelfProps,
  MaxHeightProps,
  MaxWidthProps,
  MinHeightProps,
  MinWidthProps,
  SizeProps,
  TextAlignProps,
  WidthProps,
} from "styled-system";

export interface ColoringProps extends BackgroundProps, ColorProps {}

export interface FlexLayoutProps
  extends FlexWrapProps,
    FlexBasisProps,
    FlexProps,
    FlexDirectionProps {}

export interface GridLayoutProps
  extends GridAutoFlowProps,
    GridAutoRowsProps,
    GridAutoColumnsProps,
    GridGapProps,
    GridRowGapProps,
    GridColumnGapProps,
    GridTemplateRowsProps,
    GridTemplateColumnsProps {}

export interface TypographyProps
  extends FontSizeProps,
    FontWeightProps,
    FontFamilyProps,
    TextAlignProps {}

export interface ChildPositioningProps
  extends AlignItemsProps,
    AlignContentProps,
    JustifyContentProps {}
export interface SelfPositioningProps
  extends AlignSelfProps,
    JustifySelfProps {}

export interface SizingProps
  extends SizeProps,
    MinHeightProps,
    MaxHeightProps,
    HeightProps,
    MinWidthProps,
    MaxWidthProps,
    WidthProps {}
