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
import { styled } from '../theme';
import { Cell } from './grid';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import {
  faCheck,
  faExclamation,
  faTimes
} from '@fortawesome/free-solid-svg-icons';
import { fontSizes } from '../theme/scales';
import { Tooltip, Typography } from 'antd';

const { Text } = Typography;

const StatusIconWrapper = styled(Cell)<{
  iconBackgroundColor?: string;
  iconBorderColor?: string;
}>`
  width: 26px;
  height: 26px;
  background: ${props => props.iconBackgroundColor};
  border: 1px solid ${props => props.iconBorderColor};
  padding: 0px;
  border-radius: 13px;
  display: flex;
  align-items: center;
  justify-content: center;
  position: absolute;
  right: 20px;
  top: 20px;

  h1 {
    font-size: ${props => fontSizes[1]}px !important;
    line-height: ${props => fontSizes[1]}px !important;
  }
`;

interface AppStatusProps {
  appStatus?: string;
}

export const AppStatus = (props: AppStatusProps) => {
  let iconCode, iconColor, BackgroundColor, BorderColor;

  switch (props.appStatus) {
    case 'NOTFOUND':
      iconCode = faExclamation;
      iconColor = '#fff';
      BackgroundColor = '#ff4660';
      BorderColor = '#ff4660';
      break;
    case 'CREATED':
      iconCode = faCheck;
      iconColor = '#fff';
      BackgroundColor = '#219ce4';
      BorderColor = '#219ce4';
      break;
    case 'RUNNING':
      iconCode = faCheck;
      iconColor = '#fff';
      BackgroundColor = '#61d8c8';
      BorderColor = '#61d8c8';
      break;
    case 'WARNING':
      iconCode = faExclamation;
      iconColor = '#fff';
      BackgroundColor = '#ffb230';
      BorderColor = '#ffb230';
      break;
    case 'STOPPED':
      iconCode = faTimes;
      iconColor = '#fff';
      BackgroundColor = '#ff4660';
      BorderColor = '#ff4660';
      break;
    default:
      iconCode = faExclamation;
      iconColor = '#fff';
      BackgroundColor = '#ff4660';
      BorderColor = '#ff4660';
      break;
  }

  return (
    <Tooltip
      placement='top'
      mouseEnterDelay={0.01}
      mouseLeaveDelay={0.7}
      title={<Text>{props.appStatus}</Text>}
    >
      <StatusIconWrapper
        iconBackgroundColor={BackgroundColor}
        iconBorderColor={BorderColor}
      >
        <FontAwesomeIcon color={iconColor} icon={iconCode} />
      </StatusIconWrapper>
    </Tooltip>
  );
};
