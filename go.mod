module github.com/dfuse-io/dfuse-eosio

go 1.13

require (
	cloud.google.com/go/bigtable v1.2.0
	contrib.go.opencensus.io/exporter/stackdriver v0.12.6
	github.com/GeertJohan/go.rice v1.0.0
	github.com/abourget/llerrgroup v0.2.0
	github.com/acarl005/stripansi v0.0.0-20180116102854-5a71ef0e047d
	github.com/andreyvit/diff v0.0.0-20170406064948-c7f18ee00883
	github.com/araddon/dateparse v0.0.0-20190622164848-0fb0a474d195
	github.com/arpitbbhayani/tripod v0.0.0-20170425181942-66807adce3a5
	github.com/auth0/go-jwt-middleware v0.0.0-20190805220309-36081240882b
	github.com/blendle/zapdriver v1.3.1
	github.com/blevesearch/bleve v1.0.9
	github.com/coreos/bbolt v1.3.2
	github.com/daaku/go.zipexe v1.0.1 // indirect
	github.com/davecgh/go-spew v1.1.1
	github.com/desertbit/timer v0.0.0-20180107155436-c41aec40b27f // indirect
	github.com/dfuse-io/blockmeta v0.0.0-20200602201207-f1981d03ef4c
	github.com/dfuse-io/bstream v0.0.0-20200602201235-217b145d1844
	github.com/dfuse-io/dauth v0.0.0-20200601190857-60bc6a4b4665
	github.com/dfuse-io/dbin v0.0.0-20200417174747-9a3806ff5643
	github.com/dfuse-io/derr v0.0.0-20200417132224-d333cfd0e9a0
	github.com/dfuse-io/dgraphql v0.0.0-20200602201547-61cf55466aa4
	github.com/dfuse-io/dgrpc v0.0.0-20200602203441-eded01d2c0e7
	github.com/dfuse-io/dhammer v0.0.0-20200609184524-26b4b6894242
	github.com/dfuse-io/dipp v1.0.1-0.20200407033930-5c17c531c3c4
	github.com/dfuse-io/dmesh v0.0.0-20200427143025-f55305fa4b95
	github.com/dfuse-io/dmetering v0.0.0-20200529171737-525c3029795c
	github.com/dfuse-io/dmetrics v0.0.0-20200508152325-93e7e9d576bb
	github.com/dfuse-io/dstore v0.1.1-0.20200612171130-4bdf691ac986
	github.com/dfuse-io/dtracing v0.0.0-20200417133307-c09302668d0c
	github.com/dfuse-io/eosio-boot v0.0.0-20200626011330-fdc29045fb0c
	github.com/dfuse-io/eosws-go v0.0.0-20200520155921-64414618efaf
	github.com/dfuse-io/jsonpb v0.0.0-20200602202333-ba4710ca7b4a
	github.com/dfuse-io/kvdb v0.0.0-20200615144400-d12a1d37270f
	github.com/dfuse-io/logging v0.0.0-20200611143916-aade15324493
	github.com/dfuse-io/manageos v0.0.0-20200611205430-5e3a009eadaa
	github.com/dfuse-io/merger v0.0.0-20200428163055-be2bcd7d6cc7
	github.com/dfuse-io/opaque v0.0.0-20200407012705-75c4ca372d71
	github.com/dfuse-io/pbgo v0.0.6-0.20200619193216-9bbf0c9fb1f8
	github.com/dfuse-io/relayer v0.0.0-20200602204614-8cdcc29ef339
	github.com/dfuse-io/search v0.0.0-20200615145420-7f4b7dc4c850
	github.com/dfuse-io/search-client v0.0.0-20200602205137-71b300d129d2
	github.com/dfuse-io/shutter v1.4.1-0.20200407040739-f908f9ab727f
	github.com/dfuse-io/validator v0.0.0-20200407012817-82c55c634c7a
	github.com/eoscanada/eos-go v0.9.1-0.20200617180637-a51457d45afd
	github.com/eoscanada/eosc v1.4.0
	github.com/francoispqt/gojay v1.2.13
	github.com/gavv/httpexpect/v2 v2.0.3
	github.com/golang-collections/collections v0.0.0-20130729185459-604e922904d3
	github.com/golang/protobuf v1.3.5
	github.com/google/cel-go v0.4.1
	github.com/gorilla/handlers v1.4.2
	github.com/gorilla/mux v1.7.3
	github.com/gorilla/websocket v1.4.1
	github.com/graph-gophers/graphql-go v0.0.0-20191115155744-f33e81362277
	github.com/hidal-go/hidalgo v0.0.0-20190814174001-42e03f3b5eaa
	github.com/improbable-eng/grpc-web v0.12.0
	github.com/lithammer/dedent v1.1.0
	github.com/logrusorgru/aurora v0.0.0-20200102142835-e9ef32dff381
	github.com/manifoldco/promptui v0.7.0
	github.com/minio/highwayhash v1.0.0
	github.com/onsi/ginkgo v1.11.0 // indirect
	github.com/onsi/gomega v1.8.1 // indirect
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_model v0.1.0
	github.com/prometheus/prom2json v1.3.0
	github.com/rs/cors v1.7.0 // indirect
	github.com/sergi/go-diff v1.0.1-0.20180205163309-da645544ed44 // indirect
	github.com/spf13/cobra v0.0.7
	github.com/spf13/pflag v1.0.3
	github.com/spf13/viper v1.6.2
	github.com/stretchr/testify v1.6.1
	github.com/teris-io/shortid v0.0.0-20171029131806-771a37caa5cf
	github.com/thedevsaddam/govalidator v1.9.9
	github.com/tidwall/gjson v1.5.0
	github.com/tidwall/sjson v1.0.4
	github.com/urfave/negroni v1.0.0 // indirect
	go.opencensus.io v0.22.3
	go.uber.org/atomic v1.6.0
	go.uber.org/zap v1.15.0
	golang.org/x/crypto v0.0.0-20200510223506-06a226fb4e37
	google.golang.org/api v0.15.0
	google.golang.org/grpc v1.26.0
	gopkg.in/olivere/elastic.v3 v3.0.75
	gopkg.in/yaml.v2 v2.2.8
	gotest.tools v2.2.0+incompatible
)

// to solve "github.com/ugorji/go/codec: ambiguous import: found package github.com/ugorji/go/codec in multiple modules:"
replace github.com/ugorji/go/codec => github.com/ugorji/go v1.1.2

replace github.com/graph-gophers/graphql-go => github.com/dfuse-io/graphql-go v0.0.0-20191010213351-ae758277182d

replace github.com/census-instrumentation/opencensus-proto v0.1.0-0.20181214143942-ba49f56771b8 => github.com/census-instrumentation/opencensus-proto v0.0.3-0.20181214143942-ba49f56771b8

replace github.com/ShinyTrinkets/overseer => github.com/maoueh/overseer v0.2.1-0.20191024193921-39856397cf3f
