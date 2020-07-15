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

package abicodec

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/dfuse-io/dstore"
	"github.com/eoscanada/eos-go"
	"go.uber.org/zap"
)

type Cache interface {
	ABIAtBlockNum(account string, blockNum uint32) *ABICacheItem
	SetABIAtBlockNum(account string, blockNum uint32, abi *eos.ABI)
	RemoveABIAtBlockNum(account string, blockNum uint32)
	SaveState() error
	SetCursor(cursor string)
	GetCursor() string
	Export(baseURL string, filename string) error
}

type DefaultCache struct {
	Abis      map[string][]*ABICacheItem // from account to the ABIs in range
	Cursor    string                     `json:"cursor"`
	lock      sync.Mutex
	store     dstore.Store
	cacheName string
	dirty     bool
}

func NewABICache(store dstore.Store, cacheName string) (*DefaultCache, error) {
	zlog.Info("loading cache", zap.String("cache_name", cacheName))
	start := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	exist, err := store.FileExists(ctx, cacheName)
	if err != nil {
		return nil, fmt.Errorf("validating existance of cache file %s: %s", cacheName, err)
	}

	if !exist {
		zlog.Info("no cache file found. Creating empty cache")
		return &DefaultCache{
			store:     store,
			cacheName: cacheName,
			Abis:      make(map[string][]*ABICacheItem),
		}, nil

	}

	r, err := store.OpenObject(ctx, cacheName)
	defer r.Close()

	if err != nil {
		return nil, fmt.Errorf("openning cache file %s: %s", cacheName, err)
	}

	var cache *DefaultCache
	decoder := gob.NewDecoder(r)
	err = decoder.Decode(&cache)

	if err != nil {
		return nil, fmt.Errorf("decoding cache: %s", err)
	}

	cache.store = store
	cache.cacheName = cacheName

	zlog.Info("Cache loaded", zap.String("cache_name", cacheName), zap.Duration("in", time.Since(start)))
	return cache, nil

}

type ABICacheItem struct {
	ABI      *eos.ABI
	BlockNum uint32
}

func (c *DefaultCache) SetABIAtBlockNum(account string, blockNum uint32, abi *eos.ABI) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.dirty = true

	if accountItems, ok := c.Abis[account]; ok {

		replace := false
		var newItemIndex int
		for i := len(accountItems) - 1; i >= 0; i-- {
			item := accountItems[i]
			if item.BlockNum == blockNum {
				newItemIndex = i
				replace = true
				break
			}
			if item.BlockNum < blockNum {
				newItemIndex = i + 1
				break
			}
		}

		newItem := &ABICacheItem{
			BlockNum: blockNum,
			ABI:      abi,
		}
		if replace {
			accountItems[newItemIndex] = newItem
		} else {
			accountItems = append(accountItems[:newItemIndex], append([]*ABICacheItem{newItem}, accountItems[newItemIndex:]...)...)
		}
		c.Abis[account] = accountItems
		return
	}

	//this is the first abi for the account
	c.Abis[account] = []*ABICacheItem{
		{ABI: abi, BlockNum: blockNum},
	}

	return
}

func (c *DefaultCache) RemoveABIAtBlockNum(account string, blockNum uint32) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if accountItems, ok := c.Abis[account]; ok {

		for i := len(accountItems) - 1; i >= 0; i-- {
			item := accountItems[i]
			if item.BlockNum == blockNum {
				accountItems = append(accountItems[:i], accountItems[i+1:]...)
				c.Abis[account] = accountItems
				break
			}
		}
		return
	}
	return
}

func (c *DefaultCache) SaveState() error {
	c.lock.Lock()
	defer c.lock.Unlock()
	zlog.Info("saving cache", zap.String("cache_name", c.cacheName), zap.Bool("dirty", c.dirty))

	if !c.dirty {
		zlog.Info("not a dirty cache, no need to be save", zap.String("cache_name", c.cacheName), zap.Bool("dirty", c.dirty))
		return nil
	}

	start := time.Now()

	var b bytes.Buffer
	encoder := gob.NewEncoder(&b)
	err := encoder.Encode(&c)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	err = c.store.WriteObject(ctx, c.cacheName, bytes.NewReader(b.Bytes()))
	if err != nil {
		return fmt.Errorf("saving cache: %s", err)
	}

	c.dirty = false

	zlog.Info("Cache save", zap.String("cache_name", c.cacheName), zap.Duration("in", time.Since(start)))
	return nil
}

func (c *DefaultCache) ABIAtBlockNum(account string, blockNum uint32) *ABICacheItem {
	if abis, ok := c.Abis[account]; ok {

		for i := len(abis) - 1; i >= 0; i-- {
			a := abis[i]
			if a.BlockNum <= blockNum {
				return a
			}
		}
	}
	return nil //todo: should we return a "not found error"
}

func (c *DefaultCache) SetCursor(cursor string) {
	c.Cursor = cursor
}

func (c *DefaultCache) GetCursor() string {
	return c.Cursor
}

func (c *DefaultCache) Load(workerID string) (string, error) {
	return c.Cursor, nil
}

func (c *DefaultCache) Export(baseURL, filename string) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	zlog.Debug("exporting ABIs",
		zap.String("base_url", baseURL),
		zap.String("filename", filename),
	)

	store, err := dstore.NewStore(baseURL, "", "zstd", true)
	if err != nil {
		return fmt.Errorf("error creating export store: %w", err)
	}

	data, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("error marshalling default cache: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	err = store.WriteObject(ctx, filename, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("exporting cache: %w", err)
	}

	return nil
}

func getStoreInfo(storeUrl string) (baseURL, filename string, err error) {
	u, err := url.Parse(storeUrl)
	if err != nil {
		return "", "", fmt.Errorf("cannot pause upload url: %s", storeUrl)
	}
	filename = path.Base(u.Path)
	u.Path = path.Dir(u.Path)
	baseURL = strings.TrimRight(u.String(), "/")
	return
}

func (c *DefaultCache) Save(cursor string, workerID string) error {
	c.Cursor = cursor
	return nil
}
