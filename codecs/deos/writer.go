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

package deos

import (
	"fmt"
	"io"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/dbin"
	pbbstream "github.com/dfuse-io/pbgo/dfuse/bstream/v1"
	"github.com/golang/protobuf/proto"
)

func blockWriterFactory(writer io.Writer) (bstream.BlockWriter, error) {
	return NewBlockWriter(writer)
}

// BlockWriter reads the dbin format where each element is assumed to be a `bstream.Block`.
type BlockWriter struct {
	src *dbin.Writer
}

func NewBlockWriter(writer io.Writer) (*BlockWriter, error) {
	dbinWriter := dbin.NewWriter(writer)
	err := dbinWriter.WriteHeader(pbbstream.Protocol_EOS.String(), 1)
	if err != nil {
		return nil, fmt.Errorf("unable to write file header: %s", err)
	}

	return &BlockWriter{
		src: dbinWriter,
	}, nil
}

func (w *BlockWriter) Write(block *bstream.Block) error {
	// FIXME: We could probably store directly the `PayloadBuffer` and reconstruct
	//        the `bstream.Block` from the payload directly (pbeos, pbdeth, etc.).
	//        This would shave a few bytes (around 74) per stored block that is
	//        duplicated somewhere in the proto messsage already.
	pbBlock, err := block.ToProto()
	if err != nil {
		return err
	}

	bytes, err := proto.Marshal(pbBlock)
	if err != nil {
		return fmt.Errorf("unable to marshal proto block: %s", err)
	}

	return w.src.WriteMessage(bytes)
}
