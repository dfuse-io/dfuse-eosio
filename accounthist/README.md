## Account history service

This service keeps a partial, windowed history for each EOSIO account,
truncating after a given number of actions.

It is different from dfuse Search, in that it doesn't truncate based
on time, but on amount of actions.

As it keeps, say, 1000 actions per account, it can keep very old ones
when there aren't many actions by/for a given account, and it will
truncate aggressively on those accounts that spew mucho actions to the
chain.

It brings a nice balance of value per byte kept.

### Usage

```
grpcurl -plaintext -d '{"account": "eosio"}' localhost:13033 dfuse.eosio.accounthist.v1.AccountHistory.GetActions
```

### Keyspace

#### Tables

```
max window = 3

block: 1

put: [02][eosio][^00000001] -> glob_seq,action_data
#put: [01][eosio] -> seq=1,prev=0,prevBlock=0

put: [03] -> block 1

block: 4

02:00000ef3438a:ff:ffffffffff600 -> 1000
02:00000ef3438a:ff:ffffffffffff7
02:00000ef3438a:ff:ffffffffffff6
02:00000ef3438a:fc:fffffffffffff
02:00000ef3438a:fc:ffffffffffffe
02:00000ef3438a:60:fffffffffffff


02:00000ef3438a:ff:ffffffffffffe
02:00000ef3438a:ff:fffffffffffff
02:00000ef3438a:fe:ffffffffffffe
02:00000ef3438a:fd:ffffffffffffd
02:00000ef3438a:fc:ffffffffffffc
put: [02][eosio][^00000002] -> action234,glob_seq
#put: [01][eosio] -> seq=2,prev=1,prevBlock=1

--

put: [02][eosio][^00000003] -> action345
#put: [01][eosio] -> seq=3

put: [03][ff] -> firstblockofshard: 10000, target: 0 (infinity), block 2_000_000
put: [03][fe] -> firstblockofshard: 5000, target: 9999, block 8000
put: [03][f4] -> firstblockofshard: 1, target: 4999, block 4000

block: 5

put: [02][eosio][^00000004] -> action456
del: [02][eosio][^00000001] -> action123

put: [03] -> block 5

---

put: [04][ff] /* first */ -> block 120M
put: [03][ff] /* last */ -> block 150M
```