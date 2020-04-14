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

package sql

var createTrxsTableStmt = `CREATE TABLE IF NOT EXISTS trxs (
    id varchar(64) NOT NULL,
    blockId varchar(64) NOT NULL,

    signedTrx blob NOT NULL,
    receipt blob,             -- Will be NULL if we're talking about an Internal Addition
    publicKeys blob,          -- Will be NULL if it's an internal addition

    PRIMARY KEY (id, blockId)
);`

var createImplicitTrxsTableStmt = `CREATE TABLE IF NOT EXISTS implicittrxs (
    id varchar(64) NOT NULL,
    blockId varchar(64) NOT NULL,

    signedTrx blob NOT NULL,
    name varchar(12) NOT NULL,

    PRIMARY KEY (id, blockId)
);`

var createIrrBlocksTableStmt = `CREATE TABLE IF NOT EXISTS irrblks (
    id varchar(64) NOT NULL PRIMARY KEY,

    irreversible boolean NOT NULL
);`

var createTracesTableStmt = `CREATE TABLE IF NOT EXISTS trxtraces (
    id varchar(64) NOT NULL,
    blockId varchar(64) NOT NULL,

    trace blob,

    PRIMARY KEY (id, blockId)
);`

// TODO: put creations and cancelations in two separate tables rather,
// as we'll need to create yet another event from the contents of this table.
var createDtrxsTableStmt = `CREATE TABLE IF NOT EXISTS dtrxs (
    id varchar(64) NOT NULL,
    blockId varchar(64) NOT NULL,

    signedTrx blob,
    createdBy blob,
    canceledBy blob,

    PRIMARY KEY (id, blockId)
);`

var createBlocksTableStmt = `CREATE TABLE IF NOT EXISTS blks (
    id varchar(64) NOT NULL PRIMARY KEY,
    number int unsigned NOT NULL,
    previousId varchar(64),
    irrBlockNum int unsigned NOT NULL,

    blockProducer varchar(13) NOT NULL,
    blockTime datetime(6) NOT NULL,

    block blob,
    blockHeader blob,

    implicitTrxRefs blob,
    trxRefs blob,
    traceRefs blob
);`

var createBlocksBlksTimeIndexesStmt = `
	CREATE INDEX blks_time ON blks (blockTime);
`
var createBlocksBlksNumIndexesStmt = `
	CREATE INDEX blks_num ON blks (number);
`

// TODO: add an `accountCreationSummary` in `pbdeos` and use that here instead of
//       creationJson.
// Eventually EXPOSE IT in GraphQL (!!!)
// TODO: this needs a `blockId` to be navigated, and use the irreversible one
var createAccountsTableStmt = `CREATE TABLE IF NOT EXISTS accts (
    account varchar(13) NOT NULL,
    blockId varchar(64) NOT NULL,
    blockNum int unsigned NOT NULL,
    blockTime datetime(6) NOT NULL,
    trxId varchar(64) NOT NULL,

    creator varchar(13) NOT NULL,

    PRIMARY KEY (account, blockId)
);`
