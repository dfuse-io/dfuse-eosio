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

package superviser

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/streamingfast/dstore"
	node_manager "github.com/streamingfast/node-manager"
	"go.uber.org/zap"
)

var _ node_manager.BootstrapableChainSuperviser = (*NodeosSuperviser)(nil)

func (s *NodeosSuperviser) Bootstrap(bootstrapDataURL string) error {
	s.Logger.Info("bootstrapping blocks.log from pre-built data", zap.String("bootstrap_data_url", bootstrapDataURL))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	reader, _, _, err := dstore.OpenObject(ctx, bootstrapDataURL)
	if err != nil {
		return fmt.Errorf("cannot get snapshot from gstore: %w", err)
	}
	defer reader.Close()

	s.createBlocksLogFile(reader)
	return nil
}

func (s *NodeosSuperviser) createBlocksLogFile(reader io.Reader) error {
	err := os.MkdirAll(s.blocksDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("unable to create blocks log file: %w", err)
	}

	file, err := os.Create(filepath.Join(s.blocksDir, "blocks.log"))
	if err != nil {
		return fmt.Errorf("unable to create blocks log file: %w", err)
	}

	defer file.Close()

	_, err = io.Copy(file, reader)
	if err != nil {
		return fmt.Errorf("unable to create blocks log file: %w", err)
	}

	return nil
}
