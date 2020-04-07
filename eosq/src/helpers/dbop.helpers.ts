import { DbOp, StateAbiToJsonResponse } from "@dfuse/client"

interface KeyValues {
  [key: string]: string[]
}

export interface ABIDecoderParams {
  table: string
  account: string
  hex_rows: string[]
  block_num: number
}

export function groupDBOpHex(dbops: DbOp[]): KeyValues {
  const groupedDBOp: KeyValues = {}
  dbops.forEach((dbop: DbOp) => {
    const key = `${dbop.account}::${dbop.table}`
    if (groupedDBOp[key]) {
      groupedDBOp[key] = addDBOpHex(groupedDBOp[key], dbop)
    } else {
      groupedDBOp[key] = addDBOpHex([], dbop)
    }
  })

  return groupedDBOp
}

export function addDBOpHex(groupedDBOp: string[], dbop: DbOp) {
  if (dbop.old && dbop.old.hex) {
    groupedDBOp.push(dbop.old.hex)
  }

  if (dbop.new && dbop.new.hex) {
    groupedDBOp.push(dbop.new.hex)
  }

  return groupedDBOp
}

export function decodedResponseToDBOps(
  responses: (StateAbiToJsonResponse<any> | undefined)[],
  dbops: DbOp[]
): DbOp[] {
  let decodedDBOps: DbOp[] = []
  responses.forEach((response) => {
    if (!response) {
      return
    }

    let index = 0
    const tmpDBOps = dbops
      .filter((dbop: DbOp) => dbop.table === response.table && dbop.account === response.account)
      .map((dbop: DbOp) => {
        if (dbop.old && dbop.old.hex) {
          dbop.old.json = response.rows[index]
          index += 1
        }

        if (dbop.new && dbop.new.hex) {
          dbop.new.json = response.rows[index]
          index += 1
        }
        return dbop
      })

    decodedDBOps = decodedDBOps.concat(tmpDBOps)
  })

  return decodedDBOps
}
