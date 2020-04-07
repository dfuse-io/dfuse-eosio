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
import { Cell } from '../../atoms/grid';
import { styled } from '../../theme';
import { fontSizes } from '../../theme/scales';
import { colors } from '../../theme/colors';

const MetricLabel = styled(Cell)`
  color: ${colors.ternary500};
  font-weight: normal;
  letter-spacing: 0px;
  font-family: 'Lato', sans-serif;
  margin-bottom: 10px;
  font-size: ${fontSizes[1]}px;
  line-height: ${fontSizes[1]}px;
`;
const MetricData = styled(Cell)`
  color: ${colors.ternary1000};
  font-weight: normal;
  letter-spacing: 0px;
  font-family: 'Lato', sans-serif;
  margin-bottom: 5px;
  font-size: ${fontSizes[3]}px;
  line-height: ${fontSizes[3]}px;
`;

export const WidgetMetricItem: React.FC<{
  metricLabel: string;
  metricData?: React.ReactNode;
}> = ({ metricLabel, metricData }) => {
  return (
    <>
      <MetricLabel>
        {metricLabel}
      </MetricLabel>
      <MetricData>
        {metricData}
      </MetricData>
    </>
  );
};
