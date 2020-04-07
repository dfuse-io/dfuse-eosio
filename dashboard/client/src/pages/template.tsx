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
import { Col, Row } from 'antd';
import { withBaseLayout } from '../components/layout/layout';
import { WidgetBox } from '../components/widgets/widget-box';
import { WidgetContent } from '../components/widgets/widget-content';
import { WidgetTitle } from '../components/widgets/widget-title';
import { DriftGraph } from '../components/drift-graph/drift-graph';
import { WidgetMetricItem } from '../components/widgets/header-metric-item';
import { AppStatus } from '../atoms/status';
import { colors } from '../theme/colors';
import { useMetrics } from '../context/metrics';
import { useStatus } from '../context/status';

const BaseTemplatePage = (props: { appId: string }) => {
  const { appId } = props;

  const appStatus = useStatus()?.find(status => status.name === appId) || {
    name: '',
    description: '',
    status: 'NOTFOUND'
  };
  const appMetrics = useMetrics()?.filter(metric => metric.id === appId);

  return (
    <>
      <Row gutter={[16, 16]}>
        <Col className='gutter-row' span={24}>
          <WidgetBox>
            <WidgetTitle
              widgetTitleSize={5}
              widgetTitleText={
                appStatus.name.charAt(0).toUpperCase() + appStatus.name.slice(1)
              }
            >
              <AppStatus appStatus={appStatus?.status} />
            </WidgetTitle>
            <WidgetContent
              widgetPadding={25}
              asBorderBottom
              asBorderTop
              backgroundColor={colors.ternary200}
            >
              <Row gutter={[16, 0]} justify='space-between'>
                <Col className='gutter-row'>
                  <div>
                    <WidgetMetricItem
                      metricLabel={'Current Drift Time'}
                      metricData={'200ms'}
                    />
                  </div>
                </Col>
                <Col className='gutter-row'>
                  <div>
                    <WidgetMetricItem
                      metricLabel={'Current Head Block Number'}
                      metricData={12323400}
                    />
                  </div>
                </Col>
              </Row>
            </WidgetContent>
            <WidgetContent asBorderBottom>
              <DriftGraph data={appMetrics} />
            </WidgetContent>
            {/* <WidgetContent widgetPadding={25}>
              <WidgetLogs
                logs={[
                  {
                    timestamp: '2020-03-17T14:40:22.952',
                    tag: 'INFO',
                    description: `thread-0 chain_plugin.cpp:426
              operator() ] Support for builtin protocol feature
              'PREACTIVATE_FEATURE' (with digest of
              '0ec7e080177b2c02b278d5088611686b49d739925a92d9bfcacd7fc6b74053bd')
              is enabled without activation restrictions`
                  },
                  {
                    timestamp: '2020-03-17T14:40:22.961',
                    tag: 'DEBUG',
                    description: `thread-0 chain_plugin.cpp:595 plugin_initialize ]
                    initializing chain plugin`
                  },
                  {
                    timestamp: '2020-03-17T14:40:22.952',
                    tag: 'ERROR',
                    description: `thread-0 chain_plugin.cpp:426
              operator() ] Support for builtin protocol feature
              'PREACTIVATE_FEATURE' (with digest of
              '0ec7e080177b2c02b278d5088611686b49d739925a92d9bfcacd7fc6b74053bd')
              is enabled without activation restrictions`
                  },
                  {
                    timestamp: '2020-03-17T14:40:22.951',
                    tag: 'WARN',
                    description: `thread-0 chain_plugin.cpp:595 plugin_initialize ]
                    initializing chain plugin`
                  }
                ]}
              />
            </WidgetContent> */}
          </WidgetBox>
        </Col>
      </Row>
    </>
  );
};

export const TemplatePage = withBaseLayout(BaseTemplatePage);
