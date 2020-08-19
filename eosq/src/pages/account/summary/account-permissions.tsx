import { observer } from "mobx-react"
import * as React from "react"
import { Cell, Grid } from "../../../atoms/ui-grid/ui-grid.component"
import { SubTitle, Text } from "../../../atoms/text/text.component"
import { t } from "i18next"
import {
  Account,
  KeyWeight,
  Permission,
  PermissionLevelWeight,
  WaitWeight,
  LinkedPermission
} from "../../../models/account"
import { theme, styled } from "../../../theme"
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome"
import { Links } from "../../../routes"
import {
  faCaretDown,
  faCaretRight,
  faClock,
  faKey,
  faUser
} from "@fortawesome/free-solid-svg-icons"
// eslint-disable-next-line import/no-unresolved
import { IconDefinition } from "@fortawesome/fontawesome-common-types"
import Collapsible from "react-collapsible"
import { MonospaceTextLink } from "../../../atoms/text-elements/misc"
import { secondsToTime } from "@dfuse/explorer"
import { assignHierarchy, HierarchyData } from "../../../helpers/account.helpers"
import { AutorizationBox } from "../../../components/authorization-box/authorization-box.component"
import { SearchShortcut } from "../../../components/search-shortcut/search-shortcut"

interface Props {
  account: Account
}

const WrapperWithChilds: React.ComponentType<any> = styled(Grid)`
  &:after {
    content: " ";
    width: 0px;
    height: 0px;
    position: absolute;
    bottom: -6px;
    border-left: 6px solid transparent;
    border-right: 6px solid transparent;
    border-top: 6px solid ${(props) => props.theme.colors.grey4};
    left: 5px;
  }
`

const CellCorner: React.ComponentType<any> = styled(Cell)`
  position: absolute;
  top: 0px;
  right: 0px;
  width: 13px;
  height: 10px;
  border-left: 2px solid ${(props) => props.theme.colors.grey4};
  border-bottom: 2px solid ${(props) => props.theme.colors.grey4};
`

const CellBottomLine: React.ComponentType<any> = styled(Cell)`
  position: absolute;
  bottom: -20px;
  left: 10px;
  width: 16px;
  height: 20px;
  border-left: 2px solid ${(props) => props.theme.colors.grey4};
`

@observer
export class AccountPermissions extends React.Component<Props> {
  py = "15px"

  renderPermissionPill = (permission: Permission) => {
    return (
      <Cell>
        <Grid py={this.py} pl="20px">
          <Text fontSize={[2]} color="text" fontWeight="800">
            {permission.perm_name}
          </Text>
        </Grid>
      </Cell>
    )
  }

  renderLinkedPermissions = (permission: Permission) => {
    return (this.props.account.linked_permissions || [])
      .filter(
        (linkedPermission: LinkedPermission) =>
          linkedPermission.permission_name === permission.perm_name
      )
      .map((linkedPermission: LinkedPermission, index: number) => {
        return (
          <Cell pb={this.py} key={index}>
            <SearchShortcut
              query={`auth:${this.props.account.account_name}@${permission.parent} action:${linkedPermission.action} receiver:${linkedPermission.contract}`}
            >
              <MonospaceTextLink
                color="link"
                to={Links.viewAccount({ id: linkedPermission.contract })}
              >
                {linkedPermission.contract}
              </MonospaceTextLink>
              <Text display="inline-block" color="text">
                @{linkedPermission.action}
              </Text>
            </SearchShortcut>
          </Cell>
        )
      })
  }

  renderPermissionGroup = (permission: Permission, idx: number, hasChilds: boolean) => {
    const Wrapper = hasChilds ? WrapperWithChilds : Grid
    return (
      <Wrapper
        mr={[3, 3, 0]}
        mb={[3]}
        bg={["#fff", "#fff", "grey1"]}
        border="1px solid"
        borderColor={theme.colors.grey3}
        gridTemplateColumns={["180px auto 250px"]}
        gridTemplateRows={["min-content"]}
        key={idx}
      >
        {hasChilds ? <CellBottomLine /> : null}
        <Cell height="100%">{this.renderPermissionPill(permission)}</Cell>
        {this.renderPermissionValues(permission)}
        <Cell
          height="100%"
          bg="bleu4"
          width="100%"
          justifySelf="left"
          alignSelf="left"
          pt={this.py}
          pl="15px"
        >
          {this.renderLinkedPermissions(permission)}
        </Cell>
      </Wrapper>
    )
  }

  renderPermissionValues = (permission: Permission) => {
    return (
      <Cell pb={this.py}>
        {(permission.required_auth.keys || []).map((keyWeight: KeyWeight, index: number) => {
          return this.renderPermissionContent(
            faKey,
            keyWeight.weight.toString(),
            permission.required_auth.threshold.toString(),
            index,
            () => {
              const { key } = keyWeight
              const query = `(data.auth.keys.key:${key} OR data.active.keys.key:${key} OR data.owner.keys.key:${key})`
              return (
                <SearchShortcut fixed={true} query={query}>
                  {key}
                </SearchShortcut>
              )
            }
          )
        })}
        {(permission.required_auth.accounts || []).map(
          (level: PermissionLevelWeight, index: number) => {
            return this.renderPermissionContent(
              faUser,
              level.weight.toString(),
              permission.required_auth.threshold.toString(),
              index,
              () => {
                return (
                  <SearchShortcut
                    query={`auth:${level.permission.actor}@${level.permission.permission}`}
                  >
                    <AutorizationBox authorization={level.permission} />
                  </SearchShortcut>
                )
              }
            )
          }
        )}
        {(permission.required_auth.waits || []).map((waitWeight: WaitWeight, index: number) => {
          return this.renderPermissionContent(
            faClock,
            waitWeight.weight.toString(),
            permission.required_auth.threshold.toString(),
            index,
            () => {
              return secondsToTime(waitWeight.wait_sec)
            }
          )
        })}
      </Cell>
    )
  }

  renderPermissionContent = (
    icon: IconDefinition,
    weight: string,
    threshold: string,
    index: number,
    renderValue: () => JSX.Element | string
  ): JSX.Element => {
    // TODO: humanize wait_sec
    return (
      <Grid pl="30px" pt={this.py} key={index} gridTemplateColumns={["80px 1fr"]}>
        <Cell>
          <Cell color="text" display="inline" pr="35px">
            +{weight}/{threshold}
          </Cell>
          <Cell display="inline">
            <FontAwesomeIcon color={theme.colors.text} icon={icon} />
          </Cell>
        </Cell>
        <Cell color="text" pl={[2]} justifySelf="left" alignSelf="left">
          {renderValue()}
        </Cell>
      </Grid>
    )
  }

  renderTitle(caret: IconDefinition) {
    return (
      <Cell p="20px" cursor="pointer">
        <Cell width="20px" cursor="pointer" display="inline-block" lineHeight="30px" pr={[2]}>
          <FontAwesomeIcon size="lg" color={theme.colors.bleu8} icon={caret} />
        </Cell>
        <SubTitle color={theme.colors.bleu8} display="inline-block" mb={[20]}>
          {t("account.permissions.title")}
        </SubTitle>
      </Cell>
    )
  }

  renderFromHierarchy(hierarchyData: HierarchyData[]) {
    return (
      <Cell>
        {hierarchyData.map((entry: HierarchyData, index: number) => {
          return (
            <Grid
              minWidth={["1100px"]}
              key={index}
              gridTemplateColumns={this.getGridTemplateColumns(entry.depth)}
            >
              {[...Array(entry.depth)].fill(1).map((_: number, idx: number) => {
                return this.renderLine(entry, idx, index * 100 + idx)
              })}
              {this.renderPermissionGroup(
                entry.permission,
                index * 100 + entry.depth,
                entry.hasChilds
              )}
            </Grid>
          )
        })}
      </Cell>
    )
  }

  renderLine = (hierarchyDataEntry: HierarchyData, index: number, largeIndex: number) => {
    return (
      <Cell key={largeIndex}>
        {index + 1 === hierarchyDataEntry.depth ? <CellCorner /> : null}
        <Cell
          borderRight={
            hierarchyDataEntry.parentDepths.includes(index)
              ? `2px solid  ${theme.colors.grey4}`
              : "2px solid transparent"
          }
          height="100%"
          width="13px"
        />
      </Cell>
    )
  }

  getGridTemplateColumns(depth: number) {
    const columns = [...Array(depth)]
      .fill(1)
      .map((_: number) => {
        return "24px"
      })
      .join(" ")

    return `${columns} 1fr`
  }

  render() {
    const hierarchyData = assignHierarchy(this.props.account.permissions, [])
    return (
      <Cell bg="white">
        <Collapsible
          trigger={this.renderTitle(faCaretRight)}
          triggerWhenOpen={this.renderTitle(faCaretDown)}
        >
          <Cell p="20px">
            <Cell
              bg={["#f2f5f9", "#f2f5f9", "#fff"]}
              borderRadius="0px"
              mt={[3]}
              overflow="hidden"
              overflowX="auto"
              p={[3, 3, 0]}
              border={["1px solid #ddd", "1px solid #ddd", "0px solid #ccc"]}
            >
              <Cell minWidth="800px" width="100%">
                {this.renderFromHierarchy(hierarchyData)}
              </Cell>
            </Cell>
          </Cell>
        </Collapsible>
      </Cell>
    )
  }
}
