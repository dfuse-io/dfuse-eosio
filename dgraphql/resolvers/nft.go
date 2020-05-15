// Copyright 2019 dfuse Platform Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package resolvers

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/dfuse-io/logging"
)

type SearchNftsArgs struct {
	Query string
}

type NftAsset struct {
	Id_       string `json:"id"`
	Owner_    string `json:"owner"`
	Author_   string `json:"author"`
	Category_ string `json:"category,omitempty"`
	Idata_    string `json:"idata,omitempty"`
	Mdata_    string `json:"mdata,omitempty"`
}

func (r *Root) QuerySearchNfts(ctx context.Context, args SearchNftsArgs) (*NftAssets, error) {
	zlogger := logging.Logger(ctx, zlog)
	zlogger.Info("Searching NFTs")

	// TODO: filter results by query string of format "owner:xxx,xxx author:xxx,xxx category:xxx,xxx"
	jsonFile, err := os.Open("nft-mock.json")
	if err != nil {
		fmt.Println(err)
	}
	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	var assets []*NftAsset

	json.Unmarshal(byteValue, &assets)
	return &NftAssets{
		assets: assets,
	}, nil
}

type NftAssets struct {
	assets []*NftAsset
}

func (n *NftAssets) Assets(ctx context.Context) []*NftAsset {
	return n.assets
}

func (n *NftAsset) Id() string {
	return n.Id_
}

func (n *NftAsset) Owner() string {
	return n.Owner_
}

func (n *NftAsset) Author() string {
	return n.Author_
}

func (n *NftAsset) Category() string {
	return n.Category_
}

func (n *NftAsset) Idata() string {
	return n.Idata_
}

func (n *NftAsset) Mdata() string {
	return n.Mdata_
}

type NftFilters struct {
	Owners     []string
	Authors    []string
	Categories []string
}

func (r *Root) QueryNftFilters(ctx context.Context) (*NftFilters, error) {
	zlogger := logging.Logger(ctx, zlog)
	zlogger.Info("Searching NFTs")

	jsonFile, err := os.Open("nft-mock.json")
	// if we os.Open returns an error then handle it
	if err != nil {
		fmt.Println(err)
	}
	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	var assets []*NftAsset

	// we unmarshal our byteArray which contains our
	// jsonFile's content into 'users' which we defined above
	json.Unmarshal(byteValue, &assets)

	filters := &NftFilters{
		Owners:     []string{"pixas.wam"},
		Authors:    []string{"immortals", "darkcountrya", "gpk.topps"},
		Categories: []string{"immortals", "card", "series1"},
	}
	return filters, nil
}
