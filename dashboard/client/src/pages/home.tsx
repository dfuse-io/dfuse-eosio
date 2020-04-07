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
import { withBaseLayout } from '../components/layout/layout';
import { WidgetBox } from '../components/widgets/widget-box';
import { Col, Row } from 'antd';
import { WidgetContent } from '../components/widgets/widget-content';
import { WidgetTitle } from '../components/widgets/widget-title';
import { WidgetApp, appInfo } from '../components/widgets/widget-app';
import { DriftGraph } from '../components/drift-graph/drift-graph';
import { history } from '../services/history';
import { useAppsList, AppInfoToDisplay } from '../context/apps-list';
import { useMetrics } from '../context/metrics';
import { useStatus } from '../context/status';
import { appMetric } from '../services/data-providers/metrics';
import { AppStatusDisplay } from '../services/data-providers/status';
import { getAppColor } from '../theme/colors';
import { METRIC_CONFIG } from '../utils/constants';

const renderAppWidgets = (
  appsList: AppInfoToDisplay[] | null,
  appsStatus: AppStatusDisplay[] | undefined,
  metrics: appMetric[] | undefined
) =>
  appsList?.map(app => {
    const appMetric = metrics?.filter(m => m.id === app.id)[0];
    const appStatus = appsStatus?.filter(a => a.name === app.id)[0];
    const appDrift = appMetric?.headBlockDrift.slice(-1)[0]?.value;
    const appHeadBlockNumber = appMetric?.headBlockNumber;
    const appMetricConfig = METRIC_CONFIG[app.id];
    const info: appInfo = {
      color: getAppColor(app.id),
      title: app.title.toUpperCase(),
      description: app.description,
      status: appStatus?.status,
      drift: appDrift,
      headBlockNumber: appHeadBlockNumber,
      metricConfig: appMetricConfig
    };
    return (
      <Col className='gutter-row' span={8} key={`col-${app.id}-graph`}>
        <WidgetBox>
          <WidgetApp appInfo={info} onNavigate={() => history.push(app.id)} />
        </WidgetBox>
      </Col>
    );
  });

const BaseHomePage: React.FC = () => {
  const appsList = useAppsList();
  const appsStatus = useStatus();
  const appMetrics = useMetrics();
  return (
    <>
      <Row gutter={[16, 16]}>
        <Col className='gutter-row' span={24} key={'col-drif-graph'}>
          <WidgetBox>
            <WidgetTitle
              widgetTitleSize={3}
              widgetTitleText={'Head Block Drift'}
            />
            <WidgetContent>
              <DriftGraph data={appMetrics} />
            </WidgetContent>
          </WidgetBox>
        </Col>
        {renderAppWidgets(appsList, appsStatus, appMetrics)}
      </Row>
    </>
  );
};

export const HomePage = withBaseLayout(BaseHomePage);
