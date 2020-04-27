# Bootstrapping a new chain created by dfuse for EOSIO

The [genesis.key](./genesis.key) and [genesis.pub](./genesis.pub) files contain the master key (genesis and eosio@active). That key is also available in eosc-vault.json (with an empty password) for your convenience.

* First, get [eosc](https://github.com/eoscanada/eosc)

* Execute the content of `bootseq.yaml`

    eosc -u http://localhost:8888 boot bootseq.yaml   ## password is empty

* Do some transfers

    eosc -u http://localhost:8888 transfer eosio eosio2 20 ## password is empty

* Create an account

    eosc system  newaccount  eosio myaccount --auth-key EOS3245345....(your-public-key) --stake-cpu 10 --stake-net 10
