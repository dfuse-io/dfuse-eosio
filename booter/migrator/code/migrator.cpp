#include "migrator.hpp"

using namespace eosio;

#define IMPORT extern "C" __attribute__((eosio_wasm_import))


IMPORT int32_t db_store_i64(uint64_t scope, uint64_t table, uint64_t payer, uint64_t id,  const void* data, uint32_t len);

void migrator::inject(name table,name scope,name payer,name id, std::vector<char>  data) {            
    db_store_i64(
      scope.value,      // The scope where the record will be stored
      table.value,      // The ID/name of the table within the current scope/code context
      payer.value,      // The account that is paying for this storage
      id.value,         // Id of the entry
      (void*)&data[0],  // Record to store
      data.size()       // Size of data
    );  
};