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
import { Typography } from 'antd';
import { fontSizes } from '../../theme/scales';
import { Cell } from '../../atoms/grid';

const { Title } = Typography;

const WidgetTitleWrapper = styled(Cell)<{ level?: number }>`
  display: flex;
  align-items: center;
  padding: 25px 25px 20px;
  position:relative;
  h1 {
    font-size: ${props =>
      props.level ? fontSizes[props.level] : fontSizes[3]}px !important;
    line-height: ${props =>
      props.level ? fontSizes[props.level] : fontSizes[3]}px !important;
  }
  > :last-child {
    margin-left: 10px;
  }
`;

const TitleStyled = styled(Title)`
  color: ${colors.ternary500}!important;
  font-weight: 400 !important;
  letter-spacing: 0.3px !important;
  font-family: 'Lato', sans-serif;
  margin-bottom: 0px !important;
`;

// @todo impossible de mettre number to widgetTitleSize au lieu de any!!!!
export const WidgetTitle: React.FC<{
  widgetTitleText: string;
  widgetTitleSize?: any;
}> = ({ widgetTitleText, widgetTitleSize, children }) => {
  return (
    <WidgetTitleWrapper level={widgetTitleSize}>
      <TitleStyled>{widgetTitleText}</TitleStyled>
      {children}
    </WidgetTitleWrapper>
  );
};
