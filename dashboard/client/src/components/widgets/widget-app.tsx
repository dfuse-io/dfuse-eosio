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
import { styled } from '../../theme';
import { Cell } from '../../atoms/grid';
import { AppStatus } from '../../atoms/status';
import { TitleStyled } from '../../atoms/typography';
import { ColorLine } from '../../atoms/color-line';
import { BlockNumberWrapper } from './block-number';
import { Col, Row } from 'antd';
import { colors } from '../../theme/colors';
import { durationToHumanBeta } from '../../utils/time';
import { MetricConfig, INFINITE_DRIFT_THRESHOLD } from '../../utils/constants';

const WidgetAppStyled = styled(Cell)`
  display: flex;
  font-size: 20px;
  flex-direction: column;
  font-size: ${fontSizes[1]}px !important;
  position: relative;
  overflow: hidden;
  min-height: 220px;
  padding: 30px 30px 30px 30px;
`;

const WidgetTitleWrapper = styled(Cell)<{ level?: number }>`
  display: flex;
  align-items: center;
  margin-bottom: 10px;
  h1 {
    flex-grow: 1;
    margin: 0;
    font-size: ${props =>
      props.level ? fontSizes[props.level] : fontSizes[3]}px !important;
    line-height: ${props =>
      props.level ? fontSizes[props.level] : fontSizes[3]}px !important;
  }
  > :last-child {
    flex-grow: 0;
  }
`;

const DescriptionWrapper = styled(Cell)`
  display: block;
  width: 100%;
  overflow: hidden;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  line-height: 1.5em;
  margin: 0px;
  min-height: 50px;
`;

const MetricLabel = styled(Cell)`
  color: ${colors.ternary500};
  font-weight: normal;
  letter-spacing: 0px;
  font-family: 'Lato', sans-serif;
  margin-bottom: 5px;
  font-size: ${fontSizes[0]}px;
  line-height: ${fontSizes[0]}px;
`;

const MetricData = styled(Cell)`
  color: ${colors.ternary1000};
  font-weight: normal;
  letter-spacing: 0px;
  font-family: 'Lato', sans-serif;
  margin-bottom: 5px;
  font-size: ${fontSizes[1]}px;
  line-height: ${fontSizes[1]}px;
`;

export type appInfo = {
  color?: string;
  title?: string;
  description?: string;
  status?: string;
  drift?: number;
  headBlockNumber?: number;
  metricConfig?: MetricConfig;
};

export const WidgetApp: React.FC<{
  onNavigate?: () => void;
  appInfo: appInfo;
}> = props => {
  const {
    appInfo: {
      color,
      title,
      description,
      status,
      drift,
      headBlockNumber,
      metricConfig
    }
  } = props;

  let driftToDisplay = '0 sec';

  if (drift) {
    if (drift < INFINITE_DRIFT_THRESHOLD) {
      driftToDisplay = durationToHumanBeta(drift);
    } else {
      driftToDisplay = '∞';
    }
  }
  return (
    <WidgetAppStyled>
      <ColorLine borderColor={color} />
      <AppStatus appStatus={status} />
      <WidgetTitleWrapper level={3}>
        <TitleStyled>{title}</TitleStyled>
      </WidgetTitleWrapper>
      <DescriptionWrapper>{description}</DescriptionWrapper>
      <Row gutter={[1, 0]} justify='space-between'>
        {metricConfig && metricConfig.headBlockDrift && (
          <Col className='gutter-row'>
            <MetricLabel>drift</MetricLabel>
            <MetricData>{driftToDisplay}</MetricData>
          </Col>
        )}
        {metricConfig && metricConfig.headBlockNumber && (
          <Col
            className='gutter-row'
            style={{ float: 'right', textAlign: 'right' }}
          >
            <MetricLabel>head block #</MetricLabel>
            <MetricData>
              <BlockNumberWrapper blockNumber={headBlockNumber} />
            </MetricData>
          </Col>
        )}
      </Row>
    </WidgetAppStyled>
  );
};
