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

import emotionStyled, { CreateStyled } from "@emotion/styled";
import { colors } from "./colors";
import { fonts } from "./fonts";
import { breakpoints, fontSizes, lineHeights, shadows, space } from "./scales";

export const theme = {
  breakpoints,
  fontSizes,
  lineHeights,
  space,
  colors,
  fonts,
  shadows,

  Link: {
    color: colors.primary5,
    cursor: "pointer",
    textDecoration: "underline",
  },
};

export type ThemeInterface = typeof theme;

export const styled = emotionStyled as CreateStyled<ThemeInterface>;