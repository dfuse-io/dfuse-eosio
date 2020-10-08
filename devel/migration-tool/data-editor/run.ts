import { Migrator, Element, Account, TableScope } from '@dfuse/migrator-client'
const path = require("path");

const onAccount = (account: Account): Account | undefined => {
    console.log(`received account: ${account.name}`)
    // To delete an account, return `undefined`, will not walk rows at this point anymore
    if (account.name === "battlefield4") {
        return undefined
    }

    // To modify an account, return the account with the desired changes
    if (account.name === "battlefeld3") {
        account.abi = "ABI 2.0/"
        // account.limits = {
        //   cpu: -1,
        //   net: -1,
        //   ram: -1,
        // }
        return account
    }

    return account
}

const onTable = (tableScope: TableScope): TableScope | undefined => {
    console.log(`received table scope: ${tableScope.contract} ${tableScope.tableName} with ${tableScope.rows.length} rows`)

    // To delete all table/scope pair for a given table
    if (tableScope.contract === "eosio.token" && tableScope.tableName === "accounts") {
        return undefined
    }

    // Modifiy the rows of a table/scope pair
    if (tableScope.contract === "eosio.token" && tableScope.tableName === "accounts") {
        if (tableScope.scope === "battlefield3") {
            // Delete unwanted rows
            tableScope.rows = tableScope.rows.filter((row) => row.key > "")

            // Modify the remaining rows
            tableScope.rows = tableScope.rows.map((row) => {
                row.json_data = { ...row.json_data, memo: row.json_data.memo.tof }
                return row
            })

            // Add new json_data
            tableScope.rows.push({
                key: "accc", payer: "battlefield3", json_data: {
                    expires_at: "1970-01-01T00:00:00",
                    created_at: "2020-06-26T12:51:03",
                    memo: "new row added",
                    amount: "100",
                    account: "",
                    id: 10
                }, secondary_indexes: []
            })

            return tableScope
        }
    }

    return tableScope
}

const inputDir = path.join(__dirname,"..","migration-data")
const outputDir = path.join(__dirname,"..","edited-migration-data")
const migrator = new Migrator(inputDir, outputDir)

migrator.Migrate((element: Element) => {
    if (element.type === "account") {
        return onAccount(element)
    }

    if (element.type === "tableScope") {
        return onTable(element)
    }

    // Untouched
    return element
})

// Add new account after `Migrate` so they are not walked, it's where you also define list of new tables to add.
migrator.AddAccount("ultra.new", {
    abi: "ABI 1.0/",
    wasm: "wasm code!",
})

// Add rows to an existing account, override
migrator.AddTable("ultra.new", "accounts", "ultra.nft", [
    {
        key: "value",
        payer: "value",
        json_data: { name: "accounts" },
        secondary_indexes: [],
    },
])
