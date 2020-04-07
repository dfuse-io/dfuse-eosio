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
  useStreamMetrics,
  appMetric
} from '../services/data-providers/metrics';

export const context = createContext<appMetric[] | undefined>(undefined);

export function MetricsProvider(props: { children: React.ReactNode }) {
  const [appMetrics, setAppMetrics] = useState<appMetric[]>([]);

  const metrics = useStreamMetrics({
    appId: '',
    maxCount: 100
  });
  useEffect(() => {
    setAppMetrics(metrics);
  }, [metrics]);

  return (
    <context.Provider value={appMetrics}>{props.children}</context.Provider>
  );
}

export const useMetrics = () => useContext(context);
