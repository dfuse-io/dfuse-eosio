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
import { AppsListRequest } from '../../pb/dashboard_pb';
import * as PbDashboard from '../../pb/dashboard_pb';

const client = new DashboardClient(
  process.env.REACT_APP_DASHBOARD_GRPC_WEB_URL || 'http://localhost:8081/api'
);

export const getAppsList = async (): Promise<PbDashboard.AppsListResponse.AsObject | null> => {
  const request = new AppsListRequest();
  const res = await new Promise<PbDashboard.AppsListResponse.AsObject | null>(
    (resolve, reject) => {
      client.appsList(
        request,
        (
          err: ServiceError | null,
          response: PbDashboard.AppsListResponse | null
        ) => {
          if (err || !response) {
            reject(err);
          }
          resolve(response ?.toObject());
        }
      );
    }
  );
  return res;
};

export const tryGetAppsList = async () => {
  const res = await getAppsList();
  if (!res || !res.appsList || res.appsList.length <= 0)
    throw new Error('apps list empty');
};
