import {
  streamVoteTally,
  isInboundMessageType,
  streamAccount,
  GetTableRowParams
} from "../clients/websocket/eosws"
import { InboundMessage, InboundMessageType, ErrorData } from "@dfuse/client"
import { voteStore } from "../stores"
import { Account, RexFunds, RexBalance, RexLoan, BlockProducerInfo } from "../models/account"
import {
  requestAccountLinkedPermissions,
  requestContractTableRows,
  requestProducerAccountTableRows
} from "../clients/rest/account"
import { extractValueWithUnits } from "../helpers/formatters"

import { getDfuseClient } from "@dfuse/explore"

export async function registerAccountDetailsListeners(
  accountName: string,
  blockNum: number,
  successCallback: (account: Account) => any,
  errorCallback: (message: ErrorData) => any
) {
  const voteStream = await streamVoteTally(getDfuseClient(), (message: InboundMessage<any>) => {
    if ((message.type as any) === "vote_tally") {
      if (!message.data.vote_tally) {
        return
      }

      voteStore.update(message.data.vote_tally)
    }
  })

  const accountStream = await streamAccount(
    getDfuseClient(),
    accountName,
    (message: InboundMessage) => {
      if (message.type === InboundMessageType.ERROR) {
        errorCallback(message.data as ErrorData)
        return
      }

      if (!isInboundMessageType(message, "account")) {
        return
      }

      let { account } = message.data as { account: Account }
      let producerInfo: any

      const rexParams: GetTableRowParams = {
        json: true,
        scope: "eosio",
        table: "rexbal",
        code: "eosio",
        table_key: "",
        lower_bound: accountName,
        upper_bound: "",
        limit: 10
      }

      const rexfundsParams: GetTableRowParams = {
        json: true,
        scope: "eosio",
        table: "rexfund",
        code: "eosio",
        table_key: "",
        lower_bound: accountName,
        upper_bound: "",
        limit: 10
      }

      const cpuLoans: any = {
        code: "eosio",
        json: true,
        scope: "eosio",
        table: "cpuloan",
        lower_bound: accountName,
        upper_bound: "zzzzzzzzzzzz",
        limit: 100,
        index_position: "3",
        key_type: "name"
      }

      const netLoans: any = {
        code: "eosio",
        json: true,
        scope: "eosio",
        table: "netloan",
        lower_bound: accountName,
        upper_bound: "zzzzzzzzzzzz",
        limit: 100,
        index_position: "3",
        key_type: "name"
      }

      if (account) {
        Promise.all([
          requestProducerAccountTableRows(accountName),
          requestAccountLinkedPermissions(accountName, blockNum),
          requestContractTableRows(rexParams),
          requestContractTableRows(rexfundsParams),
          requestContractTableRows(cpuLoans),
          requestContractTableRows(netLoans)
        ])
          .then((response: any) => {
            if (response && response.length >= 2) {
              if (response[0]) {
                account = addProducerInfoToAccount(account, response[0])
              }

              if (response[1].linked_permissions) {
                account.linked_permissions = response[1].linked_permissions
              }

              successCallback(account)
              if (response.length >= 3 && response[2]) {
                account = addRexTokensToAccount(account, response[2])
              }

              if (response.length >= 4 && response[3]) {
                account = addRexFundsToAccount(account, response[3])
              }

              if (response.length >= 5 && response[4]) {
                account = addRexCpuLoanToAccount(account, response[4])
              }

              if (response.length >= 6 && response[5]) {
                account = addRexNetLoanToAccount(account, response[5])
              }
            }
          })
          .catch(() => {
            producerInfo = { rows: [] }
            account = addProducerInfoToAccount(account, producerInfo)
            account.linked_permissions = []
            successCallback(account)
          })
      }
    }
  )

  return {
    voteStream,
    accountStream
  }
}

function parseProducerInfo(data: any, accountName: string): BlockProducerInfo | undefined {
  if (data && data.rows && data.rows[0]) {
    const blockProducerInfo = JSON.parse(data.rows[0].json) as BlockProducerInfo
    if (blockProducerInfo.producer_account_name === accountName) {
      return blockProducerInfo
    }
  }
  return undefined
}

function addProducerInfoToAccount(account: Account, producerInfo: any): Account {
  if (producerInfo && producerInfo.rows && producerInfo.rows[0]) {
    account.block_producer_info = parseProducerInfo(producerInfo, account.account_name)
  }

  return account
}

function addRexTokensToAccount(account: Account, rexTokens: any): Account {
  if (rexTokens && rexTokens.rows && rexTokens.rows[0]) {
    const rexTokensData = rexTokens.rows.find((row: RexBalance) => {
      return account.account_name === row.owner
    })
    account.rex_balance = rexTokensData
    return account
  }
  return account
}

function addRexFundsToAccount(account: Account, rexTokens: any): Account {
  if (rexTokens && rexTokens.rows && rexTokens.rows[0]) {
    const rexTokensData = rexTokens.rows.find((row: RexFunds) => {
      return account.account_name === row.owner
    })
    account.rex_funds = rexTokensData
    return account
  }
  return account
}

function addRexCpuLoanToAccount(account: Account, rexTokens: any): Account {
  if (rexTokens && rexTokens.rows && rexTokens.rows[0]) {
    const rexTokensData = rexTokens.rows.filter((row: RexLoan) => {
      return account.account_name === row.from
    })
    account.cpu_loans = rexTokensData.reduce((sum: number, row: RexLoan) => {
      return sum + parseFloat(extractValueWithUnits(row.balance)[0])
    }, 0)
    return account
  }
  return account
}

function addRexNetLoanToAccount(account: Account, rexTokens: any): Account {
  if (rexTokens && rexTokens.rows && rexTokens.rows[0]) {
    const rexTokensData = rexTokens.rows.filter((row: RexLoan) => {
      return account.account_name === row.from
    })
    account.net_loans = rexTokensData.reduce((sum: number, row: RexLoan) => {
      return sum + parseFloat(extractValueWithUnits(row.balance)[0])
    }, 0)
    return account
  }
  return account
}
