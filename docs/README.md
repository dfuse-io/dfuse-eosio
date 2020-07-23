# Docs about `dfuse for EOSIO`

This is a (temporary?) place to store documentation about the
open-source `dfuse for EOSIO` software.

If you are looking for documentation on API usage, please see:
https://docs.dfuse.io


## Administration Guide

* [3 phase launch of large networks instructions](https://github.com/dfuse-io/dfuse-eosio/issues/26)
* Some sample [partial sync instructions](../PARTIAL_SYNC.md) for Kylin

### Merger

#### From Reprocessing to Live

After a re-processing phase where `mindreader` created the merged bundles on its own, the `merger` needs to
get back correctly on this feet.

When starting, the `merger` first determines what was the last bundle merged. From that value, it will then
start to drop one-block files until it reached the first file it should merge in a bundle.

So to go from re-processing to a live process, one need to perform those steps:

- Starts back `mindreader` in live mode earlier from the last merged bundle
- Starts back `merger`.

Here example cases and explanation, assuming the last merged bundle is `900` (covers block `0-999`):

- `mindreader` restarts at 0890 | Produces block 0890, 0891, 0892 ... 1010 | OK, `merger` checks last merged bundle and drops `890-999`
- `mindreader` restarts at 0910 | Produces block 0910, 0911, 0912 ... 1010 | OK, `merger` checks last merged bundle and drops `910-999`
- `mindreader` restarts at 1000 | Produces block 1000, 1001, 1002 ... 1010 | OK, `merger` checks last merged bundle and drops nothing
- `mindreader` restarts at 1010 | Produces block 1010, 1011, 1012 ...      | KO, `merger` will complain about missing one blocks

#### Recovering from missing one-block files

If the merger complains about missing one-block files and there are indeed missing, to recover from
this situation, you will need to start back mindreader **before** the missing one-block file.

For example, let's say you are missing block 925, then you should start back `mindreader` at block
`924`.

## Architecture Diagrams

### General components overview

![General components overview](general_architecture.png)

---

### Search components diagram

![Search diagrams](search.png)

---

## In depth, in video format

* [General Overview — dfuse for EOSIO Architecture Series](https://www.youtube.com/watch?v=q3Mi1S4nvcU)
* [manageos & mindreader — dfuse for EOSIO Architecture Series](https://www.youtube.com/watch?v=uR1cB5QpvcY)
* [deepmind & the dfuse Data Model — dfuse for EOSIO Architecture Series](https://www.youtube.com/watch?v=BMcSmqvNU1Q)
* [bstream part 1 — dfuse for EOSIO Architecture Series](https://www.youtube.com/watch?v=LX7_Q7b5pyc)
* [bstream part 2 — dfuse for EOSIO Architecture Series](https://www.youtube.com/watch?v=3HK95ng51ZM)
* [pitreos — dfuse for EOSIO Architecture Series](https://www.youtube.com/watch?v=9oPa8OqZdWE)
* [High Availability with Relayers, Merger — dfuse for EOSIO Architecture Series](https://www.youtube.com/watch?v=yG-lxgp7g10)
* [Install and Run the dfuse for EOSIO Stack w/ Alex Bourget, CTO @ dfuse — Free Webinar](https://www.youtube.com/watch?v=1AH2wMESu2Y)
* [How to Use dfuse for EOSIO as a Blockchain Developer w/ Alex Bourget, CTO @ dfuse — Free Webinar](https://www.youtube.com/watch?v=bFi6H5iO8ww)


