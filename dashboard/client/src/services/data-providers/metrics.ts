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

import { useState, useEffect } from "react";
import { grpc } from "@improbable-eng/grpc-web";
import { Dashboard } from "../../pb/dashboard_pb_service";
import { normalizeDrift, timestampToTimeString, stringToMoment } from "../../utils/time";
import { DRIFT_RETENTION } from "../../utils/constants";
import { retryFunc } from "../../utils/retry";
import { AppsMetricsRequest, AppMetricsResponse, MetricType } from "../../pb/dashboard_pb";
import { tryGetAppsList } from "./apps-list";

export type metricData = {
  timestamp: string;
  value: number;
};

export type appMetric = {
  id: string;
  title: string;
  headBlockDrift: metricData[];
  headBlockNumber: number;
};

const processDriftData = (
  existingAppHeadBlockDrift: metricData[],
  appMetric: AppMetricsResponse.AsObject,
  maxCount: number
): metricData[] => {
  return appMetric.metricsList.reduce((resultArray: metricData[], d) => {
    let newResultArray = resultArray;
    if (resultArray.length > 0) {
      const firstValidPoint = resultArray.findIndex((data) => {
        const delta = d.timestamp!.seconds - stringToMoment(data.timestamp).unix();
        return delta <= DRIFT_RETENTION;
      });

      if (firstValidPoint > 0) {
        newResultArray = resultArray.slice(firstValidPoint);
      }
    }

    // if not type drift, don't store it
    if (d.type === MetricType.HEAD_BLOCK_TIME_DRIFT) {
      newResultArray.push({
        timestamp: timestampToTimeString(d.timestamp!.seconds),
        value: normalizeDrift(d.value),
      });
    }
    return newResultArray.slice(0, maxCount);
  }, existingAppHeadBlockDrift);
};

const processMetricEntry = (
  metricObject: AppMetricsResponse.AsObject,
  existingAppHeadBlockDrift: metricData[],
  currentMaxBlockNumbers: { [key: string]: number },
  maxCount: number
): appMetric => {
  let currentMaxBlockNumber = 0;
  if (metricObject.id in currentMaxBlockNumbers) {
    currentMaxBlockNumber = currentMaxBlockNumbers[metricObject.id];
  }
  const newMaxBlockNumber = Math.max(
    ...metricObject.metricsList.reduce(
      (resultArray: number[], d) => {
        if (d.type === MetricType.HEAD_BLOCK_NUMBER) {
          resultArray.push(d.value);
        }
        return resultArray;
      },
      [currentMaxBlockNumber]
    )
  );
  currentMaxBlockNumbers[metricObject.id] = newMaxBlockNumber;

  return Object.assign({}, metricObject, {
    headBlockDrift: processDriftData(existingAppHeadBlockDrift, metricObject, maxCount),
    headBlockNumber: newMaxBlockNumber,
  });
};

export function useStreamMetrics(params: { appId?: string; maxCount: number }): appMetric[] {
  const { appId = "", maxCount = 100 } = params;
  const [metrics, setMetrics] = useState<appMetric[]>([]);
  const [isStreaming, setIsStreaming] = useState(false);
  const [initialized, setInitialized] = useState(false);
  let currentMetrics: appMetric[] = [];
  const currentHeadBlockNumbers: { [key: string]: number } = {};

  // retry mechanism if client connection failed
  let client: grpc.Client<grpc.ProtobufMessage, grpc.ProtobufMessage> | null = null;

  const setCallbacks = (client: grpc.Client<grpc.ProtobufMessage, grpc.ProtobufMessage>) => {
    client.onEnd(async (status: grpc.Code, statusMessage: string, trailers: grpc.Metadata) => {
      let streaming = false;
      // eslint-disable-next-line @typescript-eslint/no-use-before-define
      await retryFunc(tryStreamMetrics);
      streaming = true;
      setIsStreaming(streaming);
    });

    client.onMessage((message: grpc.ProtobufMessage) => {
      const metricObject = message.toObject() as AppMetricsResponse.AsObject;

      const appMetricIndex = currentMetrics.findIndex((m) => m.id === metricObject.id);

      // make new metric object for unseen app name
      if (appMetricIndex === -1) {
        const newMetricEntry = processMetricEntry(metricObject, [], currentHeadBlockNumbers, maxCount);
        currentMetrics = [...currentMetrics, newMetricEntry];
      } else {
        // push metricData to existing app metric
        const newMetricEntry = processMetricEntry(
          metricObject,
          currentMetrics[appMetricIndex].headBlockDrift,
          currentHeadBlockNumbers,
          maxCount
        );
        currentMetrics[appMetricIndex] = newMetricEntry;
      }

      setMetrics([...currentMetrics]);
    });
    setInitialized(true);
  };

  const tryStreamMetrics = async () => {
    // block until AppsList grpc endpoint is live
    await retryFunc(tryGetAppsList);
    client = grpc.client(Dashboard.AppsMetrics, {
      host: process.env.REACT_APP_DASHBOARD_GRPC_WEB_URL || "http://localhost:8080/api",
    });

    if (!client) console.log("error creating streaming client");

    client.start();
    const request = new AppsMetricsRequest();
    request.setFilterAppId(appId);

    if (!initialized) {
      setCallbacks(client);
    }
    if (client) {
      try {
        await client.send(request);
      } catch (error) {
        console.log("error trying to stream metrics: ", error);
        throw error;
      }
    }
  };

  useEffect(() => {
    if (!isStreaming || !initialized) {
      retryFunc(tryStreamMetrics).then(() => setIsStreaming(true));
    }
  }, [isStreaming, initialized]);

  return metrics;
}
