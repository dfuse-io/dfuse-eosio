Token metadata service
----------------------

This service is there to inform anyone about the known token contracts on a given network.

It serves a list of top currency contracts for each platform.

Bootstrap /tmp/cache.json (based on abicodec's cache)

    go install -v ./cmd/abicodec/ && abicodec --cache-base-url=gs://dfuseio-global-abicache-us --cache-file-name=eos-kylin-v2.bin --export-cache --export-cache-base-url=gs://dfuseio-global-abicache-us --export-cache-file-name=eos-kylin-v2.json

Running
======
`tokenmeta serve --listen-grpc-addr=:9000`

TO DO
======

1) implement pipeline that calls the mutations batches
    a)forkable irreversible                             @s
    
2) implement bootstrap from cached abi json file
    a) loop through and setTokens
    
3) Implment flush cache to JSON store (GS)  @j
   a) dstore
   b) every 200 blocks 
   

4) Implement GRPC server @j

5) SHIP IT

6) DON"T TEST

Performence
*kylin @ block 89,692,219 bootrstrap took: 5m30s*
*jungle @ block 74,611,793 boostrap took: 6m30s* 

GRPC Calls
=====

*Get Tokens*
```shell script
grpcurl -plaintext localhost:9010 dfuse.tokenmeta.v1.EOS.GetTokens | jq
```

*Get Token Holders*
```shell script
*Get Token Holders*
echo '{
    "tokenContract":"eosio.token",
    "filterTokenSymbols":["EOS"],
    "limit": "1",
    "sortOrder":  "DESC",
    "sortField": "AMOUNT"
}' | grpcurl -plaintext -d @ localhost:9010 dfuse.tokenmeta.v1.EOS.GetTokenBalances | jq
```

*Get An Account*
```shell script
*Get Token Holders*
echo '{
    "account": "zbeoscharge1",
    "limit": "25",
    "sortOrder":  "DESC",
    "sortField": "AMOUNT"
}' | grpcurl -plaintext -d @ localhost:9010 dfuse.tokenmeta.v1.EOS.GetAccountBalances | jq
```