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

package eosws

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/tidwall/gjson"
	"go.uber.org/zap"

	"github.com/dfuse-io/dfuse-eosio/eosws/metrics"
	"github.com/dfuse-io/dfuse-eosio/eosws/wsmsg"
)

type PriceHub struct {
	CommonHub
}

func (ws *WSConn) onGetPrice(ctx context.Context, msg *wsmsg.GetPrice) {
	if msg.Listen {
		ws.priceHub.Subscribe(ctx, msg, ws)
		return
	}

	if msg.Fetch {
		out := ws.priceHub.Last()
		if out == nil {
			ws.EmitErrorReply(ctx, msg, AppPriceNotReadyError(ctx))
			return
		}
		metrics.DocumentResponseCounter.Inc()
		ws.EmitReply(ctx, msg, out)
	}
}

func NewPriceHub() *PriceHub {
	return &PriceHub{
		CommonHub: CommonHub{name: "Price"},
	}
}

func (h *PriceHub) Launch(ctx context.Context) {
	errorCount := 0
	for {
		price, err := GetBinanceData()
		if err != nil {
			errorCount += 1
			if errorCount >= 5 {
				zlog.Error("fetching price failed more than 5 times in a row", zap.Error(err), zap.Int("error_count", errorCount))
			}
			time.Sleep(5 * time.Second)
			continue
		}
		errorCount = 0

		previousPrice := h.Last()
		if previousPrice == nil || price.Data.Price != previousPrice.(*wsmsg.PriceResp).Data.Price {
			h.SetLast(price)
			h.EmitAll(ctx, price)
		}
		time.Sleep(5 * time.Second)
	}
}

func GetBinanceData() (d *wsmsg.PriceResp, err error) {
	resp, err := http.Get("https://api.binance.com/api/v1/ticker/24hr?symbol=EOSUSDT")
	if err != nil {
		return
	}
	defer resp.Body.Close()

	cnt, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	if resp.StatusCode != 200 {
		err = fmt.Errorf("binance api, status %d != 200: %s", resp.StatusCode, string(cnt))
		return
	}

	price := gjson.GetBytes(cnt, "lastPrice").Float()
	priceChangePercent := gjson.GetBytes(cnt, "priceChangePercent").Float()

	d = &wsmsg.PriceResp{}
	d.Data.Price = price
	d.Data.Variation = priceChangePercent
	d.Data.Symbol = "EOSUSDT"
	d.Data.LastUpdated = time.Now()

	return
}
