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

package launcher

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"

	"go.uber.org/zap"
)

type Controller struct {
	nodeosCommandListenAddr string
}

func NewController(nodeosCommandURL string) *Controller {
	return &Controller{
		nodeosCommandListenAddr: nodeosCommandURL,
	}
}

func (c *Controller) StartNode() (string, error) {
	url := fmt.Sprintf("http://localhost%s/v1/resume?sync=true", c.nodeosCommandListenAddr)
	userLog.Debug("resuming node", zap.String("node_url", url))

	body := bytes.NewBufferString("{}")
	response, err := http.Post(url, "application/json", body)
	if err != nil {
		return "", fmt.Errorf("unable to contact manager API: %w", err)
	}

	defer response.Body.Close()
	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("unable to read manager API response: %w", err)
	}

	return string(data), nil
}

func (c *Controller) StopNode() (string, error) {
	url := fmt.Sprintf("http://localhost%s/v1/maintenance?sync=true", c.nodeosCommandListenAddr)
	userLog.Debug("pausing node", zap.String("node_url", url))

	body := bytes.NewBufferString("{}")
	response, err := http.Post(url, "application/json", body)
	if err != nil {
		return "", fmt.Errorf("unable to contact manager API: %w", err)
	}

	defer response.Body.Close()
	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("unable to read manager API response: %w", err)
	}

	return string(data), nil
}

func (c *Controller) NodeHealth() error {
	url := fmt.Sprintf("http://localhost%s/v1/healthz", c.nodeosCommandListenAddr)
	userLog.Debug("get node health", zap.String("node_url", url))
	_, err := http.Get(url)
	if err != nil {
		return err
	}

	return nil
}
