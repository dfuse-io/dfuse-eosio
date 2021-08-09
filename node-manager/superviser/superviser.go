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
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ShinyTrinkets/overseer"
	"github.com/eoscanada/eos-go"
	"github.com/spf13/viper"
	nodeManager "github.com/streamingfast/node-manager"
	logplugin "github.com/streamingfast/node-manager/log_plugin"
	"github.com/streamingfast/node-manager/metrics"
	"github.com/streamingfast/node-manager/superviser"
	"go.uber.org/zap"
)

type NodeosSuperviser struct {
	*superviser.Superviser
	name string

	api          *eos.API
	blocksDir    string
	options      *SuperviserOptions
	snapshotsDir string

	chainID       eos.SHA256Bytes
	lastBlockSeen uint32

	producerHostname    string
	serverVersion       string
	serverVersionString string
	forceProduction     bool

	productionState             nodeManager.ProductionState
	productionStateLock         sync.Mutex
	productionStateLastProduced time.Time

	snapshotRestoreOnNextStart bool
	snapshotRestoreFilename    string

	headBlockUpdateFunc nodeManager.HeadBlockUpdater

	logger *zap.Logger
}

func (s *NodeosSuperviser) GetName() string {
	return "nodeos"
}

type SuperviserOptions struct {
	// LocalNodeEndpoint is the URL to reach the locally managed node (`http://localhost:8888` if empty)
	LocalNodeEndpoint string

	// ConfigPath points to the path where the config.ini lives (`/etc/nodeos` if empty)
	ConfigDir string

	// NodeosBinPath points to the file system location of the`nodeos` binary. Required.
	BinPath string

	// NodeosDataDir points to the location of the data dir. Required.
	DataDir string

	// Hostname is the hostname of the current machine. Leave blank for auto-detection.
	Hostname string

	// ProducerHostname is the hostname of the machine that should be
	// an active producer.  If `hostname` matches this name, the node
	// will *not* be started with `--pause-on-startup`. If you are not
	// managing a node that is configured to produce, you can safely
	// ignore this config entry.
	ProducerHostname string

	// Wheter or not to read the producing hostname from Viper config
	ProducerHostnameFromViper bool

	// TrustedProducer is the EOS account name of a Block Producer
	// we trust all blocks signed from.
	TrustedProducer string

	// ForceProduction will force isActiveProducer to return true and make mana
	ForceProduction bool

	// AdditionalArgs are parameters you want to pass down to `nodeos`
	// in addition to the ones `node manager` would add itself.  You're
	// better off putting long-running parameters in the `config.ini`
	// though.
	AdditionalArgs []string

	// ReadinessMaxLatency is the max delta between head block time and
	// now before /healthz starts returning success
	//ReadinessMaxLatency time.Duration

	// NoBlocksLog
	// NoBlocksLog is useful when extracting data from the chain only (mindreader) without calls to "get_block", etc.
	// It will *DELETE* the blocks.log and blocks.index on start (as well as on restore, restart, etc.)
	// When using that flag, the node CANNOT perform a pitreos backup, only restore (with exclude filters),
	// to prevent inadvertently deleting blocks.log from other nodes
	NoBlocksLog bool

	// Redirects all output to zlog instance configured for this process
	// instead of the standard console output
	LogToZap bool
}

func NewSuperviser(debugDeepMind bool, headBlockUpdateFunc nodeManager.HeadBlockUpdater, options *SuperviserOptions, logger *zap.Logger) (*NodeosSuperviser, error) {
	// Ensure process manager line buffer is large enough (50 MiB) for our Deep Mind instrumentation outputting lot's of text.
	overseer.DEFAULT_LINE_BUFFER_SIZE = 50 * 1024 * 1024

	s := &NodeosSuperviser{
		// The arguments field is actually `nil` because arguments are re-computed upon each start
		Superviser:          superviser.New(logger, options.BinPath, nil),
		api:                 eos.New(fmt.Sprintf("http://%s", options.LocalNodeEndpoint)),
		blocksDir:           filepath.Join(options.DataDir, "blocks"),
		producerHostname:    options.ProducerHostname,
		snapshotsDir:        path.Join(options.DataDir, "snapshots"),
		options:             options,
		forceProduction:     options.ForceProduction,
		headBlockUpdateFunc: headBlockUpdateFunc,
		logger:              logger,
	}

	s.RegisterLogPlugin(logplugin.LogPluginFunc(s.analyzeLogLineForStateChange))

	if options.LogToZap {
		s.RegisterLogPlugin(newToZapLogPlugin(debugDeepMind, logger))
	} else {
		s.RegisterLogPlugin(logplugin.NewToConsoleLogPlugin(debugDeepMind))
	}

	return s, nil
}

func (s *NodeosSuperviser) GetCommand() string {
	return s.options.BinPath + " " + strings.Join(s.getArguments(), " ")
}

func (s *NodeosSuperviser) HasData() bool {
	dir, err := ioutil.ReadDir(s.blocksDir)
	if err != nil || len(dir) == 0 {
		return false
	}

	return true
}

func (s *NodeosSuperviser) removeState() error {
	stateDir := path.Join(s.options.DataDir, "state")
	dir, err := ioutil.ReadDir(stateDir)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("cannot read state directory: %w", err)
	}

	for _, file := range dir {
		err = os.RemoveAll(path.Join(stateDir, file.Name()))
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("cannot delete state element: %w", err)
		}
	}

	return nil
}

func (s *NodeosSuperviser) removeBlocksLog() error {
	err := os.Remove(path.Join(s.blocksDir, "blocks.log"))
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("cannot delete blocks.log before starting: %w", err)
	}
	err = os.Remove(path.Join(s.blocksDir, "blocks.index"))
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("cannot delete blocks.index directory before starting: %w", err)
	}
	return nil
}

func (s *NodeosSuperviser) removeReversibleBlocks() error {
	err := os.RemoveAll(path.Join(s.blocksDir, "reversible"))
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("cannot delete blocks/reversible directory before starting: %w", err)
	}
	return nil
}

func (s *NodeosSuperviser) Start(options ...nodeManager.StartOption) error {
	s.Logger.Info("updating nodeos arguments before starting binary")
	s.Superviser.Arguments = s.getArguments()

	// Clears transient snapshot arguments now that we've computed binary arguments
	if s.snapshotRestoreOnNextStart {
		s.removeState()
		s.removeReversibleBlocks()
	}
	s.snapshotRestoreOnNextStart = false
	s.snapshotRestoreFilename = ""
	if s.options.NoBlocksLog {
		s.removeBlocksLog()
	}
	err := s.Superviser.Start(options...)
	if err != nil {
		return err
	}

	return nil
}

func (s *NodeosSuperviser) IsRunning() bool {
	isRunning := s.Superviser.IsRunning()
	isRunningMetricsValue := float64(0)
	if isRunning {
		isRunningMetricsValue = float64(1)
	}

	metrics.NodeosCurrentStatus.SetFloat64(isRunningMetricsValue)

	return isRunning
}

func (s *NodeosSuperviser) LastSeenBlockNum() uint64 {
	return uint64(s.lastBlockSeen)
}

func (s *NodeosSuperviser) ServerID() (string, error) {
	return os.Hostname()
}

func (s *NodeosSuperviser) getArguments() []string {
	s.maybeReloadProducerHostnameFromConfigFile()

	var args []string
	args = append(args, "--config-dir="+s.options.ConfigDir)
	args = append(args, "--data-dir="+s.options.DataDir)

	if !s.HasData() && !s.snapshotRestoreOnNextStart {
		args = append(args, "--genesis-json="+filepath.Join(s.options.ConfigDir, "genesis.json"))
	}

	if s.snapshotRestoreOnNextStart && s.snapshotRestoreFilename != "" {
		args = append(args, "--snapshot="+s.snapshotRestoreFilename)
	}

	if !s.IsActiveProducer() {
		args = append(args, "--pause-on-startup")
	}

	if s.options.TrustedProducer != "" {
		s.Logger.Info("running with trusted-producer mode", zap.String("trusted_producer", s.options.TrustedProducer))
		args = append(args, "--trusted-producer="+s.options.TrustedProducer)
	}

	args = append(args, s.options.AdditionalArgs...)

	return args
}

func (s *NodeosSuperviser) maybeReloadProducerHostnameFromConfigFile() {
	if !s.options.ProducerHostnameFromViper {
		return
	}

	_ = viper.ReadInConfig() // viper.WatchConfig broken on symlinks...
	s.producerHostname = viper.GetString("producer_hostname")
	s.Logger.Info("reloaded config", zap.String("hostname", s.options.Hostname), zap.String("producing_hostname", s.producerHostname))
}
