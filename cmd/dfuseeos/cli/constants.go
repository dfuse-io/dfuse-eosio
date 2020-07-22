package cli

import (
	pbbstream "github.com/dfuse-io/pbgo/dfuse/bstream/v1"
)

const (
	Protocol               pbbstream.Protocol = pbbstream.Protocol_EOS
	TrxdbDSN               string             = "badger://{dfuse-data-dir}/storage/trxdb"   //%s will be replaced by `<data-dir>`
	FluxDSN                string             = "badger://{dfuse-data-dir}/storage/statedb" //%s will be replaced by `<data-dir>/<flux-data-dir>
	MergedBlocksStoreURL   string             = "file://{dfuse-data-dir}/storage/merged-blocks"
	FilteredBlocksStoreURL string             = "file://{dfuse-data-dir}/storage/filtered-merged-blocks"
	IndicesStoreURL        string             = "file://{dfuse-data-dir}/storage/indexes"
	OneBlockStoreURL       string             = "file://{dfuse-data-dir}/storage/one-blocks"
	PitreosURL             string             = "file://{dfuse-data-dir}/storage/pitreos"
	SnapshotsURL           string             = "file://{dfuse-data-dir}/storage/snapshots"
	DmeshDSN               string             = "local://"
	DmeshServiceVersion    string             = "v1"
	NetworkID              string             = "eos-local"
	NodeosBinPath          string             = "nodeos"
	// Ports
	NodeManagerHTTPServingAddr  string = ":13008"
	MindreaderHTTPServingAddr   string = ":13009"
	MindreaderGRPCAddr          string = ":13010"
	RelayerServingAddr          string = ":13011"
	MergerServingAddr           string = ":13012"
	AbiServingAddr              string = ":13013"
	BlockmetaServingAddr        string = ":13014"
	ArchiveServingAddr          string = ":13015"
	ArchiveHTTPServingAddr      string = ":13016"
	LiveServingAddr             string = ":13017"
	RouterServingAddr           string = ":13018"
	RouterHTTPServingAddr       string = ":13019"
	KvdbHTTPServingAddr         string = ":13020"
	IndexerServingAddr          string = ":13021"
	IndexerHTTPServingAddr      string = ":13022"
	DgraphqlHTTPServingAddr     string = ":13023"
	DgraphqlGrpcServingAddr     string = ":13024"
	EoswsHTTPServingAddr        string = ":13026"
	ForkresolverServingAddr     string = ":13027"
	ForkresolverHTTPServingAddr string = ":13028"
	FluxDBServingAddr           string = ":13029"
	EosqHTTPServingAddr         string = ":13030"
	DashboardGrpcServingAddr    string = ":13031"
	FilteringRelayerServingAddr string = ":13032"
	TokenmetaGrpcServingAddr    string = ":14001"
	DashboardHTTPListenAddr     string = ":8081"
	APIProxyHTTPListenAddr      string = ":8080"
	MindreaderNodeosAPIAddr     string = ":9888"
	NodeosAPIAddr               string = ":8888"
	MetricsListenAddr           string = ":9102"

	DgraphqlAPIKey string = "web_0000"
	JWTIssuerURL   string = "null://dfuse"
	EosqAPIKey     string = "web_0000"
)
