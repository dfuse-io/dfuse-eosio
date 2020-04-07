import { Abi, AbiStruct, AbiStructField, AbiTable, AbiType } from "@dfuse/client"

export class AbiLoader {
  abi: Abi

  constructor(abi: Abi) {
    this.abi = abi
  }

  get tables(): AbiTable[] {
    return this.abi.tables
  }

  get tableNames(): string[] {
    return this.tables.map((table: AbiTable) => table.name)
  }

  get tableTypes(): string[] {
    return this.tables.map((table: AbiTable) => table.type)
  }

  get structs(): AbiStruct[] {
    return this.abi.structs
  }

  get baseTypes(): AbiType[] {
    return this.abi.types
  }

  getTableFirstKey(tableName: string): string | undefined {
    const table = this.getTable(tableName)
    if (table && table.key_names === undefined) {
      const structRef = this.structs.find((struct: AbiStruct) => {
        return struct.name === table.type
      })

      if (structRef) {
        return structRef.fields[0].name
      }
    }

    return table && table.key_names && table.key_names.length > 0 ? table.key_names[0] : undefined
  }

  get actionNames(): string[] {
    return this.abi.actions.map((action) => action.name)
  }

  get composedTypes(): AbiStruct[] {
    return this.abi.structs.filter((struct: any) => {
      return !this.actionNames.includes(struct.name) && !this.tableTypes.includes(struct.name)
    })
  }

  get actionStructs(): AbiStruct[] {
    return this.abi.structs.filter((struct: AbiStruct) => {
      return this.actionNames.includes(struct.name)
    })
  }

  getTable(tableName: string): AbiTable | undefined {
    return this.tables.find((table: AbiTable) => table.name === tableName)
  }

  getTableStructFromType(tableType: string): AbiStruct | undefined {
    return this.structs.find((struct: AbiStruct) => struct.name === tableType)
  }

  getTableFields(tableName: string): AbiStructField[] {
    const table = this.getTable(tableName)
    const tableType = table ? table.type : ""
    const tableStruct = this.getTableStructFromType(tableType)
    return tableStruct ? tableStruct.fields : []
  }
}
