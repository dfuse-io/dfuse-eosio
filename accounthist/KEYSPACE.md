
Tables
------


message ActionData {
  string trx_id = 1;
  deos.BlockRef block_ref = 2;
  deos.ActionTrace action_trace = 3;
}

max window = 3

block: 1

put: [02][eosio][^00000001] -> glob_seq,action_data
#put: [01][eosio] -> seq=1,prev=0,prevBlock=0

put: [03] -> block 1

block: 4

put: [02][eosio][^00000002] -> action234,glob_seq
#put: [01][eosio] -> seq=2,prev=1,prevBlock=1

--

put: [02][eosio][^00000003] -> action345
#put: [01][eosio] -> seq=3

put: [03] -> block 4

block: 5

put: [02][eosio][^00000004] -> action456
#put: [01][eosio] -> seq=4
del: [02][eosio][^00000001] -> action123

put: [03] -> block 5
