## KVDB Loader

This repository holds the code necessary to take the blockchain data
and insert it into out database instance (BigTable).

### Indexing Development

Here the instructions to easily try out your code modifications. First,
ensure you have the `gcloud` BigTable Emulator installed through the
`gcloud` CLI:

    gcloud components list | grep "Cloud Bigtable Emulator"

    # If not already installed
    gcloud components install bigtable

Once installed, you will need to have three terminals open. The first one
will run the actual BigTable Emulator.

The second one will open a port forward so a communication channel can exists
with the block relayer to pull blocks from.

The third one will run the actual indexing process (your code) that
connects to the BigTable Emulator instead of a real database instance.

In the first terminal, starts the BigTable Emulator:

    gcloud beta emulators bigtable start

In the second terminal, open a port forward with the block relayer service
you are interested with (use `eos-dev1` namespace to start):

    kc -n eos-dev1 port-forward svc/relayer 10101

In the third terminal, use the `index_dev.sh` script to automatically
starts the indexing in development mode, filling your own local BigTable
Emulator with data so you can inspect it.

    ./index_dev.sh

By default, the `index_dev.sh` scripts is configured to pull blocks
from the `eos-dev1` development test network.

You can easily override those parameters by specifying the `extra_args`
environment variable, being at the bottom, it will override the default
parameters.

For example, to index EOS Mainnet blocks, first kill your initial port
forward and point it to EOS Mainnet block relayer:

    kc -n eos-mainnet port-forward relayer 10101

Then in the third terminal, start the `index_dev.sh` script like this:

    extra_args="-chain-id=aca376f206b8fc25a6ed44dbdc66547c36c6c33e3a119ffbeaef943642f0e906 \
    -source-store=gs://example/blocks" ./index_dev.sh

As long as the BigTable Emulator loader instance is running, restarting the process
will start back where it left previously, just like in production.

To start from scratch, simply kill the BigTable Emulator and start it back.

#### Specific Block Range

If you are hunting a specific thing that happen within a particular block
or block range, the best is to start the indexer for this exact range. To
do that, we will simply use `extra_args` environment parameters to override
the processing mode to `batch` and pass the block range we are interested in.

    extra_args="-processing-type=batch -start-block-num=10 -stop-block-num=20" ./index_dev.sh

This will start the loading process, in batch mode for block range `[10, 20]`.

#### Inspect Data

The best way to check out if to inspect the data being inserted into BigTable Emulator.
Open a fourth terminal (yeah sorry :$).

_Note_: You can also kill the `./index_dev.sh` script and use the terminal there if you
don't need the indexing to continue while testing your stuff.

In this terminal, initialize the environment so `cbt` connects to BigTable Emulator:

    $(gcloud beta emulators bigtable env-init)

Once this is done, you can now use `cbt` to check some tables:

    cbt -project=test -instance=dev read eos-test-v1-accounts prefix=a: count=1
    cbt -project=test -instance=dev read eos-test-v1-blocks prefix=fffffffc count=1
    cbt -project=test -instance=dev read eos-test-v1-trxs prefix=3 count=1
    cbt -project=test -instance=dev read eos-test-v1-timeline prefix=bf: count=1

More queries (perform them on low-level instances):

    # Count number of blocks
    cbt -project=test -instance=dev read eos-test-v1-blocks columns=meta:header | grep "meta:header" | wc -l

    # Count number of transaction keys
    cbt -project=test -instance=dev read eos-test-v1-trxs | grep '\----------------------------------------' | wc -l

    # Count number of irreversible transactions
    cbt -project=test -instance=dev read eos-test-v1-trxs columns=meta:irreversible | grep "meta:irreversible" | wc -l

##### Troubleshooting

If you see the message:

```
2019/06/04 11:02:37 -creds flag unset, will use gcloud credential
2019/06/04 11:02:38 Getting list of tables: rpc error: code = InvalidArgument desc = When parsing 'projects/dev/instances/dev' : Invalid id for collection instances : Length should be between [6,33], but found 3 'dev'
```

When using `cbt`, it means you are trying to connect to a real Google BigTable instance and the
terminal environment is not initialized correctly with `$(gcloud beta emulators bigtable env-init)`.

Run `$(gcloud beta emulators bigtable env-init)` and retry the command.
