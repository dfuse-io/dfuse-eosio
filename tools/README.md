# Putting some transactions in an empty new chain ('local' dfuse BP)

## set up some system contracts
eosc -u http://localhost:8888 boot bootseq.yaml   ## password is empty

## do at least a transfer...
eosc -u http://localhost:8888 transfer eosio eosio2 20 ## password is empty
