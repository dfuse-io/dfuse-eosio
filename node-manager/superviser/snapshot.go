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

	"github.com/dfuse-io/dstore"

	"github.com/dfuse-io/node-manager/metrics"
	eos "github.com/eoscanada/eos-go"
	"go.uber.org/zap"
)

func (s *NodeosSuperviser) TakeSnapshot(snapshotStore dstore.Store, numberOfSnapshotsToKeep int) error {
	s.Logger.Info("asking nodeos API to create a snapshot")
	api := s.api
	snapshot, err := api.CreateSnapshot(context.Background())
	if err != nil {
		return fmt.Errorf("api call failed: %s", err)
	}

	filename := fmt.Sprintf("%010d-%s-snapshot.bin", eos.BlockNum(snapshot.HeadBlockID), snapshot.HeadBlockID)

	s.Logger.Info("saving state snapshot", zap.String("destination", filename))
	fileReader, err := os.Open(snapshot.SnapshotName)
	if err != nil {
		return fmt.Errorf("cannot open snapshot file: %s", err)
	}
	defer fileReader.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	err = snapshotStore.WriteObject(ctx, filename, fileReader)
	if err != nil {
		return fmt.Errorf("cannot write snapshot to store: %s", err)
	}

	metrics.NodeosSuccessfulSnapshots.Inc()

	if numberOfSnapshotsToKeep > 0 {
		err := cleanupSnapshots(snapshotStore, numberOfSnapshotsToKeep)
		if err != nil {
			s.Logger.Warn("cannot cleanup snapshots", zap.Error(err))
		}
	}

	return os.Remove(snapshot.SnapshotName)
}

func cleanupSnapshots(snapshotStore dstore.Store, keep int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	var snapshots []string
	err := snapshotStore.Walk(ctx, "", "", func(filename string) (err error) {
		snapshots = append(snapshots, filename)
		return nil
	})
	if err != nil {
		return err
	}

	if len(snapshots) <= keep {
		return nil
	}

	for _, s := range snapshots[:len(snapshots)-keep] {
		err := snapshotStore.DeleteObject(ctx, s)
		if err != nil {
			return err
		}
	}
	return nil

}

func findLatestSnapshotName(snapshotStore dstore.Store) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	latestSnapshot := ""
	err := snapshotStore.Walk(ctx, "", "", func(filename string) (err error) {
		latestSnapshot = filename
		return nil
	})

	if err != nil {
		return "", fmt.Errorf("unable to find latest snapshot: %s", err)
	}
	return latestSnapshot, nil

}

func (s *NodeosSuperviser) downloadSnapshotFile(snapshotName string, snapshotStore dstore.Store) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	reader, err := snapshotStore.OpenObject(ctx, snapshotName)
	if err != nil {
		return "", fmt.Errorf("cannot get snapshot from gstore: %s", err)
	}
	defer reader.Close()

	os.MkdirAll(s.snapshotsDir, 0755)
	snapshotPath := filepath.Join(s.snapshotsDir, snapshotName)
	w, err := os.Create(snapshotPath)
	if err != nil {
		return "", err
	}
	defer w.Close()

	_, err = io.Copy(w, reader)
	if err != nil {
		return "", err
	}
	return snapshotPath, nil

}

func (s *NodeosSuperviser) RestoreSnapshot(snapshotName string, snapshotStore dstore.Store) error {
	if snapshotStore == nil {
		return fmt.Errorf("trying to get snapshot store, but instance is nil, have you provided --snapshot-store-url flag?")
	}

	if snapshotName == "latest" {
		var err error
		snapshotName, err = findLatestSnapshotName(snapshotStore)
		if err != nil {
			return err
		}
	}

	if snapshotName == "" {
		s.Logger.Warn("Cannot find latest snapshot, will replay from blocks.log")
		s.snapshotRestoreFilename = ""
	} else {
		s.Logger.Info("getting snapshot from store", zap.String("snapshot_name", snapshotName))
		snapshotPath, err := s.downloadSnapshotFile(snapshotName, snapshotStore)
		if err != nil {
			return err
		}
		s.snapshotRestoreFilename = snapshotPath
		s.snapshotRestoreOnNextStart = true
	}

	err := s.removeState()
	if err != nil {
		return err
	}
	err = s.removeReversibleBlocks()
	if err != nil {
		return err
	}

	if s.HandlePostRestore != nil {
		s.HandlePostRestore()
	}

	return nil
}
