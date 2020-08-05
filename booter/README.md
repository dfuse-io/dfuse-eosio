## App `booter`

The `booter` responsibility is to bootstrap a chain with the necessary state so you can quickly
start working on real development or production data.

#### Export

To be completed.

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
