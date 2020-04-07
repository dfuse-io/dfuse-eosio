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

import React, {useState} from 'react';
import { Line } from 'react-chartjs-2';
import { appMetric } from '../../services/data-providers/metrics';
import { METRIC_CONFIG, MAX_GRAPH_DRIFT } from '../../utils/constants';
import { getAppColor } from '../../theme/colors';
import { Button } from 'antd'
import { AreaChartOutlined } from '@ant-design/icons';


const chartWrapperStyle = {
};
const chartActionWrapperStyle = {
  paddingBottom: "40px",
  marginRight: "10px"
};

const chartOptions =  (isLogarithmic: boolean): any =>  {
  return {
    responsive: true,
    title: {
      display: false,
      text: 'Head Block Drift (secs)'
    },
    tooltips: {
      callbacks: {}
    },
    legend: {
      position: 'top'
    },
    scales: {
      yAxes: [
        {
          scaleLabel: {
            display: true,
            labelString: 'Drift time in seconds',
            fontColor: '#9fadbc'
          },
          type: (isLogarithmic ? 'logarithmic' : 'linear'),
          gridLines: {
            color: '#f0f3f5'
          },
          ticks: {
            beginAtZero: true
          }
        }
      ],
      xAxes: [
        {
          scaleLabel: {
            display: true,
            labelString: 'Time',
            fontColor: '#9fadbc'
          },
          type: 'time',
          distribution: 'linear',
          gridLines: {
            color: '#f0f3f5'
          },
          time: {
            unit: 'second',
            stepSize: 3,
            displayFormats: {
              second: 'H:mm:ss'
            }
          }
        }
      ]
    }
  }
};


const cleanData = (appMetrics: appMetric[]): any[] => {
  const datasets: any[] = [];
  appMetrics.forEach(function(appMetric) {
    const metricConfig = METRIC_CONFIG[appMetric.id];
    if (metricConfig && metricConfig.headBlockDrift) {
      const filteredHeadBlockDrifts = appMetric.headBlockDrift.filter(
        value => value.value < MAX_GRAPH_DRIFT
      );
      datasets.push({
        label: appMetric.title,
        backgroundColor: 'transparent',
        pointBackgroundColor: getAppColor(appMetric.id),
        borderColor: getAppColor(appMetric.id),
        pointRadius: 2,
        fill: false,
        data: filteredHeadBlockDrifts.map((d, i) => ({
          x: d.timestamp,
          y: d.value
        })),
        borderWidth: 1
      });
    }
  });
  return datasets;
};

export const DriftGraph: React.FC<{ data: appMetric[] | undefined }> = ({
  data
}) => {
  const [isLogarithmic, setIsLogarithmic] = useState(false);

  if (!data) return null;
  const chartDataset = {
    datasets: cleanData(data)
  };

  const  toggleScaleType = () =>{
    setIsLogarithmic(!isLogarithmic)
  };

  return (
    <div style={chartWrapperStyle}>
      <Line data={chartDataset} options={chartOptions(isLogarithmic)} height={100} />
      <div style={chartActionWrapperStyle}>
        <Button size={'small'} onClick={toggleScaleType} type="primary" style={{float: 'right'}}  icon={<AreaChartOutlined />}>
          { isLogarithmic ? 'Set linear scale' : 'Set logarithmic scale'}
        </Button>
      </div>
    </div>
  );
};
