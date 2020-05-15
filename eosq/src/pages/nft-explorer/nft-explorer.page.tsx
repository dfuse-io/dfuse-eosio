import React, { useState, useEffect } from "react"
import { RouteComponentProps, Link } from "react-router-dom"
import styled from "@emotion/styled"
import { useNft, useNftFilters, NFT, NFTFilter } from "../../hooks/nft"

const PageWrapper = styled.div`
  display: grid;
  grid-template-columns: 250px auto;
  grid-template-rows: minmax(820px, auto);
`

const SideBar = styled.form`
  padding: 20px;
  background-color: #c4caff;
  display: flex;
  flex-direction: column;
  height: 100%;
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

const FilterCheckbox: React.FC<{
  name: string
  value: string
  handleChange: ((event: React.ChangeEvent<HTMLInputElement>) => void) | undefined
}> = ({ name, value, handleChange }) => {
  return (
    <>
      <label htmlFor={value}>
        {value}&nbsp;
        <input type="checkbox" name={name} value={value} onChange={handleChange} />
      </label>
    </>
  )
}

const RenderAssetItem: React.FC<{ asset: NFT }> = ({ asset }) => {
  const { id, owner, author, category, idata, mdata } = asset
  let imageLink
  if (mdata && (mdata.img || mdata.backimg)) {
    if (mdata.img) {
      if (mdata.img.includes("http")) {
        imageLink = mdata.img
      } else {
        imageLink = `https://ipfs.io/ipfs/${mdata.img}`
      }
    } else if (mdata.backimg.includes("http")) {
      imageLink = mdata.backimg
    } else {
      imageLink = `https://ipfs.io/ipfs/${mdata.img}`
    }
  } else {
    imageLink = "/images/not-found.png"
  }
  return (
    <Link to={`nft/${id}`}>
      <Card>
        <tbody>
          <div className="imageContainer">
            <img src={imageLink} alt={mdata.name!} />
          </div>
          <tr>ID:&nbsp;{id}</tr>
          <tr>Owner:&nbsp;{owner}</tr>
          <tr>Author:&nbsp;{author}</tr>
          {category && <tr>Category:&nbsp;{category}</tr>}
        </tbody>
      </Card>
    </Link>
  )
}

export const NftExplorerPage: React.FC<Props> = () => {
  const [filters, setFilters] = useState<NFTFilter>({
    owners: [],
    authors: [],
    categories: [],
    id: ""
  })
  const allFilters: NFTFilter = useNftFilters()
  const assets = useNft(filters)

  console.log(filters)
  console.log(assets)

  const handleFilter: (event: React.ChangeEvent<HTMLInputElement>) => void = (e) => {
    const { checked, name, value } = e.target
    if (checked) {
      if (!filters[name]?.includes(value)) {
        const newFilters = {}
        newFilters[name] = filters[name]
        newFilters[name].push(value)
        setFilters({ ...filters, ...newFilters })
      }
    }
    if (!checked) {
      if (filters[name]?.includes(value)) {
        const newFilters = {}
        newFilters[name] = filters[name].filter((f: string) => f !== value)
        setFilters({ ...filters, ...newFilters })
      }
    }
  }

  return (
    <PageWrapper>
      <SideBar>
        <strong>Author</strong>
        {allFilters.authors.length > 0 &&
          allFilters.authors.map((author) => (
            <FilterCheckbox name="authors" value={author} handleChange={handleFilter} />
          ))}
        <br />
        <strong>Category</strong>
        {allFilters.categories.length > 0 &&
          allFilters.categories.map((category) => (
            <FilterCheckbox name="categories" value={category} handleChange={handleFilter} />
          ))}
        <br />
      </SideBar>
      <Content>
        {assets.length > 0 && assets.map((asset) => <RenderAssetItem asset={asset} />)}
      </Content>
    </PageWrapper>
  )
}
