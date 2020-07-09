import * as React from "react"
import { Cell } from "../../../atoms/ui-grid/ui-grid.component"
import { SubTitle, Text } from "../../../atoms/text/text.component"
import { t } from "i18next"
import { theme } from "../../../theme"
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome"
import { faCaretRight, faCaretDown } from "@fortawesome/free-solid-svg-icons"
// eslint-disable-next-line import/no-unresolved
import { IconDefinition } from "@fortawesome/fontawesome-common-types"
import Collapsible from "react-collapsible"
import {
  UiTable,
  UiTableBody,
  UiTableCell,
  UiTableCellNarrow,
  UiTableHead,
  UiTableRow,
  UiTableRowAlternated
} from "../../../atoms/ui-table/ui-table.component"
import { MonospaceTextLink } from "../../../atoms/text-elements/misc"
import { Links } from "../../../routes"
import { SearchShortcut } from "../../../components/search-shortcut/search-shortcut"
import { LOGO_PLACEHOLDER } from "../../../helpers/airdrops-list"
import { UserBalance } from "../../../hooks/use-account-balances"

const Title: React.FC<{ icon: IconDefinition }> = ({ icon }) => (
  <Cell p="20px" cursor="pointer">
    <Cell width="20px" cursor="pointer" display="inline-block" lineHeight="30px" pr={[2]}>
      <FontAwesomeIcon size="lg" color={theme.colors.bleu8} icon={icon} />
    </Cell>
    <SubTitle color={theme.colors.bleu8} display="inline-block" mb={[20]}>
      {t("account.tokens.title")}
    </SubTitle>
  </Cell>
)

const TokenRow: React.FC<{ account: string; token: UserBalance }> = ({ account, token }) => (
  <UiTableRowAlternated>
    <UiTableCell width="120px" fontSize={[2]}>
      <Cell pl={[4]}>
        <img
          width="40px"
          height="40px"
          src={token.metadata.logo ? token.metadata.logo : LOGO_PLACEHOLDER}
          alt="token-logo"
        />
      </Cell>
    </UiTableCell>
    <UiTableCell textAlign="right" fontSize={[2]}>
      <Text ml={[4]} display="inline-block" title={token.symbol}>
        {token.metadata.name || token.symbol}
      </Text>
    </UiTableCell>
    <UiTableCellNarrow textAlign="right" fontSize={[2]}>
      <SearchShortcut
        position="left"
        query={`action:transfer (data.from:${account} OR data.to:${account}) receiver:${token.contract}`}
      >
        {token.balance}
      </SearchShortcut>
    </UiTableCellNarrow>
    <UiTableCell textAlign="right" fontSize={[2]}>
      <MonospaceTextLink to={Links.viewAccount({ id: token.contract })}>
        {token.contract}
      </MonospaceTextLink>
    </UiTableCell>
  </UiTableRowAlternated>
)

const TokenList: React.FC<Props> = ({ account, tokens }) => (
  <UiTable>
    <UiTableHead>
      <UiTableRow>
        <UiTableCellNarrow width="60px" fontSize={[2]} />
        <UiTableCell textAlign="right" fontSize={[2]}>
          {t("account.tokens.table.token")}
        </UiTableCell>

        <UiTableCellNarrow textAlign="right" fontSize={[2]}>
          {t("account.tokens.table.quantity")}
        </UiTableCellNarrow>
        <UiTableCell textAlign="right" fontSize={[2]}>
          {t("account.tokens.table.contract")}
        </UiTableCell>
      </UiTableRow>
    </UiTableHead>
    <UiTableBody>
      {tokens.map((token: UserBalance) => (
        <TokenRow key={token.contract + token.symbol} account={account} token={token} />
      ))}
    </UiTableBody>
  </UiTable>
)

type Props = { account: string; tokens: UserBalance[] }

export const AccountTokens: React.FC<Props> = ({ account, tokens }) => {
  return (
    <Cell bg="white">
      <Collapsible
        trigger={<Title icon={faCaretRight} />}
        triggerWhenOpen={<Title icon={faCaretDown} />}
      >
        <Cell
          bg={["#fff"]}
          borderRadius="0px"
          mt={[3]}
          overflow="hidden"
          overflowX="auto"
          p={[0]}
          border={["0px solid #ccc"]}
        >
          <Cell minWidth="800px" width="100%">
            <TokenList account={account} tokens={tokens} />
          </Cell>
        </Cell>
      </Collapsible>
    </Cell>
  )
}
