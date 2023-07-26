export default {
  transaction: {
    displayedTree: {
      creationTree: "Creation Tree",
      executionTree: "Execution Tree"
    },
    showMoreActions: {
      title: "+ {{extraActions}} action(s)",
      longTitle: "+ {{extraActions}} action(s) / + {{extraDeferred}} deferred operation(s)"
    },
    tableops: {
      operations: {
        INS: "CREATE TABLE",
        REM: "REMOVE TABLE"
      },
      label: "<0>{{operation}}</0> | table:<1>{{table}}</1> | scope:<2>{{scope}}</2> "
    },
    dbops: {
      operations: {
        INS: "INSERT ROW",
        UPD: "UPDATE ROW",
        REM: "REMOVE ROW"
      },
      label:
        "<0>{{operation}}</0> | table:<1>{{table}}</1> | scope:<2>{{scope}}</2> | primary key: <3>{{primaryKey}}</3>:"
    },
    ramUsage: {
      operations: {
        create_table: "creating table",
        deferred_trx_add: "storing deferred transaction",
        deferred_trx_cancel: "canceling deferred transaction",
        deferred_trx_pushed: "creating deferred transaction",
        deferred_trx_removed: "executing deferred transaction",
        deleteauth: "deleting authority",
        linkauth: "linking authority",
        newaccount: "creating new account",
        primary_index_add: "storing row (primary)",
        primary_index_remove: "removing row (primary)",
        primary_index_update: "updating row (primary)",
        primary_index_update_add_new_payer: "storing payer (primary)",
        primary_index_update_remove_old_payer: "removing payer (primary)",
        remove_table: "removing a table",
        secondary_index_add: "storing row (secondary)",
        secondary_index_remove: "removing row (secondary)",
        secondary_index_update_add_new_payer: "storing payer (secondary)",
        secondary_index_update_remove_old_payer: "removing payer (secondary)",
        setabi: "updating ABI for account",
        setcode: "updating contract for account",
        unlinkauth: "unlinking authority",
        updateauth_create: "creating new permission",
        updateauth_update: "updating permission",
        kv_add: "storing a key/value pair",
        kv_update: "updating a key/value pair",
        kv_remove: "removing a key/value pair",
        unknown: "unknown",
      },
      title: "Ram Usage Summary",
      consumed: "<0>{{accountName}}</0> consumed <1>{{bytes}}</1> (now has <2>{{totalBytes}}</2>)",
      released: "<0>{{accountName}}</0> released <1>{{bytes}}</1> (now has <2>{{totalBytes}}</2>)",
      consumedDetail: "<0>{{accountName}}</0> consumed <1>{{bytes}}</1> — <2>{{operation}}</2>",
      releasedDetail: "<0>{{accountName}}</0> released <1>{{bytes}}</1> — <2>{{operation}}</2>"
    },
    deferred: {
      delayedFor: "Delayed for",
      create:
        "<0>Created</0> deferred transaction <1>{{transactionId}}</1> delayed for <2>{{delay}}</2>",

      triggeredBy: {
        label: "Triggered By",
        content: "Failure of <0>{{transactionId}}</0> in block <1>{{blockNum}}</1>"
      },
      creationMethod: {
        label: "Creation Method",
        PUSH_CREATE: "Pushed directly to the chain",
        CREATE: "Created by a smart contract ",
        MODIFY_CREATE: "Modified by a smart contract"
      },
      cancel: "<0>Canceled</0> deferred transaction <1>{{transactionId}}</1>",
      createdBy: {
        label: "Created By",
        content: "<0>{{transactionId}}</0> in block <1>{{blockNum}}</1>"
      },
      canceledBy: {
        label: "Canceled By",
        content: "<0>{{transactionId}}</0> in block <1>{{blockNum}}</1>"
      }
    },
    status: {
      hard_fail: "Failed (hard)",
      soft_fail: "Failed (soft)",
      delayed: "Deferred",
      canceled: "Canceled",
      executed: "Executed",
      expired: "Expired",
      pending: "Pending"
    },
    pill: {
      console: "Console",
      dbOps: "DB Operations",
      ramOps: "RAM Operations",
      tableOps: "Table Operations",
      general: "General",
      jsonData: "JSON Data",
      hexData: "HEX Data",
      cpu_usage: "CPU Usage",
      total_cpu_usage: "Total CPU Usage",
      receiver: "Receiver",
      memo: "Memo:",
      account: "Contract account",
      action_name: "Action name",
      authorization: "Authorization"
    },
    loading: "Loading transaction",
    traces: {
      title: "Actions",
      empty: "No actions",
      raw: "Raw",
      memo: "Memo",
      pill: {
        names: {
          buyram: "Buy Ram"
        }
      }
    },
    notSeenYet: {
      notFound: "Transaction not found",
      watchingForNetwork: "Watching network for incoming transaction"
    },
    banner: {
      transaction_count: "Transaction Count",
      block_produced: "Block Produced",
      total_value: "Total Value",
      title: "Transaction"
    },
    blockPanel: {
      title: "Block",
      block: "Block #",
      blockId: "Block Id",
      age: "Timestamp",
      status: "Status",
      producer: "Producer",

      statuses: {
        notSeenYet: "Not yet seen",
        waiting: "Waiting",
        confidence: "Confidence",
        irreversible: "Executed"
      }
    },
    detailPanel: {
      producer: {
        unknown: "Unknown"
      },
      fullTrace: "Full Trace",
      title: "Transaction",
      hash: "Hash",
      status: "Status",
      expirationDate: "Expiration Date",
      cpuUsage: "CPU Usage",
      networkUsage: "Network Usage",
      authorizations: "Authorizations",
      signedBy: "Signed By",
      noUsage: "None",
      statuses: {
        executed: "Executed,",
        expired: "Expired",
        accepted: "awaiting irreversibility",
        irreversible: "irreversible",
        blockDeep: " blocks deep"
      }
    },
    progressCircle: {
      confidence: "Confidence"
    },
    list: {
      extendSearch: "Extend your Search",
      advancedOptions: "Advanced Options",
      noResultsExtend:
        "There are no results that match your search in the last {{lastBlocks}} blocks",
      noMoreResultsExtend: "There is no more results in the last {{lastBlocks}} blocks",
      empty: "No transactions",
      loading: "Loading transactions...",
      title: "Transactions",
      header: {
        timestamp: "Timestamp",
        id: "Transaction ID",
        blockId: "Block #",
        timeCreated: "Date - Time",
        expiration: "Expiration",
        blockTime: "Timestamp",
        account: "Account",
        contract: "Contract",
        action: "Action",
        value: "Value",
        summary: "Transaction / Block",
        moreActions: "More Actions"
      }
    }
  }
}
