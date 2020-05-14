import * as React from "react"
import { RouteComponentProps } from "react-router-dom"
import { useNft, useNftOwners, NFT } from "../../hooks/use-nft"
import styled from "@emotion/styled"

const PageWrapper = styled.div`
  display: grid;
  grid-template-columns: 100px auto;
`

const SideBar = styled.form`
  display: table;
`
const Content = styled.div`
  display: grid;
  grid-gap: 20px;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
`

const Card = styled.table`
  max-width: 200px;
`

interface Props extends RouteComponentProps<any> {}

const FilterCheckbox: React.FC<{ name: string }> = ({ name }) => (
  <>
    <label htmlFor={name}>
      {name}
      <input type="checkbox" name={name} onChange={() => {}} />
    </label>
  </>
)

const RenderAssetItem: React.FC<{ asset: NFT }> = ({ asset }) => {
  const { id, owner, author, category, idata, mdata } = asset
  const imageSource = mdata.img ? `https://ipfs.io/ipfs/${mdata.img}` : "/images/not-found.png"
  return (
    <Card>
      <tbody>
        <img src={`https://ipfs.io/ipfs/${mdata.img}`} alt={mdata.name!} />
        <tr>ID: {id}</tr>
        <tr>Owner: {owner}</tr>
        <tr>Author: {author}</tr>
        <tr>Category: {category}</tr>
      </tbody>
    </Card>
  )
}
export const NftExplorerPage: React.FC<Props> = () => {
  const assets = useNft("").resultOr([])
  const owners = useNftOwners().resultOr([])
  console.log(owners)
  return (
    <PageWrapper>
      <SideBar>
        {owners.map((owner) => (
          <FilterCheckbox name={owner} />
        ))}
      </SideBar>
      <Content>
        {assets.map((asset) => (
          <RenderAssetItem asset={asset} />
        ))}
      </Content>
    </PageWrapper>
  )
}
