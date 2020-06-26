# Bootstrapping a new chain created by dfuse for EOSIO

The [bootseq.yaml](./bootseq.yaml) files contains sample operations executed to bootstrap a fresh chain. The master key  (genesis and eosio@active) is available in the [bootseq.yaml](./bootseq.yaml) as well as  in `eosc-vault.json` (with an empty password) for your convenience.  

By default, `dfuse-eosio` will look for `bootseq.yaml` in your working directory. Moreover, you can override this behavior by providing it with flag `--booter-bootseq` that contains the path to the `bootseq.yaml` file.