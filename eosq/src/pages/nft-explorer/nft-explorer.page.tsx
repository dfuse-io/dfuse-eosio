import * as React from "react"
import { RouteComponentProps } from "react-router-dom"
import { Cell } from "../../atoms/ui-grid/ui-grid.component"
import { PageContainer } from "../../components/page-container/page-container"
import { useNft } from "../../hooks/use-nft"

interface Props extends RouteComponentProps<any> {}

export const NftExplorerPage: React.FC<Props> = () => {
  const response = useNft("")
  return (
    <PageContainer>
      <Cell>{JSON.stringify(response)}</Cell>
    </PageContainer>
  )
}
