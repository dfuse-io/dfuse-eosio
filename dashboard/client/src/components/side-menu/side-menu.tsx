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
import { styled } from "../../theme";
import { colors } from "../../theme/colors";
import { Menu } from "antd";
import {
  // faThLarge,
  faTools,
  // faCog,
  faHome,
  faSearch,
} from "@fortawesome/free-solid-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { Cell } from "../../atoms/grid";
import { Link } from "react-router-dom";
import { Paths } from "../routes/paths";
// import { useAppsList } from '../../context/apps-list';

const IconWrapper = styled(Cell)`
  cursor: pointer;
  font-size: 18px;
  line-height: 18px;
  display: inline-box;
  width: 35px;
  align-items: flex-start;
  svg {
    filter: drop-shadow(0px 5px 8px ${colors.ternary400});
  }
  &:hover {
    color: ${colors.ternary300};
  }
`;

const MenuWrapper = styled(Cell)`
  .ant-menu:first-child {
    border-right: 0px;
  }
  .ant-menu {
    background: none;
  }

  .ant-menu-submenu,
  .ant-menu:first-child > .ant-menu-item {
    margin-bottom: 30px;
  }
  .ant-menu .ant-menu-submenu-title,
  .ant-menu:first-child > .ant-menu-item {
    padding-left: 0px !important;
    font-weight: 600;
    height: 25px;
    line-height: 25px;
    overflow: visible;
  }

  .ant-menu.ant-menu-sub .ant-menu-item {
    padding-left: 35px !important;
    height: 25px;
    line-height: 25px;
  }

  .ant-menu-submenu.unexpandable .ant-menu-submenu-title {
    pointer-events: none;
    i.ant-menu-submenu-arrow {
      display: none;
    }
  }
`;

export const SideMenu: React.FC = () => {
  return (
    <MenuWrapper>
      <Menu style={{ width: "230px" }} defaultSelectedKeys={["0"]} defaultOpenKeys={["apps"]} mode="inline">
        <Menu.Item key="home">
          <Link to={Paths.home}>
            <IconWrapper>
              <FontAwesomeIcon icon={faHome} />
            </IconWrapper>
            <span>HOME</span>
          </Link>
        </Menu.Item>
        <Menu.Item key="graphiql">
          <Link to={Paths.graphiql}>
            <IconWrapper>
              <FontAwesomeIcon icon={faTools} />
            </IconWrapper>
            <span>GRAPHiQL</span>
          </Link>
        </Menu.Item>
        <Menu.Item key="eosqElese">
          {/* TODO: Must come from some config provided by the server */}
          <a href="http://localhost:8080" target="_blank" rel="noopener noreferrer">
            <IconWrapper>
              <FontAwesomeIcon icon={faSearch} />
            </IconWrapper>
            <span>eosq</span>
          </a>
        </Menu.Item>
      </Menu>
    </MenuWrapper>
  );
};

// const appsList = useAppsList();
// return (
//   <SubMenu
//     className='unexpandable'
//     key='apps'
//     title={
//       <span>
//           <IconWrapper>
//             <FontAwesomeIcon icon={faThLarge} />
//           </IconWrapper>
//           <span>APPS</span>
//         </span>
//     }
//   >
//     {appsList?.map(app => (
//       <Menu.Item key={app.id}>
//         {app.title.charAt(0).toUpperCase() + app.title.slice(1)}
//       </Menu.Item>
//     ))}
//   </SubMenu>
// )
