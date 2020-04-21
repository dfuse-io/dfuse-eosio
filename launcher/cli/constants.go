package cli

import (
	pbbstream "github.com/dfuse-io/pbgo/dfuse/bstream/v1"
)

const (
	Protocol              pbbstream.Protocol = pbbstream.Protocol_EOS
	BlockmetaDSN          string             = "badger://%s/kvdb/kvdb_badger.db?compression=zstd" //%s will be replaced by data-dir
	KVDBDSN               string             = "badger://%s/kvdb/kvdb_badger.db?compression=zstd" //%s will be replaced by `<data-dir>`
	FluxDSN               string             = "badger://%s/fluxdb/flux.db"                       //%s will be replaced by `<data-dir>/<flux-data-dir>
	MergedBlocksFilesPath string             = "storage/merged-blocks"
	IndicesFilePath       string             = "storage/indexes"
	OneBlockFilesPath     string             = "storage/one-blocks"
	PitreosPath           string             = "storage/pitreos"
	SnapshotsPath         string             = "storage/snapshots"
	DmeshServiceVersion   string             = "v1"
	DmeshNamespace        string             = "local"
	NetworkID             string             = "eos-local"
	NodeosBinPath         string             = "nodeos"

	// Ports
	EosManagerAPIAddr           string = ":13008"
	EosMindreaderHTTPAddr       string = ":13009"
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
	DashboardHTTPListenAddr     string = ":8081"
	APIProxyHTTPListenAddr      string = ":8080"
	MindreaderNodeosAPIAddr     string = ":9888"
	NodeosAPIAddr               string = ":8888"

	DgraphqlAPIKey string = "web_0000"
	JWTIssuerURL   string = "null://dfuse"
	EosqAPIKey     string = "web_0000"
)
