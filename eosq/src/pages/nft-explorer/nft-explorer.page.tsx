import * as React from "react"
import { RouteComponentProps, Link } from "react-router-dom"
import styled from "@emotion/styled"
import { useNft, useNftFilters, NFT, NFTFilter } from "../../hooks/use-nft"

const PageWrapper = styled.div`
  display: grid;
  grid-template-columns: 250px auto;
`

const SideBar = styled.form`
  padding: 20px;
  background-color: #c4caff;
  display: flex;
  flex-direction: column;
  label,
  strong {
    font-size: 21px;
  }
  input {
    width: 18px;
    height: 18px;
  }
`

const Content = styled.div`
  padding: 20px;
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
      {name}&nbsp;
      <input type="checkbox" name={name} onChange={() => {}} />
    </label>
  </>
)

const RenderAssetItem: React.FC<{ asset: NFT }> = ({ asset }) => {
  const { id, owner, author, category, idata, mdata } = asset
  const imageLink = mdata.img.includes("http") ? mdata.img : `https://ipfs.io/ipfs/${mdata.img}`
  const imageSource = mdata.img ? imageLink : "/images/not-found.png"
  return (
    <Link to={`nft/${id}`}>
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
    </Link>
  )
}
export const NftExplorerPage: React.FC<Props> = () => {
  const assets: NFT[] = useNft("").resultOr([])
  const filters: NFTFilter = useNftFilters().resultOr({
    owners: [],
    authors: [],
    categories: []
  })
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
