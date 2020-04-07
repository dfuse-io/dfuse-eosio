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

import { withRouter } from "react-router";
import { Route, Switch, Redirect } from "react-router-dom";
import { RouteComponentProps } from "react-router";
import React from "react";
import { Paths } from "./paths";
import { HomePage } from "../../pages/home";
import { MockPage } from "../../pages/mock";
import { GrapiQLPage } from "../../pages/graphiql";

const RoutesBase: React.FC<RouteComponentProps> = () => {
  // FIXME: Commented for now
  // const appsList = useAppsList();
  // const routesToComponent = (appName: string) => <TemplatePage appId={appName} />;

  // {appsList?.map((app) => {
  //   const { id } = app;
  //   return <Route key={`app-page-${id}`} exact={true} path={`/${id}`} render={() => routesToComponent(id)} />;
  // })}

  return (
    <Switch>
      <Route exact={true} path={Paths.home} component={HomePage} />
      <Route exact={true} path={Paths.graphiql} component={GrapiQLPage} />
      <Route exact={true} path={Paths.mock} component={MockPage} />
      <Redirect to="/" />
    </Switch>
  );
};
export const Routes = withRouter(RoutesBase);
