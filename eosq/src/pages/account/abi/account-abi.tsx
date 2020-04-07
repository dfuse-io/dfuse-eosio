import { observer } from "mobx-react"
import * as React from "react"
import { Cell, Grid } from "../../../atoms/ui-grid/ui-grid.component"
import { ContentLoaderComponent } from "../../../components/content-loader/content-loader.component"
import { JsonWrapper } from "../../../atoms/json-wrapper/json-wrapper"
import { BorderLessPanel } from "../../../atoms/panel/panel.component"
import { theme } from "../../../theme"
import { Text } from "../../../atoms/text/text.component"
import { Abi } from "@dfuse/client"

interface State {
  currentTable: string
  scope: string
}

interface Props {
  abi: Abi | null
}

@observer
export class AccountAbi extends ContentLoaderComponent<Props, State> {
  submitRequest = (tableName: string, scope: string) => {
    this.setState({ currentTable: tableName, scope })
  }

  renderContent = () => {
    if (!this.props.abi) {
      return <div />
    }

    return (
      <BorderLessPanel
        title={
          <Text fontSize={[4]} color={theme.colors.bleu8}>
            Contract ABI
          </Text>
        }
      >
        <Cell px={[4]}>
          <JsonWrapper>{JSON.stringify(this.props.abi, null, "   ")}</JsonWrapper>
        </Cell>
      </BorderLessPanel>
    )
  }

  render() {
    return <Grid>{this.renderContent()}</Grid>
  }
}
