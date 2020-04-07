import { observer } from "mobx-react"
import * as React from "react"
import { Banner } from "../../components/banner/banner.component"
import { PagedBlocks } from "./paged-blocks"
import { PageContainer } from "../../components/page-container/page-container"
import { RouteComponentProps } from "react-router"

interface Props extends RouteComponentProps<any> {}

@observer
export class BlocksPage extends React.Component<Props, any> {
  render() {
    return (
      <PageContainer>
        <Banner />
        <PagedBlocks {...this.props} />
      </PageContainer>
    )
  }
}
