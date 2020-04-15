module github.com/dfuse-io/dfuse-eosio

go 1.13

require (
	cloud.google.com/go/bigtable v1.2.0
	contrib.go.opencensus.io/exporter/stackdriver v0.12.6
	github.com/GeertJohan/go.rice v1.0.0
	github.com/abourget/llerrgroup v0.2.0
	github.com/abourget/viperbind v0.1.0
	github.com/acarl005/stripansi v0.0.0-20180116102854-5a71ef0e047d
	github.com/andreyvit/diff v0.0.0-20170406064948-c7f18ee00883
	github.com/araddon/dateparse v0.0.0-20190622164848-0fb0a474d195
	github.com/arpitbbhayani/tripod v0.0.0-20170425181942-66807adce3a5
	github.com/auth0/go-jwt-middleware v0.0.0-20190805220309-36081240882b
	github.com/blevesearch/bleve v0.8.0
	github.com/coreos/bbolt v1.3.2
	github.com/davecgh/go-spew v1.1.1
	github.com/desertbit/timer v0.0.0-20180107155436-c41aec40b27f // indirect
	github.com/dfuse-io/blockmeta v0.0.0-20200415105003-0b2b107fb2ca
	github.com/dfuse-io/bstream v0.0.0-20200415145349-b867db804e41
	github.com/dfuse-io/dauth v0.0.0-20200407002455-d8256e743286
	github.com/dfuse-io/dbin v0.0.0-20200406215642-ec7f22e794eb
	github.com/dfuse-io/derr v0.0.0-20200406214256-c690655246a1
	github.com/dfuse-io/dgraphql v0.0.0-20200413182955-46b5777006c2
	github.com/dfuse-io/dgrpc v0.0.0-20200406214416-6271093e544c
	github.com/dfuse-io/dhammer v0.0.0-20200407040847-5fb1dcc749e1
	github.com/dfuse-io/dipp v1.0.1-0.20200407033930-5c17c531c3c4
	github.com/dfuse-io/dmesh v0.0.0-20200407045015-ea4e41ecdb6c
	github.com/dfuse-io/dmetering v0.0.0-20200407002734-b1d962316f28
	github.com/dfuse-io/dmetrics v0.0.0-20200406214800-499fc7b320ab
	github.com/dfuse-io/dstore v0.0.0-20200407173215-10b5ced43022
	github.com/dfuse-io/dtracing v0.0.0-20200406213603-4b0c0063b125
	github.com/dfuse-io/eosws-go v0.0.0-20191011181529-0eb3d4ce8743
	github.com/dfuse-io/jsonpb v0.0.0-20200406211248-c5cf83f0e0c0
	github.com/dfuse-io/kvdb v0.0.0-20200414142539-055d6b6fb265
	github.com/dfuse-io/logging v0.0.0-20200407175011-14021b7a79af
	github.com/dfuse-io/manageos v0.0.0-20200414173559-7946045b7139
	github.com/dfuse-io/merger v0.0.0-20200415110734-afa95bd5597e
	github.com/dfuse-io/opaque v0.0.0-20200407012705-75c4ca372d71
	github.com/dfuse-io/pbgo v0.0.6-0.20200415053022-5b009da68d9c
	github.com/dfuse-io/relayer v0.0.0-20200415111224-818cc0ff8f14
	github.com/dfuse-io/search v0.0.0-20200415150057-e6b4d3086942
	github.com/dfuse-io/search-client v0.0.0-20200415150146-6e9b323949c7
	github.com/dfuse-io/shutter v1.4.1-0.20200407040739-f908f9ab727f
	github.com/dfuse-io/validator v0.0.0-20200407012817-82c55c634c7a
	github.com/eoscanada/eos-go v0.9.1-0.20200415144303-2adb25bcdeca
	github.com/facebookgo/ensure v0.0.0-20200202191622-63f1cf65ac4c // indirect
	github.com/facebookgo/subset v0.0.0-20200203212716-c811ad88dec4 // indirect
	github.com/francoispqt/gojay v1.2.13
	github.com/gavv/httpexpect/v2 v2.0.3
	github.com/go-sql-driver/mysql v1.5.0
	github.com/gogo/protobuf v1.3.1
	github.com/golang-collections/collections v0.0.0-20130729185459-604e922904d3
	github.com/golang/protobuf v1.3.4
	github.com/gorilla/handlers v1.4.2
	github.com/gorilla/mux v1.7.3
	github.com/gorilla/websocket v1.4.1
	github.com/graph-gophers/graphql-go v0.0.0-20191115155744-f33e81362277
	github.com/hidal-go/hidalgo v0.0.0-20190814174001-42e03f3b5eaa
	github.com/improbable-eng/grpc-web v0.12.0
	github.com/koding/websocketproxy v0.0.0-20181220232114-7ed82d81a28c
	github.com/logrusorgru/aurora v0.0.0-20200102142835-e9ef32dff381
	github.com/manifoldco/promptui v0.7.0
	github.com/mattn/go-sqlite3 v2.0.3+incompatible
	github.com/minio/highwayhash v1.0.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_model v0.1.0
	github.com/prometheus/prom2json v1.3.0
	github.com/spf13/cobra v0.0.7
	github.com/spf13/pflag v1.0.3
	github.com/spf13/viper v1.6.2
	github.com/stretchr/testify v1.4.0
	github.com/tecbot/gorocksdb v0.0.0-20191217155057-f0fad39f321c // indirect
	github.com/teris-io/shortid v0.0.0-20171029131806-771a37caa5cf
	github.com/thedevsaddam/govalidator v1.9.9
	github.com/tidwall/gjson v1.5.0
	github.com/tidwall/sjson v1.0.4
	github.com/urfave/negroni v1.0.0 // indirect
	go.opencensus.io v0.22.3
	go.uber.org/atomic v1.6.0
	go.uber.org/automaxprocs v1.3.0
	go.uber.org/zap v1.14.0
	google.golang.org/api v0.15.0
	google.golang.org/grpc v1.26.0
	gopkg.in/olivere/elastic.v3 v3.0.75
	gopkg.in/yaml.v2 v2.2.8
	gotest.tools v2.2.0+incompatible
)

// to solve "github.com/ugorji/go/codec: ambiguous import: found package github.com/ugorji/go/codec in multiple modules:"
replace github.com/ugorji/go/codec => github.com/ugorji/go v1.1.2

replace github.com/graph-gophers/graphql-go => github.com/dfuse-io/graphql-go v0.0.0-20191010213351-ae758277182d

replace github.com/blevesearch/bleve => github.com/fproulx-eoscanada/bleve v0.0.0-20190823192325-db63d5f16d8b

replace github.com/census-instrumentation/opencensus-proto v0.1.0-0.20181214143942-ba49f56771b8 => github.com/census-instrumentation/opencensus-proto v0.0.3-0.20181214143942-ba49f56771b8

replace github.com/ShinyTrinkets/overseer => github.com/maoueh/overseer v0.2.1-0.20191024193921-39856397cf3f
