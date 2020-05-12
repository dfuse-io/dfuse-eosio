This package holds the storage abstraction needed for FluxDB to function.

Historically, there was:
* One native Bigtable implementation
* Then we had a few implementations
* Then we settled on using `kvdb` as an abstraction, to use whatever backing KV store.
