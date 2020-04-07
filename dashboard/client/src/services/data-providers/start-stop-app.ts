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

import { DashboardClient, ServiceError } from '../../pb/dashboard_pb_service';
import { StartAppRequest, StopAppRequest } from '../../pb/dashboard_pb';
import * as PbDashboard from '../../pb/dashboard_pb';

const client = new DashboardClient(
  process.env.REACT_APP_DASHBOARD_GRPC_WEB_URL || 'http://localhost:8080/api'
);

export const startApp = async (
  appId: string
): Promise<PbDashboard.StartAppResponse | null> => {
  const request = new StartAppRequest();
  request.setAppId(appId);

  try {
    const res = await new Promise<PbDashboard.StartAppResponse | null>(
      (resolve, reject) => {
        client.startApp(
          request,
          (
            err: ServiceError | null,
            response: PbDashboard.StartAppResponse | null
          ) => {
            if (err || !response) {
              reject(err);
            }
            resolve(response);
          }
        );
      }
    );
    return res;
  } catch (err) {
    console.log(err);
    return null;
  }
};

export const stopApp = async (
  appId: string
): Promise<PbDashboard.StopAppResponse | null> => {
  const request = new StopAppRequest();
  request.setAppId(appId);
  try {
    const res = await new Promise<PbDashboard.StopAppResponse | null>(
      (resolve, reject) => {
        client.stopApp(
          request,
          (
            err: ServiceError | null,
            response: PbDashboard.StopAppResponse | null
          ) => {
            if (err || !response) {
              reject(err);
            }
            resolve(response);
          }
        );
      }
    );
    return res;
  } catch (err) {

    console.log(err);
    return null;
  }
};
