## gRPCurl

_HeadInfo on mindreader_
`grpcurl -v -plaintext -d '{}' localhost:13010 dfuse.headinfo.v1.HeadInfo.GetHeadInfo`

_Relayer stream_
`grpcurl -plaintext -d '{}' localhost:13011 dfuse.bstream.v1.BlockStream.Blocks | jq .number`

_Blockmeta_

```
grpcurl -plaintext    -d '{}' localhost:13015 dfuse.blockmeta.v1.BlockID.LIBID
grpcurl -plaintext    -d '{"blockNum":"545"}' localhost:13015 dfuse.blockmeta.v1.BlockID.NumToID
```


## gRPC-web

in dashboard folder, run:

```
protoc --plugin="protoc-gen-ts=./client/node_modules/.bin/protoc-gen-ts" --js_out="import_style=commonjs,binary:./client/src/" --ts_out="service=grpc-web:./client/src/" ./pb/dashboard.proto
```

if getting `'proto' is not defined` when building js client, add `/* eslint-disable */` to first line of each generated `.js` and `.ts` file
