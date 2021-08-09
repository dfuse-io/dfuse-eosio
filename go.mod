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
	github.com/daaku/go.zipexe v1.0.1 // indirect
	github.com/davecgh/go-spew v1.1.1
	github.com/dfuse-io/blockmeta v0.0.2-0.20200818234314-2ad05605ed8d
	github.com/dfuse-io/bstream v0.0.2-0.20210218150906-c8e1b835d219
	github.com/dfuse-io/cli v0.0.2
	github.com/dfuse-io/client-go v0.0.0-20210526205821-9a3731282240
	github.com/dfuse-io/dauth v0.0.0-20200601190857-60bc6a4b4665
	github.com/dfuse-io/dbin v0.0.0-20200417174747-9a3806ff5643
	github.com/dfuse-io/derr v0.0.0-20201001203637-4dc9d8014152
	github.com/dfuse-io/dgraphql v0.0.2-0.20210506024839-da408de168b1
	github.com/dfuse-io/dgrpc v0.0.0-20210424033943-10e04dd5b19c
	github.com/dfuse-io/dhammer v0.0.0-20200723173708-b7e52c540f64
	github.com/dfuse-io/dipp v1.0.1-0.20200407033930-5c17c531c3c4
	github.com/dfuse-io/dlauncher v0.0.0-20201112212422-91f62bcef971
	github.com/dfuse-io/dmesh v0.0.0-20210526183400-53c814798369
	github.com/dfuse-io/dmetering v0.0.0-20210208201314-50d9add52005
	github.com/dfuse-io/dmetrics v0.0.0-20200508170817-3b8cb01fee68
	github.com/dfuse-io/dstore v0.1.1-0.20210507180120-88a95674809f
	github.com/dfuse-io/dtracing v0.0.0-20200417133307-c09302668d0c
	github.com/dfuse-io/eosio-boot v0.0.0-20201007140702-70b54b34c7a2
	github.com/dfuse-io/eosws-go v0.0.0-20210210152811-b72cc007d60a
	github.com/dfuse-io/firehose v0.1.1-0.20210424034522-6e5c05a23323
	github.com/dfuse-io/fluxdb v0.0.0-20210528171211-9df379b9a4e1
	github.com/dfuse-io/jsonpb v0.0.0-20200819202948-831ad3282037
	github.com/dfuse-io/kvdb v0.0.2-0.20210712145411-a765c572fce7
	github.com/dfuse-io/logging v0.0.0-20210518215502-2d920b2ad1f2
	github.com/dfuse-io/node-manager v0.0.2-0.20210413133800-b1893707a2f6
	github.com/dfuse-io/opaque v0.0.0-20210108174126-bc02ec905d48
	github.com/dfuse-io/pbgo v0.0.6-0.20210429181308-d54fc7723ad3
	github.com/dfuse-io/relayer v0.0.2-0.20210525190602-eeec872ddedf
	github.com/dfuse-io/search v0.0.2-0.20210506151819-66f981913d7f
	github.com/dfuse-io/search-client v0.0.0-20200602205137-71b300d129d2
	github.com/dfuse-io/shutter v1.4.1
	github.com/dfuse-io/validator v0.0.0-20200407012817-82c55c634c7a
	github.com/dustin/go-humanize v1.0.0
	github.com/eoscanada/eos-go v0.9.1-0.20210802215146-d4a45e07e9b5
	github.com/eoscanada/eosc v1.4.0
	github.com/eoscanada/pitreos v1.1.1-0.20200721154110-fb345999fa39
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
	github.com/streamingfast/merger v0.0.3-0.20210809165038-14f85d21b69b
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
