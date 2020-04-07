import React from 'react';
import { withBaseLayout } from '../components/layout/layout';
import { Col, Row } from 'antd';
import { WidgetBox } from '../components/widgets/widget-box';
import { WidgetContent } from '../components/widgets/widget-content';
import { WidgetTitle } from '../components/widgets/widget-title';
import { WidgetLogs } from '../components/widgets/widget-logs';
// import { styled } from '../theme';
// import { Cell } from '../atoms/grid';
import { DriftGraph } from '../components/drift-graph/drift-graph';
import appMetrics from '../components/drift-graph/drift-data-sample.json';
import { WidgetMetricItem } from '../components/widgets/header-metric-item';
import { AppStatus } from '../atoms/status';
import { colors } from '../theme/colors';

const BaseMockPage = () => {
  return (
    <>
      <Row gutter={[16, 16]}>
        <Col className='gutter-row' span={24}>
          <WidgetBox>
            <WidgetTitle widgetTitleSize={5} widgetTitleText={'Mock'}>
              <AppStatus appStatus={'RUNNING'} />
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
            <WidgetContent>
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
            </WidgetContent>
          </WidgetBox>
        </Col>
      </Row>
    </>
  );
};

export const MockPage = withBaseLayout(BaseMockPage);
