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

import React, { createContext, useState, useEffect, useContext } from 'react';
import { getAppsList } from '../services/data-providers/apps-list';
import * as PbDashboard from '../pb/dashboard_pb';
import { AppStatusNumberToStringMap } from '../services/data-providers/status';
import { retryFunc } from '../utils/retry';

export const context = createContext<AppInfoToDisplay[] | null>(null);

export interface AppInfoToDisplay extends PbDashboard.AppInfo.AsObject {
  statusString: string;
}

export function AppsListProvider(props: { children: React.ReactNode }) {
  const [apps, setApps] = useState<AppInfoToDisplay[] | null>(null);

  const tryGetAppsList = async () => {
    const res = await getAppsList();
    if (!res || !res.appsList || res.appsList.length <= 0)
      throw new Error('apps list empty');
    const appsList = res.appsList
      .map(app => {
        return {
          ...app,
          statusString: AppStatusNumberToStringMap[app.status]
        };
      })
      .sort((a, b) => a.id.localeCompare(b.id));
    setApps(appsList);
  };

  // As soon as this provider is instanciated, it fetches it's data
  useEffect(() => {
    retryFunc(tryGetAppsList);
  }, []);

  return <context.Provider value={apps}>{props.children}</context.Provider>;
}

export const useAppsList = () => useContext(context);
