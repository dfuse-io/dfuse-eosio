module github.com/dfuse-io/dfuse-eosio

go 1.14

require (
	contrib.go.opencensus.io/exporter/stackdriver v0.13.8
	github.com/GeertJohan/go.rice v1.0.0
	github.com/ShinyTrinkets/overseer v0.3.0
	github.com/acarl005/stripansi v0.0.0-20180116102854-5a71ef0e047d
	github.com/andreyvit/diff v0.0.0-20170406064948-c7f18ee00883
	github.com/araddon/dateparse v0.0.0-20190622164848-0fb0a474d195
	github.com/arpitbbhayani/tripod v0.0.0-20170425181942-66807adce3a5
	github.com/auth0/go-jwt-middleware v0.0.0-20190805220309-36081240882b
	github.com/blevesearch/bleve v1.0.14
	github.com/bmizerany/assert v0.0.0-20160611221934-b7ed37b82869 // indirect
	github.com/coreos/etcd v3.3.25+incompatible // indirect
	github.com/coreos/go-systemd v0.0.0-20191104093116-d3cd4ed1dbcf // indirect
	github.com/coreos/license-bill-of-materials v0.0.0-20190913234955-13baff47494e // indirect
	github.com/daaku/go.zipexe v1.0.1 // indirect
	github.com/davecgh/go-spew v1.1.1
	github.com/dfuse-io/eosio-boot v0.0.0-20201007140702-70b54b34c7a2
	github.com/dfuse-io/eosws-go v0.0.0-20210210152811-b72cc007d60a
	github.com/dustin/go-humanize v1.0.0
	github.com/eoscanada/eos-go v0.9.1-0.20210812015252-984fc96878b6
	github.com/eoscanada/eosc v1.4.0
	github.com/eoscanada/pitreos v1.1.1-0.20210811185752-fa06394508d0
	github.com/francoispqt/gojay v1.2.13
	github.com/gavv/httpexpect/v2 v2.0.3
	github.com/golang-collections/collections v0.0.0-20130729185459-604e922904d3
	github.com/golang/protobuf v1.5.2
	github.com/google/cel-go v0.4.1
	github.com/gorilla/handlers v1.4.2
	github.com/gorilla/mux v1.7.3
	github.com/gorilla/websocket v1.4.2
	github.com/graph-gophers/graphql-go v0.0.0-20191115155744-f33e81362277
	github.com/klauspost/compress v1.10.2
	github.com/lithammer/dedent v1.1.0
	github.com/logrusorgru/aurora v2.0.3+incompatible
	github.com/lytics/lifecycle v0.0.0-20130117214539-7b4c4028d422 // indirect
	github.com/lytics/ordpool v0.0.0-20130426221837-8d833f097fe7
	github.com/manifoldco/promptui v0.8.0
	github.com/mitchellh/go-testing-interface v1.14.1
	github.com/onsi/ginkgo v1.11.0 // indirect
	github.com/onsi/gomega v1.8.1 // indirect
	github.com/paulbellamy/ratecounter v0.2.0
	github.com/pkg/errors v0.9.1
	github.com/sergi/go-diff v1.0.1-0.20180205163309-da645544ed44 // indirect
	github.com/sirupsen/logrus v1.8.1 // indirect
	github.com/spf13/cobra v1.1.3
	github.com/spf13/viper v1.7.0
	github.com/streamingfast/blockmeta v0.0.2-0.20210811194956-90dc4202afda
	github.com/streamingfast/bstream v0.0.2-0.20210811181043-4c1920a7e3e3 // indirect
	github.com/streamingfast/cli v0.0.3-0.20210811201236-5c00ec55462d // indirect
	github.com/streamingfast/client-go v0.0.0-20210811201850-a359c7648d44 // indirect
	github.com/streamingfast/dauth v0.0.0-20210811181149-e8fd545948cc
	github.com/streamingfast/dbin v0.0.0-20210809205249-73d5eca35dc5
	github.com/streamingfast/derr v0.0.0-20210811180100-9138d738bcec
	github.com/streamingfast/dgraphql v0.0.2-0.20210811200910-e1966c29c473
	github.com/streamingfast/dgrpc v0.0.0-20210811180351-8646818518b2 // indirect
	github.com/streamingfast/dhammer v0.0.0-20210811180702-456c4cf0a840 // indirect
	github.com/streamingfast/dipp v1.0.1-0.20210811200841-d2cca4e058e6 // indirect
	github.com/streamingfast/dlauncher v0.0.0-20210811194929-f06e488e63da
	github.com/streamingfast/dmesh v0.0.0-20210811181323-5a37ad73216b
	github.com/streamingfast/dmetering v0.0.0-20210811181351-eef120cfb817
	github.com/streamingfast/dmetrics v0.0.0-20210811180524-8494aeb34447 // indirect
	github.com/streamingfast/dstore v0.1.1-0.20210811180812-4db13e99cc22 // indirect
	github.com/streamingfast/dtracing v0.0.0-20210811175635-d55665d3622a
	github.com/streamingfast/firehose v0.1.1-0.20210811195158-d4b116b4b447
	github.com/streamingfast/fluxdb v0.0.0-20210811195408-0515ef659298
	github.com/streamingfast/jsonpb v0.0.0-20210811021341-3670f0aa02d0 // indirect
	github.com/streamingfast/kvdb v0.0.2-0.20210811194032-09bf862bd2e3
	github.com/streamingfast/logging v0.0.0-20210811175431-f3b44b61606a // indirect
	github.com/streamingfast/merger v0.0.3-0.20210811195536-1011c89f0a67
	github.com/streamingfast/node-manager v0.0.2-0.20210811195732-ccdf9f70dd0b
	github.com/streamingfast/opaque v0.0.0-20210811180740-0c01d37ea308
	github.com/streamingfast/pbgo v0.0.6-0.20210811160400-7c146c2db8cc // indirect
	github.com/streamingfast/relayer v0.0.2-0.20210811200014-6e0e8bc2814f
	github.com/streamingfast/search v0.0.2-0.20210811200310-ec8d3b03e104
	github.com/streamingfast/search-client v0.0.0-20210811200417-677bdb765983
	github.com/streamingfast/shutter v1.5.0 // indirect
	github.com/streamingfast/validator v0.0.0-20210812013448-b9da5752ce14 // indirect
	github.com/stretchr/testify v1.7.0
	github.com/teris-io/shortid v0.0.0-20201117134242-e59966efd125
	github.com/thedevsaddam/govalidator v1.9.9
	github.com/tidwall/gjson v1.6.7
	github.com/tidwall/sjson v1.0.4
	github.com/urfave/negroni v1.0.0 // indirect
	go.etcd.io/etcd v0.5.0-alpha.5.0.20200425165423-262c93980547 // indirect
	go.etcd.io/etcd/pkg/v3 v3.5.0-alpha.0 // indirect
	go.opencensus.io v0.23.0
	go.uber.org/atomic v1.7.0
	go.uber.org/multierr v1.6.0 // indirect
	go.uber.org/zap v1.17.0
	golang.org/x/crypto v0.0.0-20210322153248-0c34fe9e7dc2
	golang.org/x/oauth2 v0.0.0-20210805134026-6f1e6394065a
	golang.org/x/term v0.0.0-20201210144234-2321bbc49cbf // indirect
	golang.org/x/time v0.0.0-20191024005414-555d28b269f0
	google.golang.org/grpc v1.39.1
	google.golang.org/grpc/examples v0.0.0-20210526223527-2de42fcbbce3 // indirect
	gopkg.in/olivere/elastic.v3 v3.0.75
	gopkg.in/yaml.v2 v2.4.0
	gotest.tools v2.2.0+incompatible
	honnef.co/go/tools v0.0.1-2020.1.4 // indirect
)

// to solve "github.com/ugorji/go/codec: ambiguous import: found package github.com/ugorji/go/codec in multiple modules:"
replace github.com/ugorji/go/codec => github.com/ugorji/go v1.1.2

replace github.com/graph-gophers/graphql-go => github.com/streamingfast/graphql-go v0.0.0-20210204202750-0e485a040a3c

replace github.com/census-instrumentation/opencensus-proto v0.1.0-0.20181214143942-ba49f56771b8 => github.com/census-instrumentation/opencensus-proto v0.0.3-0.20181214143942-ba49f56771b8

replace github.com/ShinyTrinkets/overseer => github.com/dfuse-io/overseer v0.2.1-0.20210326144022-ee491780e3ef

// The go-testing-interface version matches the Golang version to compile against, in this case, we want
// compatibility with 1.14 which is our minimum version. So we enforce a strict version to v1.14.1 now.
replace github.com/mitchellh/go-testing-interface => github.com/mitchellh/go-testing-interface v1.14.1
