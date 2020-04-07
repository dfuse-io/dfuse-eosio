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
import { Cell, Grid } from '../../atoms/grid';
import { Header } from '../header/header';
import { SideMenu } from '../side-menu/side-menu';
import { styled } from '../../theme';

const MainWrapper = styled(Cell)`
  margin: 0px auto;
  min-width: 1200px;
  width: 80%;
`;
const ContentWrapper = styled(Grid)`
  grid-template-columns: 250px 1fr;
`;

class BaseLayout extends React.Component {
  render() {
    return (
      <MainWrapper>
        <Header />
        <ContentWrapper pb={[6]}>
          <SideMenu />
          <Cell>{this.props.children}</Cell>
        </ContentWrapper>
      </MainWrapper>
    );
  }
}

/**
 * Higher-order component that wraps the provided Component with an AuthenticatedLayout
 */
export function withBaseLayout<T>(
  Component: React.ComponentType<T>
): React.ComponentType<T> {
  return function withbaselayout2(props: T) {
    return (
      <BaseLayout>
        <Component {...props} />
      </BaseLayout>
    );
  };
}
