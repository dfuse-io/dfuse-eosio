import { observer } from "mobx-react"
import * as React from "react"
import { Account } from "../../../../models/account"
import { DetailLine, Cell, compactString } from "@dfuse/explorer"
import { t } from "i18next"

import { Text, TextLink } from "../../../../atoms/text/text.component"
import { MonospaceTextLink } from "../../../../atoms/text-elements/misc"
import { Links } from "../../../../routes"
import { formatDateFromString } from "../../../../helpers/moment.helpers"

interface Props {
  account: Account
}

@observer
export class DefaultAccountWidget extends React.Component<Props, any> {
  render() {
    const { account } = this.props

    return (
      <Cell pl={[4, 0, 0]} bg="white" pt={[4]}>
        <DetailLine variant="auto" label={t("account.summary.creation_date")}>
          <Text>{formatDateFromString(account.created, false)}</Text>
        </DetailLine>
        {account.creator
          ? [
              <DetailLine key="0" variant="auto" label={t("account.summary.created_by")}>
                <MonospaceTextLink to={Links.viewAccount({ id: account.creator.creator })}>
                  {account.creator.creator}
                </MonospaceTextLink>
              </DetailLine>,
              <DetailLine variant="auto" key="1" label={t("account.summary.creation_trx_id")}>
                <TextLink to={Links.viewTransaction({ id: account.creator.trx_id })}>
                  {compactString(account.creator.trx_id, 12, 0)}
                </TextLink>
              </DetailLine>
            ]
          : null}
      </Cell>
    )
  }
}
