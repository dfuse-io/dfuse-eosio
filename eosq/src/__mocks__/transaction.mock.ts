import { TransactionReceiptStatus } from "../models/transaction"
import {
  ExtDTrxOp,
  TransactionLifecycle,
  TransactionTrace,
  ActionTrace,
  Transaction,
  Action
} from "@dfuse/client"
import { TraceInfo } from "../models/pill-templates"

interface ActionMockParams {
  data: any
  account?: string
  name?: string
}

export function getTraceInfoMock(params: ActionMockParams): TraceInfo {
  return {
    receiver: "receiver",
    inline_traces: [getActionTraceMock(params)]
  }
}

export function getActionTraceMock(params: ActionMockParams): ActionTrace<any> {
  return {
    act: getActionMock(params),
    elapsed: 123,
    console: "console",
    block_num: 38,
    block_time: "2019-05-23T19:18:54.500",
    context_free: false,
    trx_id: "trx_id",
    inline_traces: [],
    receipt: {
      receiver: "receiver",
      act_digest: "act_digest",
      global_sequence: 123,
      recv_sequence: 123,
      auth_sequence: [["", 123]],
      code_sequence: 123,
      abi_sequence: 123
    }
  }
}

export function getActionMock(params: ActionMockParams): Action<any> {
  return {
    account: params.account ? params.account : "eosio",
    name: params.name ? params.name : "transfer",
    authorization: [
      {
        actor: "actor",
        permission: "permission"
      }
    ],
    hex_data: "hex_data",
    data: params.data
  }
}

export const getTransferDataMock = () => {
  return {
    from: "eosio.ram",
    to: "valuenetwork",
    quantity: "0.1508 EOS",
    memo: "0aa7674699771f0ca61b40db92c385fa5a6cd34100bb478f487b8c2badb861f9"
  }
}

export const getIssueDataMock = () => {
  return {
    to: "gqytknjugene",
    quantity: "1.2488 COA",
    memo: ""
  }
}

export const getRefundDataMock = () => {
  return {
    owner: "ge3tgojzhage"
  }
}

export const getTransactionMock = () => {
  return {
    id: "transactionid",
    block_num: 1234,
    block_id: "blockId",
    producer: "producer",
    ref_block_num: 1234,
    ref_block_prefix: 1234,
    max_net_usage_words: 2,
    delay_sec: 2,
    expiration: "expiration",
    max_cpu_usage_ms: 2,
    context_free_actions: [getActionMock({ data: {} })],
    actions: [getActionMock({ data: {} })],
    transaction_extensions: [],
    signatures: [],
    public_keys: [],
    irreversible: false
  } as Transaction
}

export function getTransactionRefMock(): ExtDTrxOp {
  return {
    src_trx_id: "src_trx_id",
    block_num: 123,
    block_id: "block_id",
    block_time: "block_time",
    op: "CREATE",
    action_idx: 1,
    sender: "sender",
    sender_id: "sender_id",
    payer: "sender",
    published_at: "published_at",
    delay_until: "delay_until",
    expiration_at: "expiration_at",
    trx_id: "trx_id"
  } as ExtDTrxOp
}

export function getTransactionTraceMock(): TransactionTrace {
  return {
    id: "id",
    receipt: {
      status: "soft_fail",
      cpu_usage_us: 12,
      net_usage_words: 12
    },
    except: null,
    elapsed: 1,
    net_usage: 1,
    scheduled: false,
    action_traces: [getActionTraceMock({ data: {} })],
    producer_block_id: "abc",
    block_num: 1234,
    block_time: "block_time"
  }
}

export const getTransactionLifeCycleMock = () => {
  return {
    id: "id",
    execution_block_header: null!,
    dtrxops: [],
    dbops: [],
    ramops: [],
    pub_keys: [],
    execution_irreversible: true,
    creation_irreversible: true,
    cancelation_irreversible: false,
    transaction: getTransactionMock(),
    created_by: getTransactionRefMock(),
    canceled_by: getTransactionRefMock(),
    execution_trace: getTransactionTraceMock(),
    delayed: false,
    transaction_status: TransactionReceiptStatus.EXECUTED
  } as TransactionLifecycle
}

export function generateTransactionTrace(
  index: number = 0,
  inlineTraces: ActionTrace<any>[] = []
): ActionTrace<any> {
  const actions = generateActions(0, 0)

  return {
    receipt: {
      abi_sequence: 123,
      code_sequence: 123,
      auth_sequence: [["abc", 1]],
      global_sequence: 354566,
      recv_sequence: 1255,
      receiver: "tom",
      act_digest: "a4d4a981af"
    },
    block_num: 38,
    block_time: "2019-05-23T19:18:54.500",
    context_free: false,
    act: actions[0],
    elapsed: 912,
    console: "something",
    trx_id: `${index}1asasa1212`,
    inline_traces: inlineTraces
  }
}

export function generateActions(actionIndexStart?: number, actionIndexEnd?: number): Action<any>[] {
  const bucketOfActions = [
    {
      authorization: [
        {
          actor: "hezdsnryhage",
          permission: "active"
        }
      ],
      data: {
        active: {
          accounts: [],
          keys: [
            {
              key: "EOS7F5HfUiiVeqnRsgisk7NvmB4Mxrgn8ZhQ9wZqc73QezDUYDuYs",
              weight: 1
            }
          ],
          threshold: 1,
          waits: []
        },
        creator: "hezdsnryhage",
        name: "superoneiobp",
        owner: {
          accounts: [],
          keys: [
            {
              key: "EOS5VqKPX82SaFzgKpcmhuBRaYmVozGeHTN5qQWPgXjhP2QUjLNtT",
              weight: 1
            }
          ],
          threshold: 1,
          waits: []
        }
      },
      hex_data: "asa",
      account: "eosio",
      name: "newaccount"
    },
    {
      authorization: [
        {
          actor: "hezdsnryhage",
          permission: "active"
        }
      ],
      data: {
        payer: "hezdsnryhage",
        quant: "1.0000 EOS",
        receiver: "superoneiobp"
      },
      hex_data: "a09869fe4e9cbe6a500f756ad2abaac6102700000000000004454f5300000000",
      account: "eosio",
      name: "buyram"
    },
    {
      authorization: [
        {
          actor: "hezdsnryhage",
          permission: "active"
        }
      ],
      data: {
        from: "hezdsnryhage",
        receiver: "superoneiobp",
        stake_cpu_quantity: "2.0000 EOS",
        stake_net_quantity: "2.0000 EOS",
        transfer: 1
      },
      hex_data:
        "a09869fe4e9cbe6a500f756ad2abaac6204e00000000000004454f5300000000204e00000000000004454f530000000001",
      account: "eosio",
      name: "delegatebw"
    }
  ]

  actionIndexStart = actionIndexStart || Math.floor(Math.random() * 3)
  actionIndexEnd = actionIndexEnd || Math.floor(Math.random() * 3)

  return bucketOfActions.slice(actionIndexStart, actionIndexEnd)
}
