import { decodedResponseToDBOps, groupDBOpHex } from "../helpers/dbop.helpers"
import { DbOp } from "@dfuse/client"
import { legacyHandleDfuseApiError } from "../clients/rest/api"
import { getDfuseClient } from "../data/dfuse"

export function decodeDBOps(dbops: DbOp[], blockNum: number, callback: (dbops: DbOp[]) => any) {
  const groupedDBOps = groupDBOpHex(dbops)
  const promises = Object.keys(groupedDBOps).map((groupKey: string) => {
    return getDfuseClient()
      .stateAbiBinToJson<any>(
        groupKey.split("::")[0],
        groupKey.split("::")[1],
        groupedDBOps[groupKey],
        { blockNum }
      )
      .catch(legacyHandleDfuseApiError)
  })

  Promise.all(promises).then((responses) => {
    callback(decodedResponseToDBOps(responses, dbops))
  })
}
