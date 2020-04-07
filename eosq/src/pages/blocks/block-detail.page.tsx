import { observer } from "mobx-react"
import * as React from "react"
import { Cell } from "../../atoms/ui-grid/ui-grid.component"
import { BlockHeader } from "./blocks-detail-header"
import { RouteComponentProps } from "react-router-dom"
import { BlockTransactions } from "./block-transactions"
import { PageContainer } from "../../components/page-container/page-container"

interface Props extends RouteComponentProps<any> {}
@observer
export class BlockDetailPage extends React.Component<Props> {
  render() {
    return (
      <Cell>
        <BlockHeader {...this.props} />
        <PageContainer>
          <BlockTransactions {...this.props} />
        </PageContainer>
      </Cell>
    )
  }
}
