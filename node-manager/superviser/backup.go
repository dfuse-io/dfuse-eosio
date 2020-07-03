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
	"encoding/hex"
	"errors"

	"github.com/dfuse-io/node-manager/metrics"
	"github.com/dfuse-io/node-manager/superviser"

	pitreos "github.com/eoscanada/pitreos/lib"

	"go.uber.org/zap"
)

type NodeosBackupInfo struct {
	ChainIDStr          string `yaml:"chainIdStr"`
	ServerVersion       string `yaml:"serverVersion"`
	LastBlockSeen       uint32 `yaml:"lastBlockSeen"`
	ServerVersionString string `yaml:"serverVersionString"`
}

func (s *NodeosSuperviser) TakeBackup(backupTag string, backupStoreURL string) error {
	if s.options.NoBlocksLog {
		return errors.New("unable to take backup: refusing to take backup on an instance with option 'NoBlocksLog'")
	}

	if s.IsRunning() {
		return errors.New("unable to take backup: refusing to take backup while process is running")
	}

	p, err := superviser.GetPitreos(s.Logger, backupStoreURL, "blocks/blocks.log")
	if err != nil {
		return err
	}

	details := make(map[string]interface{})
	details["nodeosInfo"] = NodeosBackupInfo{
		ChainIDStr:          hex.EncodeToString(s.chainID),
		ServerVersion:       string(s.serverVersion),
		ServerVersionString: s.serverVersionString,
		LastBlockSeen:       s.lastBlockSeen,
	}

	s.Logger.Info("creating backup", zap.String("store_url", backupStoreURL), zap.String("tag", backupTag))
	err = p.GenerateBackup(s.options.DataDir, backupTag, details, pitreos.MustNewIncludeThanExcludeFilter(".*", ""))
	if err == nil {
		metrics.SuccessfulBackups.Inc()
	}

	return err
}

func (s *NodeosSuperviser) RestoreBackup(backupName, backupTag string, backupStoreURL string) error {
	if s.IsRunning() {
		return errors.New("unable to take backup: refusing to restore backup while process is running")
	}

	var appendonlyFiles []string
	var exclusionFilter string
	if s.options.NoBlocksLog {
		exclusionFilter = "blocks/blocks.(log|index)"
	} else {
		appendonlyFiles = append(appendonlyFiles, "blocks/blocks.log")
	}

	p, err := superviser.GetPitreos(s.Logger, backupStoreURL, appendonlyFiles...)
	if err != nil {
		return err
	}

	if backupName == "latest" {
		// FIXME: This logic should be moved up to the operator, so it's not repeated between each superviser!
		backupName, err = p.GetLatestBackup(backupTag)
		if err != nil {
			return err
		}
	}

	zlog.Info("restoring from pitreos", zap.String("backup_name", backupName), zap.Any("appendonly_files", appendonlyFiles), zap.String("exclusion_filter", exclusionFilter))
	err = p.RestoreFromBackup(s.options.DataDir, backupName, pitreos.MustNewIncludeThanExcludeFilter(".*", exclusionFilter))
	if s.HandlePostRestore != nil {
		s.HandlePostRestore()
	}
	return err
}
