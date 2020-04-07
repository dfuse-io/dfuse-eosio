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
import { DfuseLogo } from './dfuse-logo';
import styled from '@emotion/styled';
import { Grid } from '../../atoms/grid';
import { Link } from 'react-router-dom';
import { Links } from '../../components/routes/paths';
import { HeaderMetrics } from '../widgets/header-metrics';

const HeaderStyled = styled(Grid)`
  grid-template-columns: 250px 1fr;
  width: 100%;
  height: 85px;
  align-items: center;
  text-align: left;
  margin-bottom: 10px;
`;

const HeaderContentWrapper = styled.div`
  height: 100%;
  width: 100%;
  display: flex;
  flex-flow: column;
  justify-content: center;
`;

export function Header() {
  return (
    <HeaderStyled>
      <Link to={Links.home()} style={{ textDecoration: 'none' }}>
        <DfuseLogo />
      </Link>
      <HeaderContentWrapper>
        <HeaderMetrics />
      </HeaderContentWrapper>
    </HeaderStyled>
  );
}
