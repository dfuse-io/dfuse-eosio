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

package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

var runNetworkErrorsTest = os.Getenv("RUN_NETWORK_TESTS") != ""

func TestBrokenPipe(t *testing.T) {
	if !runNetworkErrorsTest {
		t.Skip("manual invocation, would need to be properly turned into a valid unit test")
	}

	// TODO: Start a local serve handler

	client := &http.Client{Transport: &http.Transport{
		Dial: dialFactory,
	}}

	ctx, canceler := context.WithCancel(context.Background())
	request := makeRequest(t)

	resp, err := client.Do(request.WithContext(ctx))
	require.NoError(t, err)
	defer resp.Body.Close()

	canceler()

	// TODO: Asserts that broken pipe was correctly handled
}

func makeRequest(t *testing.T) *http.Request {
	request, err := http.NewRequest("GET", makeRequestURL(), nil)
	require.NoError(t, err)

	return request
}

func makeRequestURL() string {
	baseAddr := os.Getenv("DFUSE_REST_URL")
	if baseAddr == "" {
		baseAddr = "http://localhost:8080"
	}

	val := url.Values{}
	val.Set("account", "eosio.forum")
	val.Set("scope", "eosio.forum")
	val.Set("table", "proposal")
	val.Set("json", "true")

	return fmt.Sprintf("%s/v0/state/table?%s", baseAddr, val.Encode())
}

func dialFactory(network, addr string) (net.Conn, error) {
	conn, err := (&net.Dialer{}).Dial(network, addr)
	conn.(*net.TCPConn).SetLinger(0)

	return conn, err
}
