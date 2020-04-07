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

import React from 'react';
import { colors } from '../../theme/colors';
import styled from '@emotion/styled';
import { Cell } from '../../atoms/grid';

const WidgetStyled = styled(Cell)`
  background: ${colors.white};
  border-radius: 5px;
  box-shadow: 0px 6px 9px rgba(10, 38, 58, 0.1);
  height: 100% !important;
  display: flex;
  flex-direction: column;
  box-sizing: border-box;
`;

export const WidgetBox: React.FC<{
  className?: string;
  minHeight?: string;
}> = ({ minHeight, children }) => (
  <WidgetStyled minHeight={minHeight}>{children}</WidgetStyled>
);
