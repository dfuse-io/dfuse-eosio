import { observer } from "mobx-react"
import * as React from "react"
import { PagedTransactions } from "./paged-transactions"
import { Banner } from "../../components/banner/banner.component"
import { PageContainer } from "../../components/page-container/page-container"
import { RouteComponentProps } from "react-router"

interface Props extends RouteComponentProps<any> {}

@observer
export class TransactionsPage extends React.Component<Props, any> {
  render() {
    return (
      <PageContainer>
        <Banner />
        <PagedTransactions {...this.props} />
      </PageContainer>
    )
  }
}
