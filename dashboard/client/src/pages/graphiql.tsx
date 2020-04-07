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

import React from "react";
import { withBaseLayout } from "../components/layout/layout";
import { WidgetBox } from "../components/widgets/widget-box";
import { Col, Row } from "antd";
import { WidgetContent } from "../components/widgets/widget-content";
import Iframe from "react-iframe";

const BaseGraphiQLPage: React.FC = () => {
  return (
    <>
      <Row gutter={[16, 16]}>
        <Col className="gutter-row" span={24} key={"col-drif-graph"}>
          <WidgetBox>
            <WidgetContent>
              <Iframe url="http://localhost:8080/graphiql" width="100%" height="900px" frameBorder={0} />
            </WidgetContent>
          </WidgetBox>
        </Col>
      </Row>
    </>
  );
};

export const GrapiQLPage = withBaseLayout(BaseGraphiQLPage);
