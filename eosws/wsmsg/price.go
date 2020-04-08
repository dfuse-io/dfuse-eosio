// Copyright 2020 dfuse Platform Inc.
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

package wsmsg

import "time"

func init() {
	RegisterIncomingMessage("get_price", GetPrice{})
	RegisterOutgoingMessage("price", PriceResp{})
}

type PriceResp struct {
	CommonOut
	Data struct {
		//Title string `json:"title"`
		Symbol      string    `json:"symbol"`
		Price       float64   `json:"price"`
		Variation   float64   `json:"variation"`
		LastUpdated time.Time `json:"last_updated"`
	} `json:"data"`
	Metadata struct {
		Timestamp int   `json:"timestamp"`
		Error     error `json:"error"`
	} `json:"metadata"`
}

type GetPrice struct {
	CommonIn
}

// Structs from Coin market cap's v2 ticket..

type TicketDataQuote struct {
	Price            float64 `json:"price"`
	Volume24h        float64 `json:"volume_24h"`
	MarketCap        float64 `json:"market_cap"`
	PercentChange1h  float64 `json:"percent_change_1h"`
	PercentChange24h float64 `json:"percent_change_24h"`
	PercentChange7d  float64 `json:"percent_change_7d"`
}
