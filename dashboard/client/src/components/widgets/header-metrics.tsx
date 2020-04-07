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
import { Row, Col } from 'antd';
import { WidgetMetricItem } from './header-metric-item';
import { useMetrics } from '../../context/metrics';
import {
  MINDREADER_APP_ID
} from '../../utils/constants';
import {BlockNumberWrapper} from "./block-number";

export const HeaderMetrics = () => {
  const appMetrics = useMetrics();
  const managerAppIndex = appMetrics?.findIndex(
    metric => metric.id === MINDREADER_APP_ID
  );

  let managerHeadBlockNumber: (number|undefined) = undefined;
  if(appMetrics && managerAppIndex && managerAppIndex !== -1) {
    managerHeadBlockNumber = appMetrics[managerAppIndex].headBlockNumber;
  }
  return (
    <>
      <Row gutter={16}>
        <Col className='gutter-row' span={6}>
          <div>
            <WidgetMetricItem
              metricLabel={'Head Block Number'}
              metricData={<BlockNumberWrapper blockNumber={managerHeadBlockNumber} />}
            />
          </div>
        </Col>
      </Row>
    </>
  );
};
