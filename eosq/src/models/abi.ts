export enum AbiBaseTypes {
  NAME = "name",
  ACCOUNT_NAME = "account_name",
  PERMISSION_NAME = "scope_name",
  TABLE_NAME = "table_name",
  TIME = "time",
  SCOPE_NAME = "scope_name",
  ACTION_NAME = "action_name",
  WEIGHT_TYPE = "weight_type",
  PUBLIC_KEY = "public_key",
  SIGNATURE = "signature",
  CHECKSUM256 = "checksum256",
  CHECKSUM160 = "checksum160",
  CHECKSUM512 = "checksum512",
  TRANSACTION_ID_TYPE = "transaction_id_type",
  BLOCK_ID_TYPE = "block_id_type"
}

export const ABI_BASE_TYPE_LENGTH = {
  [AbiBaseTypes.NAME]: 12,
  [AbiBaseTypes.ACCOUNT_NAME]: 12,
  [AbiBaseTypes.PERMISSION_NAME]: 12,
  [AbiBaseTypes.TABLE_NAME]: 12,
  [AbiBaseTypes.TIME]: 16,
  [AbiBaseTypes.SCOPE_NAME]: 12,
  [AbiBaseTypes.ACTION_NAME]: 12,
  [AbiBaseTypes.WEIGHT_TYPE]: 16,
  [AbiBaseTypes.PUBLIC_KEY]: 34,
  [AbiBaseTypes.SIGNATURE]: 66,
  [AbiBaseTypes.CHECKSUM256]: 32,
  [AbiBaseTypes.CHECKSUM160]: 20,
  [AbiBaseTypes.CHECKSUM512]: 64,
  [AbiBaseTypes.CHECKSUM160]: 12,
  [AbiBaseTypes.TRANSACTION_ID_TYPE]: 32,
  [AbiBaseTypes.BLOCK_ID_TYPE]: 32
}
