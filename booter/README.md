## App `booter`

The `booter` responsibility is to bootstrap a chain with the necessary state so you can quickly
start working on real development or production data.

#### Export

##### _Folder Structure_
​
Accounts are distributed using a simple algorithm. We create two levels, the first one using the first two characters of the account, the second level using the subsequent two. All dots are replaced by `_` (to avoid any problems due to an account having two subsequent `.`). If there is not enough characters, the folder is created in the first full level.
​
Here are a few example and their resulting path:
​
- `a` => `migration-data/a/<data>`
- `b1` => `migration-data/b1/<data>`
- `t33` => `migration-data/t3/t33/<data>`
- `four` => `migration-data/fo/four/<data>`
- `eosio` => `migration-data/eo/si/eosio/<data>`
- `eosio.token` => `migration-data/eo/si/eosio_token/<data>`
- `......name5` => `migration-data/__/__/______name5/<data>`

​
Each account folder contains at its root a file `account.json`. If the account is also a contract, it will also contain an `abi.json` and a `code.wasm` (it's possible there is no ABI, if the account didn't have one).
​
The `account.json` contains the actual permissions structure as well as any linked permissions. It will eventually contain resource limits of the account, and maybe other account-related structure.
​
*Note* The `account.json` actual content structure might change in the next version.
​
If the account is a contract, it will contain this additional sub-structure:
​
```
- migration-data/eo/si/eosio/
  - account.json
  - abi.json
  - code.wasm
  - tables/
    - <table1>/
      - sc/
         - op/
            - scope1/
              - rows.json
      - ot/
         - he/
            - other1/
              - rows.json
    - <table2>/
    - ...
    - <tableN>/
```
​
Under the `tables` folder, you will find in a single level for all table names for which some data was present. In the actual table name folder (i.e. `tables/<table1>`) you will find a list of scopes, using the same layout algorithm as the account layout described above. This is because some tables can have a lot of different scopes.
​
Finally, for each table/scope pair, the final folder contains a `rows.json` file containing the exported table rows for this table/scope pair.
​
Here a sample file content:
​
```
[
  {"key":"","payer":"battlefield3","json_data":{"expires_at":"1970-01-01T00:00:00","created_at":"2020-06-25T16:44:57","memo":"from onerror handler","amount":"0","account":"onerror","id":0}}
,
  {"key":"............1","payer":"battlefield3","json_data":{"expires_at":"1970-01-01T00:00:00","created_at":"2020-06-25T16:44:57","memo":"from nested onerror handler","amount":"0","account":"nestonerror","id":1}}
]
```
​
It's a JSON array, each element being a row. The `key` is the primary key of the row. The `payer` is the account that is going to pay the RAM to store the row. Finally, you will get a `json_data` field filled with JSON value. The field will be present only if we're able to decode the raw row binary value to JSON using the account's ABI. If it's not the case, you will instead see a `hex_data` field populated with hexadecimal encoded value of the row.
​
At import, if `json_data` is present, we transform it back to binary using the contract's ABI. If it's `hex_data`, we inject it as-is.
​
*Note* The `rows.json` actual content might change in the next version to add support for secondary indexes. It's not clear how it's going to look, it may be a different file altogether.
​
Everything described is up for discussion and changes, don't hesitate to report your feedback.

#### Import

Here the general flow of how operation `` is interpreted and how it actually perform the
import phase

- Create all accounts (Step #1)
  - Walk all account
    - For each account
      - Action `newaccount` (`owner` uses <boot_key>, `active` <boot_key>)

- Import contract's data (Step #2)
  - List all contract
    - For each contract
      - Temporarily `setcode/setabi` on `account` to `dfuse.mgrt` contract
      - For each contract's table
        - For each contract's scope
          - For each row in `rows.json`
            - Import table data using `inject` actions

- Adjust all permissions (Step #3)
 - List all account
   - For each account
     - Reconstruct permission tree parent/child by reading `account.json`
     - Breath-first traversal of permission tree, skip `owner` permission
       - If `permissionName == 'active'` && `Step #1 @ active authority == permission's authority`, skip
       - Else create permission (`name`, `parent`, `authority`) using authorization `<account>@owner`
     - Perform `owner` root permission
       - If `Step #1 @ owner authority == permission's authority`, skip
       - Else create permission (`name`, `parent`, `authority`) using authorization `<account>@owner`
