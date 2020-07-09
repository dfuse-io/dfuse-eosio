import { DbOp, TableOp } from "@dfuse/client"
import * as React from "react"
import { Cell } from "../../atoms/ui-grid/ui-grid.component"
import { FormattedText } from "../formatted-text/formatted-text"
import { t } from "i18next"
import { theme, styled } from "../../theme"
import { MonospaceTextWrap } from "../../atoms/text-elements/misc"
import { JsonWrapper } from "@dfuse/explorer"

interface Props {
  dbops: DbOp[]
  tableops: TableOp[]
}

const EditIndicator: React.ComponentType<any> = styled(Cell)`
  position: absolute;
  left: -25px;
  width: 10px;
  text-align: center;
  top: -2px;
`

export class DBOperations extends React.Component<Props> {
  renderContentNew(dbop: DbOp) {
    if (dbop.new!.json) {
      return <JsonWrapper>{JSON.stringify(dbop.new!.json, null, "   ")}</JsonWrapper>
    }

    return dbop.new!.hex
  }

  renderContentOld(dbop: DbOp) {
    if (dbop.old!.json) {
      return <JsonWrapper>{JSON.stringify(dbop.old!.json, null, "   ")}</JsonWrapper>
    }

    return dbop.old!.hex
  }

  renderTableOps() {
    return this.props.tableops.map((tableop: TableOp, index: number) => {
      const [, scope, table] = tableop.path.split("/")

      const fields = [
        {
          name: "operation",
          type: "bold",
          value: t(`transaction.tableops.operations.${tableop.op}`)
        },
        {
          name: "table",
          type: "searchShortcut",
          value: table,
          query: `db.table:${table}`
        },
        {
          name: "scope",
          type: "searchShortcut",
          value: scope,
          query: `db.table:${table}/${scope}`
        }
      ]

      return (
        <Cell pb={[2]} pt={[1]} key={index}>
          <FormattedText i18nKey="transaction.tableops.label" fields={fields} fontSize="12px" />
        </Cell>
      )
    })
  }

  renderDBOps() {
    return this.props.dbops.map((dbop: DbOp, index: number) => {
      const fields = [
        { name: "operation", type: "bold", value: t(`transaction.dbops.operations.${dbop.op}`) },
        {
          name: "table",
          type: "searchShortcut",
          value: dbop.table,
          query: `db.table:${dbop.table}`
        },
        {
          name: "scope",
          type: "bold",
          value: dbop.scope
        },
        {
          name: "primaryKey",
          type: "bold",
          value: dbop.key
        }
      ]

      return (
        <Cell pb={[2]} pt={[1]} key={index}>
          <FormattedText i18nKey="transaction.dbops.label" fields={fields} fontSize="12px" />
          <Cell pt={[1]}>
            {dbop.old && dbop.old.hex ? (
              <MonospaceTextWrap color={theme.colors.editRemove} fontSize={[1]}>
                <EditIndicator>-</EditIndicator> {this.renderContentOld(dbop)}
              </MonospaceTextWrap>
            ) : null}
            {dbop.new && dbop.new.hex ? (
              <MonospaceTextWrap color={theme.colors.editAdd} fontSize={[1]}>
                <EditIndicator>+</EditIndicator> {this.renderContentNew(dbop)}
              </MonospaceTextWrap>
            ) : null}
          </Cell>
        </Cell>
      )
    })
  }

  render() {
    return (
      <Cell>
        {this.renderTableOps()}
        <br />
        {this.renderDBOps()}
      </Cell>
    )
  }
}
