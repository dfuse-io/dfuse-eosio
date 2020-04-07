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
import {
  useStreamStatus,
  AppStatusDisplay,
  AppStatusNumberToStringMap
} from '../services/data-providers/status';

export const context = createContext<AppStatusDisplay[] | undefined>(undefined);

export function StatusProvider(props: { children: React.ReactNode }) {
  const [appsStatus, setAppsStatus] = useState<AppStatusDisplay[]>([]);

  const newAppsStatus = useStreamStatus({
    appId: ''
  });

  useEffect(() => {
    setAppsStatus(
      newAppsStatus.map(appStatus => ({
        name: appStatus.id,
        description: appStatus.description,
        status: AppStatusNumberToStringMap[appStatus.status]
      }))
    );
  }, [newAppsStatus]);

  return (
    <context.Provider value={appsStatus}>{props.children}</context.Provider>
  );
}

export const useStatus = () => useContext(context);
