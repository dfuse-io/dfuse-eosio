import * as React from "react"
import { TextLink, Text } from "../../atoms/text/text.component"

import { styled } from "../../theme"
import { Authorization } from "@dfuse/client"
import { Links } from "../../routes"
import { faShieldAlt } from "@fortawesome/free-solid-svg-icons"
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome"
import { Cell } from "../../atoms/ui-grid/ui-grid.component"

interface Props {
  authorization: Authorization
}

const AuthorizationContainer = styled.div`
  padding: 2px 10px !important;
  border: 1px solid #7a85ff;
  width: auto;
  box-sizing: border-box;
  display: inline-block;
  margin-top: -2px;
  position: relative;
`

export const AutorizationBox: React.FC<Props> = ({ authorization }) => (
  <AuthorizationContainer>
    <TextLink fontSize={[2]} to={Links.viewAccount({ id: authorization.actor })}>
      {authorization.actor}
    </TextLink>
    <Cell display="inline-block" p="0px 5px 0px 10px">
      <FontAwesomeIcon icon={faShieldAlt as any} />
    </Cell>
    <Text display="inline-block" fontWeight="600" fontSize={[2]}>
      {authorization.permission}
    </Text>
  </AuthorizationContainer>
)
