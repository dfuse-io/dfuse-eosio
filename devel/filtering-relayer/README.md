# Filtering Relayer Demo

## Overview

This demos the capability of a filtering relayer. The dfuse-eosio topology is 
as follows:

### `global.yaml`
This contains all the apps that runs in a "global" namespace. Its general purpose is to store block
information for the totality of the network

1 - We have a `relayer`, that is connected to a `mindreader`. This `relayer` receives all 
the blocks from said `mindreader`. 

2 - We have a `trxdb-loader` that simply writes the blocks into our local big table store

### `filtering.yaml`
Ths is represents a filtered namespace, where it only consumes filtered blocks from the "global" namespace. 
The blocks are striped of information that do not meet specific filtering criterias.  
 
1 - We have defined the filters in the common flags

2 - We only have a `relayer` that receives the blocks from the "global" relayer and strips out the un-necessary 
information

3 - We have a `trxdb-loader` that simply write the transactions received within the stripped blocks

4 - We have setup a `switch db` implementation where the `common-trxdb-dsn` contains 2 database connection strings. One that reads
the blocks from the "global" namespace and a seconds that reads & writes the transactions to the filtered namespace

5 - We have enabled transaction truncation as well 


## Running It

To test trxdb switch db implementation, follow those steps:

1. In a terminal, run `gcloud beta emulators bigtable start`
2. In the terminal where you are going to run the `start.sh` command, do:
    `$(gcloud beta emulators bigtable env-init)`
3. Uncomment in `global.yaml` the two commented option (`trxdb-loader` app and `common-trxdb-dsn`)
4. Uncomment in `filtering.yaml` file the commented `common-trxdb-dsn` option
5. Launch `start.sh -c`

## Usefull tools

To read a block from the "global" block database you can use this:
    `dfuseeos tools db blk 200 --dsn bigkv://dev.dev/test-trxdb-blocks`
reading the same block from the "filtered" transaction database will 
return and error since it only contains transaction information, You can try it:
    `dfuseeos tools db blk 200 --dsn bigkv://dev.dev/test-trxdb-trxs`