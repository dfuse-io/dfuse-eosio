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
import { fontSizes } from '../../theme/scales';
import { styled, theme } from '../../theme';
import { Cell } from '../../atoms/grid';

const WidgetContentStyled = styled(Cell)<{ backgroundColor?: string }>`
  display: flex;
  flex-direction: column;
  font-size: 20px;
  height: 100%;
  display: flex;
  flex-direction: column;
  font-size: 20px;
  height: auto;
  font-size: ${fontSizes[1]}px!important;
  background: ${props => props.backgroundColor};
`;

export const WidgetContent: React.FC<{
  widgetPaddingX?: number;
  widgetPaddingY?: number;
  backgroundColor?: string;
  widgetPadding?: number;
  asBorderTop?: boolean;
  asBorderBottom?: boolean;
}> = ({
  backgroundColor,
  widgetPaddingX,
  widgetPaddingY,
  widgetPadding,
  asBorderTop,
  asBorderBottom,
  children
}) => (
  <WidgetContentStyled
    backgroundColor={backgroundColor}
    p={widgetPadding}
    px={widgetPaddingX}
    py={widgetPaddingY}
    borderTop={
      !asBorderTop ? '0px solid #fff' : '1px solid ' + theme.colors.ternary300
    }
    borderBottom={
      !asBorderBottom
        ? '0px solid #fff'
        : '1px solid ' + theme.colors.ternary300
    }
  >
    {children}
  </WidgetContentStyled>
);
