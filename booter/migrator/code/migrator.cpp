#include "migrator.hpp"

using namespace eosio;

#define IMPORT extern "C" __attribute__((eosio_wasm_import))


IMPORT int32_t db_store_i64(uint64_t scope, uint64_t table, uint64_t payer, uint64_t id,  const void* data, uint32_t len);


void migrator::inject(const uint64_t scope,const uint64_t table,const uint64_t payer,const uint64_t id, void* data, const uint32_t len) {
    printf("scope: %d, table: %d, payer: %d,id: %d, size of data %d", scope,table,payer,id, len);
    // data is a void pointer, it holds the address of a data time
    
    // db_store_i64(
    //   scope,  // The scope where the record will be stored
    //   table,  // The ID/name of the table within the current scope/code context
    //   payer,  // The account that is paying for this storage
    //   id,     // Id of the entry
    //   data,   // Record to store
    //   len     // Size of data
    // );  
};