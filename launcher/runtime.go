package core

import (
	"time"

	pbbstream "github.com/dfuse-io/pbgo/dfuse/bstream/v1"
	"github.com/dfuse-io/dfuse-eosio/metrics"
	dmeshClient "github.com/dfuse-io/dmesh/client"
)

type RuntimeModules struct {
	SearchDmeshClient dmeshClient.SearchClient
	MetricManager     *metrics.Manager
	Launcher          *Launcher
}

type RuntimeConfig struct {
	BoxConfig *BoxConfig

	DmeshServiceVersion      string
	DmeshNamespace           string
	DataDir                  string
	MergerServingAddr        string
	AbiServingAddr           string
	RelayerServingAddr       string
	BlockmetaServingAddr     string
	TokenmetaServingAddr     string
	ShardSize                uint64
	StartBlock               uint64
	StopBlock                uint64
	FluxDBServingAddr        string
	IndexerServingAddr       string
	IndexerHTTPServingAddr   string
	ArchiveServingAddr       string
	ArchiveHTTPServingAddr   string
	LiveServingAddr          string
	RouterServingAddr        string
	RouterHTTPServingAddr    string
	DgraphqlHTTPServingAddr  string
	EoswsHTTPServingAddr     string
	DgraphqlGrpcServingAddr  string
	DashboardGrpcServingAddr string
	DashboardHTTPListenAddr  string
	EosqHTTPServingAddress   string
	KvdbDSN                  string
	FluxDSN                  string
	Protocol                 pbbstream.Protocol
	NodeExecutable           string
	NodeosAPIAddr            string
	MindreaderNodeosAPIAddr  string
	EosManagerHTTPAddr       string
	EosMindreaderHTTPAddr    string
	MindreaderGRPCAddr       string
	BootstrapDataURL         string
	NodeosTrustedProducer    string
	NodeosShutdownDelay      time.Duration
	NodeosExtraArgs          []string
	KvdbHTTPServingAddr      string
	NetworkID                string
}
