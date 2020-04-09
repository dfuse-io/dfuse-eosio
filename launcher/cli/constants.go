package cli

import (
	pbbstream "github.com/dfuse-io/pbgo/dfuse/bstream/v1"
)

const (
	Protocol                    pbbstream.Protocol = pbbstream.Protocol_EOS
	BlockmetaDSN                string             = "badger://%s/kvdb_badger.db?compression=zstd"
	KVBDDSN                     string             = "badger://%s/kvdb_badger.db?compression=zstd" //%s will be replace by data-dir
	MergedBlocksFilesPath       string             = "storage/merged-blocks"
	OneBlockFilesPath           string             = "storage/one-blocks"
	DmeshServiceVersion         string             = "v1"
	DmeshNamespace              string             = "local"
	NetworkID                   string             = "eos-local"
	EosManagerHTTPAddr          string             = ":13008"
	EosMindreaderHTTPAddr       string             = ":13009"
	MindreaderGRPCAddr          string             = ":13010"
	RelayerServingAddr          string             = ":13011"
	MergerServingAddr           string             = ":13012"
	AbiServingAddr              string             = ":13013"
	BlockmetaServingAddr        string             = ":13014"
	ArchiveServingAddr          string             = ":13015"
	ArchiveHTTPServingAddr      string             = ":13016"
	LiveServingAddr             string             = ":13017"
	RouterServingAddr           string             = ":13018"
	RouterHTTPServingAddr       string             = ":13019"
	KvdbHTTPServingAddr         string             = ":13020"
	IndexerServingAddr          string             = ":13021"
	IndexerHTTPServingAddr      string             = ":13022"
	TokenmetaServingAddr        string             = ":13023" // Not implemented yet, present for booting purposes, does not work
	DgraphqlHTTPServingAddr     string             = ":13024"
	DgraphqlGrpcServingAddr     string             = ":13025"
	DashboardGrpcServingAddr    string             = ":13726"
	EoswsHTTPServingAddr        string             = ":13027"
	ForkresolverServingAddr     string             = ":13028"
	ForkresolverHTTPServingAddr string             = ":13029"
	FluxDBServingAddr           string             = ":13030"
	DashboardHTTPListenAddr     string             = ":8080"
	EosqHTTPServingAddr         string             = ":8081"
	MindreaderNodeosAPIAddr     string             = ":9888"
	NodeosAPIAddr               string             = ":8888"
)
