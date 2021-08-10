module github.com/dfuse-io/dfuse-eosio

go 1.14

require (
	contrib.go.opencensus.io/exporter/stackdriver v0.13.4
	github.com/GeertJohan/go.rice v1.0.0
	github.com/ShinyTrinkets/overseer v0.3.0
	github.com/acarl005/stripansi v0.0.0-20180116102854-5a71ef0e047d
	github.com/andreyvit/diff v0.0.0-20170406064948-c7f18ee00883
	github.com/araddon/dateparse v0.0.0-20190622164848-0fb0a474d195
	github.com/arpitbbhayani/tripod v0.0.0-20170425181942-66807adce3a5
	github.com/auth0/go-jwt-middleware v0.0.0-20190805220309-36081240882b
	github.com/blevesearch/bleve v1.0.9
	github.com/bmizerany/assert v0.0.0-20160611221934-b7ed37b82869 // indirect
	github.com/coreos/etcd v3.3.25+incompatible // indirect
	github.com/coreos/go-systemd v0.0.0-20191104093116-d3cd4ed1dbcf // indirect
	github.com/coreos/license-bill-of-materials v0.0.0-20190913234955-13baff47494e // indirect
	github.com/daaku/go.zipexe v1.0.1 // indirect
	github.com/davecgh/go-spew v1.1.1
	github.com/dfuse-io/bstream v0.0.2-0.20210810200727-06a4c4aaeb99
	github.com/dfuse-io/cli v0.0.2
	github.com/dfuse-io/client-go v0.0.0-20210526205821-9a3731282240
	github.com/dfuse-io/dauth v0.0.0-20200601190857-60bc6a4b4665 // indirect
	github.com/dfuse-io/derr v0.0.0-20201001203637-4dc9d8014152 // indirect
	github.com/dfuse-io/dhammer v0.0.0-20210810131337-dad508927fdd // indirect
	github.com/dfuse-io/dipp v1.0.1-0.20200407033930-5c17c531c3c4
	github.com/dfuse-io/dmesh v0.0.0-20210224224128-9a9ef510dce1 // indirect
	github.com/dfuse-io/dmetering v0.0.0-20210112023524-c3ddadbc0d6a // indirect
	github.com/dfuse-io/dmetrics v0.0.0-20200508170817-3b8cb01fee68 // indirect
	github.com/dfuse-io/dstore v0.1.1-0.20210507180120-88a95674809f // indirect
	github.com/dfuse-io/dtracing v0.0.0-20200417133307-c09302668d0c // indirect
	github.com/dfuse-io/eosio-boot v0.0.0-20201007140702-70b54b34c7a2
	github.com/dfuse-io/eosws-go v0.0.0-20210210152811-b72cc007d60a
	github.com/dfuse-io/jsonpb v0.0.0-20200819202948-831ad3282037
	github.com/dfuse-io/kvdb v0.0.2-0.20201013164626-89b668e6bd69 // indirect
	github.com/dfuse-io/logging v0.0.0-20210518215502-2d920b2ad1f2
	github.com/dfuse-io/pbgo v0.0.6-0.20210810190312-2053a3b38779
	github.com/dfuse-io/validator v0.0.0-20200407012817-82c55c634c7a
	github.com/dustin/go-humanize v1.0.0
	github.com/eoscanada/eos-go v0.9.1-0.20210802215146-d4a45e07e9b5
	github.com/eoscanada/eosc v1.4.0
	github.com/eoscanada/pitreos v1.1.1-0.20210810174831-3e4668dcda56
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
	github.com/spf13/cobra v1.1.3
	github.com/spf13/viper v1.7.0
	github.com/streamingfast/blockmeta v0.0.2-0.20210810200809-9cf9dff77535
	github.com/streamingfast/dauth v0.0.0-20210809192433-4c758fd333ac
	github.com/streamingfast/dbin v0.0.0-20210809205249-73d5eca35dc5
	github.com/streamingfast/derr v0.0.0-20210810022442-32249850a4fb
	github.com/streamingfast/dgraphql v0.0.2-0.20210810194021-e343b2730c32
	github.com/streamingfast/dgrpc v0.0.0-20210810185305-905172f728e8 // indirect
	github.com/streamingfast/dhammer v0.0.0-20210810183918-d5d2313b80a7 // indirect
	github.com/streamingfast/dlauncher v0.0.0-20210810201623-b4bad04a0782
	github.com/streamingfast/dmesh v0.0.0-20210810194153-7e56f639d42f
	github.com/streamingfast/dmetering v0.0.0-20210809193048-81d008c90843
	github.com/streamingfast/dmetrics v0.0.0-20210810172555-564d623b116f // indirect
	github.com/streamingfast/dstore v0.1.1-0.20210810110932-928f221474e4 // indirect
	github.com/streamingfast/dtracing v0.0.0-20210810040633-7c6259bea4a7
	github.com/streamingfast/firehose v0.1.1-0.20210810201729-f4f65f7bc597
	github.com/streamingfast/fluxdb v0.0.0-20210810195359-748b15076010
	github.com/streamingfast/kvdb v0.0.2-0.20210809203849-c1762028eb64
	github.com/streamingfast/merger v0.0.3-0.20210810201721-8308c7731ce1
	github.com/streamingfast/node-manager v0.0.2-0.20210810201803-d07d31a2f254
	github.com/streamingfast/opaque v0.0.0-20210809210154-b964592beb5d
	github.com/streamingfast/relayer v0.0.2-0.20210810201213-52e46787d413
	github.com/streamingfast/search v0.0.2-0.20210810201932-35eabdd80bbc
	github.com/streamingfast/search-client v0.0.0-20210810184115-1714dd3512a9
	github.com/streamingfast/shutter v1.5.0 // indirect
	github.com/stretchr/testify v1.7.0
	github.com/teris-io/shortid v0.0.0-20201117134242-e59966efd125
	github.com/thedevsaddam/govalidator v1.9.9
	github.com/tidwall/gjson v1.6.7
	github.com/tidwall/sjson v1.0.4
	github.com/urfave/negroni v1.0.0 // indirect
	go.etcd.io/etcd v0.5.0-alpha.5.0.20200425165423-262c93980547 // indirect
	go.etcd.io/etcd/pkg/v3 v3.5.0-alpha.0 // indirect
	go.opencensus.io v0.22.5
	go.uber.org/atomic v1.7.0
	go.uber.org/multierr v1.6.0 // indirect
	go.uber.org/zap v1.16.1-0.20210329175301-c23abee72d19
	golang.org/x/crypto v0.0.0-20210322153248-0c34fe9e7dc2
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	golang.org/x/term v0.0.0-20201210144234-2321bbc49cbf // indirect
	golang.org/x/time v0.0.0-20191024005414-555d28b269f0
	google.golang.org/grpc v1.37.0
	google.golang.org/grpc/examples v0.0.0-20210526223527-2de42fcbbce3 // indirect
	gopkg.in/olivere/elastic.v3 v3.0.75
	gopkg.in/yaml.v2 v2.4.0
	gotest.tools v2.2.0+incompatible
	honnef.co/go/tools v0.0.1-2020.1.4 // indirect
)

// to solve "github.com/ugorji/go/codec: ambiguous import: found package github.com/ugorji/go/codec in multiple modules:"
replace github.com/ugorji/go/codec => github.com/ugorji/go v1.1.2

replace github.com/graph-gophers/graphql-go => github.com/dfuse-io/graphql-go v0.0.0-20210204202750-0e485a040a3c

replace github.com/census-instrumentation/opencensus-proto v0.1.0-0.20181214143942-ba49f56771b8 => github.com/census-instrumentation/opencensus-proto v0.0.3-0.20181214143942-ba49f56771b8

replace github.com/ShinyTrinkets/overseer => github.com/maoueh/overseer v0.2.1-0.20191024193921-39856397cf3f

// The go-testing-interface version matches the Golang version to compile against, in this case, we want
// compatibility with 1.14 which is our minimum version. So we enforce a strict version to v1.14.1 now.
replace github.com/mitchellh/go-testing-interface => github.com/mitchellh/go-testing-interface v1.14.1
