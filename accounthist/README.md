Account history service
=======================

This service keeps a partial, windowed history for each EOSIO account,
truncating after a given number of actions.

It is different from dfuse Search, in that it doesn't truncate based
on time, but on amount of actions.

As it keeps, say, 1000 actions per account, it can keep very old ones
when there aren't many actions by/for a given account, and it will
truncate aggressively on those accounts that spew mucho actions to the
chain.

It brings a nice balance of value per byte kept.
