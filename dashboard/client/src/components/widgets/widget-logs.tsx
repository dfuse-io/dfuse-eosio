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

import React, { useState } from 'react';
import { styled } from '../../theme';
import { Checkbox, Col, Row, Timeline } from 'antd';
import { WidgetTitle } from './widget-title';
import { colors } from '../../theme/colors';
import { fonts } from '../../theme/fonts';

const TimelineWrapper = styled.div`
  &.logs .ant-timeline-item {
    opacity: 0;
    height: 0px;
    max-height: 0px;
    padding-bottom: 0px;
    transition: max-height 1s, opacity 0.2s 0.2s ease;
  }
  &.logs.info .ant-timeline-item.info,
  &.logs.error .ant-timeline-item.error,
  &.logs.debug .ant-timeline-item.debug,
  &.logs.warn .ant-timeline-item.warn {
    opacity: 1;
    height: auto;
    max-height: 1000px;
    padding-bottom: 20px;
  }

  font-family: ${fonts.mono};
  .ant-timeline.ant-timeline-label .ant-timeline-item-label {
    width: calc(25% - 12px);
  }
  .ant-timeline.ant-timeline-label .ant-timeline-item-tail {
    left: 25%;
    border-left: 2px solid ${colors.ternary250};
    top: 8px;
  }
  .ant-timeline-item-head.ant-timeline-item-head-blue {
    left: 25%;
    top: -2px;
  }
  .ant-timeline.ant-timeline-label .ant-timeline-item-content {
    left: calc(25% - 4px);
    width: calc(75% - 14px);
  }
  .ant-timeline-item.info .ant-timeline-item-head {
    border-color: ${colors.appColors[8]};
  }
  .ant-timeline-item.debug .ant-timeline-item-head {
    border-color: ${colors.link700};
  }
  .ant-timeline-item.warn .ant-timeline-item-head {
    border-color: ${colors.alert1000};
  }
  .ant-timeline-item.error .ant-timeline-item-head {
    border-color: ${colors.primary6};
  }
  .ant-timeline-item.info .log-label {
    color: ${colors.appColors[8]};
  }
  .ant-timeline-item.debug .log-label {
    color: ${colors.link700};
  }
  .ant-timeline-item.warn .log-label {
    color: ${colors.alert1000};
  }
  .ant-timeline-item.error .log-label {
    color: ${colors.primary6};
  }
`;

const plainOptions = ['INFO', 'DEBUG', 'ERROR', 'WARN'];

interface Log {
  timestamp: string;
  tag: string;
  description: string;
}

export interface LogsProps {
  logs: Log[];
}

export const WidgetLogs = (props: LogsProps) => {
  const [logsClass, setlogsClass] = useState('logs error warn');

  function onChange(checkedValues: any) {
    setlogsClass(
      'logs ' +
        checkedValues
          .toString()
          .replace(/,/g, ' ')
          .toLowerCase()
    );
  }

  return (
    <TimelineWrapper className={logsClass}>
      <Row gutter={[16, 16]} justify='space-between' align='middle'>
        <Col className='gutter-row'>
          <WidgetTitle widgetTitleSize={4} widgetTitleText={'Logs'} />
        </Col>
        <Col className='gutter-row' style={{ paddingRight: '25px' }}>
          <Checkbox.Group
            options={plainOptions}
            defaultValue={['ERROR', 'WARN']}
            onChange={onChange}
          />
        </Col>
      </Row>
      <Timeline mode='left'>
        {props.logs.map((log: Log) => (
          <Timeline.Item
            className={log.tag.toLowerCase()}
            label={log.timestamp}
          >
            <div className='log-label'>{log.tag.toUpperCase()}</div>
            {log.description}
          </Timeline.Item>
        ))}
      </Timeline>
    </TimelineWrapper>
  );
};
