ABI Encoder / Decoder Service
=============================


Inputs:
 * Relayed blocks filtering for changing ABIs.
   or
 * A search query "receiver:eosio account:eosio action:setabi" with keeping the cursor in the local cache for fast access.
   This allows us to sync back super fast when we restart.. no need for Joining source, and all.



There are a few approaches to doing this:



Built into `grapheos`
---------------------

* Grapheos would need to be connected with a joining source to learn about latest.
  Or issue a streaming search from locally, basically copy the streaming code
  in resolver.go for doing a forward search.
  * It could query `Flux` for past ABIs, and use the `payload` in the search
    results, and navigate forks through there.
* Downside: wouldn't scale independently of graphql usage

Micro-service approach
----------------------

// abicodec: handles Decode("eosio", "globals", 70000)
// Flux: here's the ABI
// abicodec: call flux.FetchABI("eosio", HEAD)
// abicodec: recursively call, until you reach block 1

* Would need to be very close (colocated with grapheos) for ultra low latency.
  Speed of this service is critical.


Flux-based approach
-------------------

* Flux could its own cache, with coverage ranges, etc... Load from Bigtable when it doesn't know.
