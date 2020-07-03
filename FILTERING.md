# dfuse for EOSIO - Filtering

## Filters

The `--common-include-filter-expr` and `--common-exclude-filter-expr` parameters are both CEL programs.

**Important**: They filter what gets **processed** by different components across dfuse for EOSIO stack. They are not a language you can use to query the indexes.

CEL is Google's Common Expression Language:

* [Introduction](https://github.com/google/cel-spec/blob/master/doc/intro.md)
* [Language reference](https://github.com/google/cel-spec/blob/master/doc/langdef.md)
* [Google Open Source CEL Website](https://opensource.google/projects/cel)
* [Go implementation used here](https://github.com/google/cel-go)

## Components

Currently, two components uses the filtering options when they are defined:
- search
- trxdb-loader

The `search` will only index actions that matched the inclusion filter and did **not** match the exclusion one. The
`trxdb-loader` component will only save transaction traces in the database that contains at least 1 matching action.

## Identifiers

An similar identifiers available for searching in **dfuse Search** is available for filtering but
with more capabilities because any action's data field can be search for whereas in **dfuse Search**,
only a subset of all fields is possible.

For example, a **dfuse Search** of:

```
receiver:eosio.token data.from:bob
```

would be filtered as:

```
receiver == 'eosio.token' && data.from == 'bob'
```

See https://docs.dfuse.io/reference/eosio/search-terms/ for all EOSIO terms that can be filtered.

### Examples

Showcase examples here are given as examples, mainly for syntax purposes, so you can see the full
power of the CEL filtering language

There might be new stuff to add to certain examples, like spam coins or new system contracts, this
is **not** a fully accurate document for those stuff, you are invited to make your own research
to ensure completeness based on your use case.

#### EOS Mainnet Spam

Here an example to filter out spam transactions on the EOS Mainnet:

```
account == 'eidosonecoin' || receiver == 'eidosonecoin' || (account == 'eosio.token' && (data.to == 'eidosonecoin' || data.from == 'eidosonecoin'))
```
