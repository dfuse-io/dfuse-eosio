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
	Id       string `json:"id"`
	Owner    string `json:"owner"`
	Author   string `json:"author"`
	Category string `json:"category,omitempty"`
	Idata    string `json:"idata,omitempty"`
	Mdata    string `json:"mdata,omitempty"`
}

func (r *Root) QuerySearchNfts(ctx context.Context, args SearchNftsArgs) ([]*NftAsset, error) {
	zlogger := logging.Logger(ctx, zlog)
	zlogger.Info("Searching NFTs")

	// TODO: filter results by query string of format "owner:xxx,xxx author:xxx,xxx category:xxx,xxx"
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
	return assets, nil
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
