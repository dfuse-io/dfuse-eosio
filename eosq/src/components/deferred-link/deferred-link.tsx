import * as React from "react"
import { Cell, Grid } from "../../atoms/ui-grid/ui-grid.component"
import { Text } from "../../atoms/text/text.component"
import { theme, styled } from "../../theme"
import { faClock, faBan } from "@fortawesome/free-solid-svg-icons"
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome"
import { formatTransactionID } from "../../helpers/formatters"
import { Links } from "../../routes"
import { MonospaceTextLink } from "../../atoms/text-elements/misc"
import { translate, Trans } from "react-i18next"

const Container: React.ComponentType<any> = styled(Grid)`
  height: 26px;
`

const Header: React.ComponentType<any> = styled(Cell)`
  padding: 5px;
  width: 40px;
  height: 26px;
  border: 1px solid ${(props) => props.theme.colors.grey6};
  border-left: 6px solid ${(props) => props.theme.colors.grey6};
`

const Content: React.ComponentType<any> = styled(Cell)`
  margin-right: 30px;
  width: calc(100% - 35px);
  height: 26px;
  padding-left: 10px;
  padding-right: 5px;
  border-right: 1px solid ${(props) => props.theme.colors.grey4};
  border-top: 1px solid ${(props) => props.theme.colors.grey4};
  border-bottom: 1px solid ${(props) => props.theme.colors.grey4};
  background-color: white;
`

export const BaseDeferredLink: React.SFC<{
  transactionId: string
  operation: string
  delay: string
}> = ({ transactionId, operation, delay }) => {
  const bg = theme.colors.grey2
  const color = theme.colors.grey6
  let icon: any = faClock
  if (operation === "CREATE" || operation === "MODIFY_CREATE") {
    icon = faClock
  } else if (operation === "CANCEL" || operation === "MODIFY_CANCEL") {
    icon = faBan
  }

  const i18nKey =
    operation === "CREATE" || operation === "MODIFY_CREATE" || operation === "PUSH_CREATE"
      ? "transaction.deferred.create"
      : "transaction.deferred.cancel"

  return (
    <Container gridTemplateColumns={["40px auto"]} alignItems="center">
      <Header bg={bg} justifySelf="center" alignSelf="center">
        <Cell textAlign="center" m="auto" lineHeight="15px">
          <FontAwesomeIcon color={color} icon={icon} />
        </Cell>
      </Header>
      <Content alignSelf="center">
        <Text lineHeight="25px" fontSize={[1]}>
          <Trans
            i18nKey={i18nKey}
            values={{
              transactionId: formatTransactionID(transactionId).join(""),
              delay
            }}
            components={[
              <Text display="inline-block" fontSize={[1]} fontWeight="bold" mr={[1]} key="1">
                Created
              </Text>,
              <MonospaceTextLink
                fontSize={[1]}
                ml={[1]}
                mr={[1]}
                key="2"
                to={Links.viewTransaction({ id: transactionId })}
              >
                {formatTransactionID(transactionId).join("")}
              </MonospaceTextLink>,
              <Text
                display="inline-block"
                fontSize={[1]}
                fontWeight="bold"
                ml={[1]}
                mr={[1]}
                key="3"
              >
                {delay}
              </Text>
            ]}
          />
        </Text>
      </Content>
    </Container>
  )
}

export const DeferredLink = translate()(BaseDeferredLink)
