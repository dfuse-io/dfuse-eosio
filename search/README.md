# EOSIO-specifics for dfuse Search

## Filters

The `--search-common-action-filter-on-expr` and `--search-common-action-filter-out-expr` parameters are both CEL programs.

NOTE: They filter what gets **indexed** into search. They are not a language you can use to query the indexes.

CEL is Google's Common Expression Language:

* [Introduction](https://github.com/google/cel-spec/blob/master/doc/intro.md)
* [Language reference](https://github.com/google/cel-spec/blob/master/doc/langdef.md)
* [Google Open Source CEL Website](https://opensource.google/projects/cel)
* [Go implementation used here](https://github.com/google/cel-go)


## Identifiers

The same list of identifiers available for searching in **dfuse Search** is available for filtering.

For example, a **dfuse Search** of:

```
receiver:eosio.token data.from:bob
```

would be filtered as:

```
receiver == 'eosio.token' && data.from == 'bob'
```

See https://docs.dfuse.io/reference/eosio/search-terms/ for all EOSIO terms that can be filtered.
