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
import { AppsInfoRequest, AppsInfoResponse, AppStatus, AppInfo } from "../../pb/dashboard_pb";
import { grpc } from "@improbable-eng/grpc-web";
import { Dashboard } from "../../pb/dashboard_pb_service";
import { retryFunc } from "../../utils/retry";
import { tryGetAppsList } from "./apps-list";

export type AppStatusDisplay = {
  name: string;
  description: string;
  status: string;
};

export function useStreamStatus(params: { appId?: string }): AppInfo.AsObject[] {
  const { appId = "" } = params;
  const [appsStatus, setAppsStatus] = useState<AppInfo.AsObject[]>([]);
  const [isStreaming, setIsStreaming] = useState(false);
  const [initialized, setInitialized] = useState(false);
  let currentAppsStatus: AppInfo.AsObject[] = [];

  // retry mechanism if client connection failed
  let client: grpc.Client<grpc.ProtobufMessage, grpc.ProtobufMessage> | null = null;

  const setCallbacks = (client: grpc.Client<grpc.ProtobufMessage, grpc.ProtobufMessage>) => {
    client.onEnd(async (status: grpc.Code, statusMessage: string, trailers: grpc.Metadata) => {
      setAppsStatus([]);
      let streaming = false;
      // eslint-disable-next-line @typescript-eslint/no-use-before-define
      await retryFunc(tryStreamStatus);
      streaming = true;
      setIsStreaming(streaming);
    });

    client.onMessage((message: grpc.ProtobufMessage) => {
      const appsStatusObject = message.toObject() as AppsInfoResponse.AsObject;
      appsStatusObject.appsList.forEach((newStatus) => {
        const appStatusIndex = currentAppsStatus.findIndex((m) => m.id === newStatus.id);

        // make new status object for unseen app name
        if (appStatusIndex === -1) {
          currentAppsStatus = [...currentAppsStatus, newStatus];
        } else {
          // push new status to existing app
          currentAppsStatus[appStatusIndex] = newStatus;
        }
      });

      setAppsStatus([...currentAppsStatus]);
    });
    setInitialized(true);
  };
  const tryStreamStatus = async () => {
    // block until AppsList grpc endpoint is live
    await retryFunc(tryGetAppsList);
    client = grpc.client(Dashboard.AppsInfo, {
      host: process.env.REACT_APP_DASHBOARD_GRPC_WEB_URL || "http://localhost:8081/api",
    });

    if (!client) console.log("error creating AppsInfo streaming client");

    client.start();
    const request = new AppsInfoRequest();
    request.setFilterAppId(appId);

    if (!initialized) {
      setCallbacks(client);
    }
    if (client) {
      try {
        await client.send(request);
      } catch (error) {
        console.log("error trying to stream app status: ", error);
        throw error;
      }
    }
  };

  useEffect(() => {
    if (!isStreaming || !initialized) {
      retryFunc(tryStreamStatus).then(() => setIsStreaming(true));
    }
  }, [isStreaming, initialized]);

  return appsStatus;
}

// Construct a reverse map from proto to make sure AppStatus number is the correct type in string
export const AppStatusNumberToStringMap = Object.values(AppStatus).reduce((map, statusNumber, index) => {
  map[statusNumber] = Object.keys(AppStatus)[index];
  return map;
}, {});
