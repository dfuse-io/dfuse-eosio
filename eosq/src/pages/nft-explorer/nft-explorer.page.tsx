import * as React from "react"
import { RouteComponentProps } from "react-router-dom"
import { useNft, useNftFilters, NFT, NFTFilter } from "../../hooks/use-nft"
import styled from "@emotion/styled"

const PageWrapper = styled.div`
  display: grid;
  grid-template-columns: 250px auto;
`

const SideBar = styled.form`
  display: flex;
  flex-direction: column;
`

const Content = styled.div`
  display: grid;
  grid-gap: 20px;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
`

const Card = styled.table`
  max-width: 200px;
  .imageContainer {
    width: 200px;
    height: 300px;
    display: flex;
    align-items: center;
    justify-content: center;
    img {
      max-width: 100%;
      max-height: 100%;
      width: auto;
      height: auto;
    }
  }
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
  const imageLink = mdata.img.includes("http") ? mdata.img : `https://ipfs.io/ipfs/${mdata.img}`
  const imageSource = mdata.img ? imageLink : "/images/not-found.png"
  return (
    <Card>
      <tbody>
        <div className="imageContainer">
          <img src={imageSource} alt={mdata.name!} />
        </div>
        <tr>ID: {id}</tr>
        <tr>Owner: {owner}</tr>
        <tr>Author: {author}</tr>
        <tr>Category: {category}</tr>
      </tbody>
    </Card>
  )
}
export const NftExplorerPage: React.FC<Props> = () => {
  const assets: NFT[] = useNft("").resultOr([])
  const filters: NFTFilter = useNftFilters().resultOr({
    owners: [],
    authors: [],
    categories: []
  })
  console.log(filters)
  return (
    <PageWrapper>
      <SideBar>
        <strong>Owner</strong>
        {filters.owners.map((owner) => (
          <FilterCheckbox name={owner} />
        ))}
        <br />
        <strong>Author</strong>
        {filters.authors.map((author) => (
          <FilterCheckbox name={author} />
        ))}
        <br />
        <strong>Category</strong>
        {filters.categories.map((category) => (
          <FilterCheckbox name={category} />
        ))}
        <br />
      </SideBar>
      <Content>
        {assets.map((asset) => (
          <RenderAssetItem asset={asset} />
        ))}
      </Content>
    </PageWrapper>
  )
}
