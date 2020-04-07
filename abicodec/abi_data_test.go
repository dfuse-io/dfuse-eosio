// Copyright 2019 dfuse Platform Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package abicodec

var ABI_TRANSFER = `
{
    "version": "eosio::abi/1.1",
    "structs": [
      {
        "name": "account",
        "base": "",
        "fields": [
          {
            "name": "balance",
            "type": "asset"
          }
        ]
      },
      {
        "name": "close",
        "base": "",
        "fields": [
          {
            "name": "owner",
            "type": "name"
          },
          {
            "name": "symbol",
            "type": "symbol"
          }
        ]
      },
      {
        "name": "create",
        "base": "",
        "fields": [
          {
            "name": "issuer",
            "type": "name"
          },
          {
            "name": "maximum_supply",
            "type": "asset"
          }
        ]
      },
      {
        "name": "currency_stats",
        "base": "",
        "fields": [
          {
            "name": "supply",
            "type": "asset"
          },
          {
            "name": "max_supply",
            "type": "asset"
          },
          {
            "name": "issuer",
            "type": "name"
          }
        ]
      },
      {
        "name": "issue",
        "base": "",
        "fields": [
          {
            "name": "to",
            "type": "name"
          },
          {
            "name": "quantity",
            "type": "asset"
          },
          {
            "name": "memo",
            "type": "string"
          }
        ]
      },
      {
        "name": "open",
        "base": "",
        "fields": [
          {
            "name": "owner",
            "type": "name"
          },
          {
            "name": "symbol",
            "type": "symbol"
          },
          {
            "name": "ram_payer",
            "type": "name"
          }
        ]
      },
      {
        "name": "retire",
        "base": "",
        "fields": [
          {
            "name": "quantity",
            "type": "asset"
          },
          {
            "name": "memo",
            "type": "string"
          }
        ]
      },
      {
        "name": "transfer",
        "base": "",
        "fields": [
          {
            "name": "from",
            "type": "name"
          },
          {
            "name": "to",
            "type": "name"
          },
          {
            "name": "quantity",
            "type": "asset"
          },
          {
            "name": "memo",
            "type": "string"
          }
        ]
      }
    ],
    "actions": [
      {
        "name": "close",
        "type": "close",
        "ricardian_contract": ""
      },
      {
        "name": "create",
        "type": "create",
        "ricardian_contract": ""
      },
      {
        "name": "issue",
        "type": "issue",
        "ricardian_contract": ""
      },
      {
        "name": "open",
        "type": "open",
        "ricardian_contract": ""
      },
      {
        "name": "retire",
        "type": "retire",
        "ricardian_contract": ""
      },
      {
        "name": "transfer",
        "type": "transfer",
        "ricardian_contract": "## Transfer Terms \u0026 Conditions\n\nI, {{from}}, certify the following to be true to the best of my knowledge:\n\n1. I certify that {{quantity}} is not the proceeds of fraudulent or violent activities.\n2. I certify that, to the best of my knowledge, {{to}} is not supporting initiation of violence against others.\n3. I have disclosed any contractual terms \u0026 conditions with respect to {{quantity}} to {{to}}.\n\nI understand that funds transfers are not reversible after the {{transaction.delay}} seconds or other delay as configured by {{from}}'s permissions.\n\nIf this action fails to be irreversibly confirmed after receiving goods or services from '{{to}}', I agree to either return the goods or services or resend {{quantity}} in a timely manner.\n"
      }
    ],
    "tables": [
      {
        "name": "accounts",
        "index_type": "i64",
        "type": "account"
      },
      {
        "name": "stat",
        "index_type": "i64",
        "type": "currency_stats"
      }
    ]
  }
`
