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

package kdvbloader

type Loader interface {
	StopBeforeBlock(uint64)

	BuildPipelineLive(bool) error
	BuildPipelineBatch(startBlock uint64, beforeStart uint64)
	BuildPipelinePatch(startBlock uint64, beforeStart uint64)

	Launch()
	Healthy() bool

	// Shutter related
	Shutdown(err error)
	OnTerminated(f func(error))
	Terminated() <-chan struct{}
	Err() error
}
