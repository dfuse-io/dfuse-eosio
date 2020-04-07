import { observer } from "mobx-react"
import { Cell } from "../../../../atoms/ui-grid/ui-grid.component"
import * as React from "react"
import { Account } from "../../../../models/account"
import { DetailLine } from "../../../../atoms/pills/detail-line"
import { t } from "i18next"
import { compactString } from "../../../../helpers/formatters"
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
        <DetailLine compact={false} label={t("account.summary.creation_date")}>
          <Text>{formatDateFromString(account.created, false)}</Text>
        </DetailLine>
        {account.creator
          ? [
              <DetailLine key="0" compact={false} label={t("account.summary.created_by")}>
                <MonospaceTextLink to={Links.viewAccount({ id: account.creator.creator })}>
                  {account.creator.creator}
                </MonospaceTextLink>
              </DetailLine>,
              <DetailLine compact={false} key="1" label={t("account.summary.creation_trx_id")}>
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
