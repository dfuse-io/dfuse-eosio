import { computed, observable } from "mobx"
import { AbiLoader } from "../services/abi-loader"
import { GetTableRowParams } from "../clients/websocket/eosws"

export class ContractTableStore {
  limit = 100
  @observable abiLoader?: AbiLoader

  @observable scope = ""
  @observable lowerBound?: string
  @observable upperBound?: string
  @observable tableName = ""
  @observable offset = 0
  @observable accountName = ""
  @observable tableKey?: string
  @observable tableRows?: any
  @observable loading = false
  @observable error = false

  @computed get nRows(): number {
    return this.tableRows && this.tableRows.rows ? this.tableRows.rows.length : 0
  }

  @computed get params(): GetTableRowParams {
    return {
      json: true,
      scope: this.scope,
      lower_bound:
        this.lowerBound && this.lowerBound.toString().length > 0 ? this.lowerBound : undefined,
      upper_bound:
        this.upperBound && this.upperBound.toString().length > 0 ? this.upperBound : undefined,
      limit: this.limit,
      code: this.accountName,
      table: this.tableName,
      table_key: this.tableKey ? this.tableKey : undefined
    }
  }

  initFromUrlParams(abiLoader: AbiLoader, accountName: string, params: any) {
    this.accountName = accountName
    this.abiLoader = abiLoader
    this.tableKey = this.abiLoader.getTableFirstKey(this.tableName)
    this.tableName = params.tableName || this.abiLoader.tableNames[0] || ""
    this.lowerBound = params.lowerBound
    this.upperBound = params.upperBound
    this.scope = params.scope ? params.scope : accountName
  }

  get firstTableKey(): string | undefined {
    return this.abiLoader ? this.abiLoader.getTableFirstKey(this.tableName) : undefined
  }

  get urlParams() {
    return {
      lowerBound: this.lowerBound,
      upperBound: this.upperBound,
      scope: this.scope ? this.scope : this.accountName,
      offset: this.offset <= 0 ? undefined : this.offset,
      tableName: this.tableName
    }
  }
}
