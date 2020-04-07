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
import './App.css';
import { Router } from 'react-router-dom';
import { Routes } from './components/routes/routes';
import { history } from './services/history';
import { theme } from './theme';
import { ThemeProvider } from 'emotion-theming';
import { AppsListProvider } from './context/apps-list';
import { MetricsProvider } from './context/metrics';
import { StatusProvider } from './context/status';

function App() {
  return (
    <AppsListProvider>
      <StatusProvider>
        <MetricsProvider>
          <ThemeProvider theme={theme}>
            <Router history={history}>
              <Routes />
            </Router>
          </ThemeProvider>
        </MetricsProvider>
      </StatusProvider>
    </AppsListProvider>
  );
}

export default App;
