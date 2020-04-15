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
	"path/filepath"
)

type StorageFolders struct {
	Mergedblocks string
	Oneblock     string
	Indexes      string
	Pitreos      string
	Snapshots    string
	AbiCache     string
}

type EOSNodeFolders struct {
	Config string
	Data   string
}

type EOSFolderStructure struct {
	Base        string
	StorageRoot string

	ManagerNode          *EOSNodeFolders
	MindreaderNode       *EOSNodeFolders
	Storage              *StorageFolders
	MindReaderWorkingDir string
	MergerWorkingDir     string
	KVDBWorkingDir       string
	FluxWorkingDir       string
	SearchWorkingDir     string
}

func NewEOSFolderStructure(userBase string) *EOSFolderStructure {
	base, err := filepath.Abs(userBase)
	if err != nil {
		userLog.Error("Unable to setup directory structure")
		return nil
	}

	managerNodeDir := filepath.Join(base, "managernode")
	mindreaderNodeDir := filepath.Join(base, "mindreadernode")
	storage := filepath.Join(base, "storage")

	return &EOSFolderStructure{
		Base:        base,
		StorageRoot: storage,

		MindReaderWorkingDir: filepath.Join(base, "mindreader"),
		MergerWorkingDir:     filepath.Join(base, "merger"),
		KVDBWorkingDir:       filepath.Join(base, "kvdb"),
		FluxWorkingDir:       filepath.Join(base, "flux"),
		SearchWorkingDir:     filepath.Join(base, "search"),
		ManagerNode: &EOSNodeFolders{
			Config: filepath.Join(managerNodeDir, "config"),
			Data:   filepath.Join(managerNodeDir, "data"),
		},
		MindreaderNode: &EOSNodeFolders{
			Config: filepath.Join(mindreaderNodeDir, "config"),
			Data:   filepath.Join(mindreaderNodeDir, "data"),
		},
		Storage: &StorageFolders{
			Mergedblocks: filepath.Join(storage, "merged-blocks"),
			Oneblock:     filepath.Join(storage, "one-blocks"),
			Indexes:      filepath.Join(storage, "indexes"),
			Pitreos:      filepath.Join(storage, "pitreos"),
			Snapshots:    filepath.Join(storage, "snapshots"),
			AbiCache:     filepath.Join(storage, "abicache"),
		},
	}
}

func (s EOSFolderStructure) ConfigDirs() []string {
	return []string{
		s.ManagerNode.Config,
		s.MindreaderNode.Config,
	}
}

func (s EOSFolderStructure) DataDirs() []string {
	return []string{
		s.StorageRoot,
		s.MindReaderWorkingDir,
		s.MergerWorkingDir,
		s.KVDBWorkingDir,
		s.SearchWorkingDir,
		s.ManagerNode.Data,
		s.MindreaderNode.Data,
		s.FluxWorkingDir,
	}
}
