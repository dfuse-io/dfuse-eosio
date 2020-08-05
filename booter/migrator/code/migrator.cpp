#include "migrator.hpp"

#define IMPORT extern "C" __attribute__((eosio_wasm_import))

extern "C" {
  #include "eosio/types.h"
}

using namespace eosio;

IMPORT int32_t db_store_i64(uint64_t scope, uint64_t table, uint64_t payer, uint64_t id,  const void* data, uint32_t len);
IMPORT void db_remove_i64(int32_t iterator);
IMPORT int32_t db_idx64_store(uint64_t scope, capi_name table, capi_name payer, uint64_t id, const uint64_t* secondary);
IMPORT int32_t db_idx128_store(uint64_t scope, capi_name table, capi_name payer, uint64_t id, const uint128_t* secondary);
IMPORT int32_t db_idx256_store(uint64_t scope, capi_name table, capi_name payer, uint64_t id, const uint128_t* data, uint32_t data_len );
IMPORT int32_t db_idx_double_store(uint64_t scope, capi_name table, capi_name payer, uint64_t id, const double* secondary);
IMPORT int32_t db_idx_long_double_store(uint64_t scope, capi_name table, capi_name payer, uint64_t id, const long double* secondary);


void migrator::inject(name table,name scope,name payer,name id, std::vector<char>  data) {            
    eosio::print("inject ", eosio::name(table), ":", eosio::name(scope), " <", eosio::name(id),":", eosio::name(payer) ,">\n");
    const auto resp = db_store_i64(
      scope.value,      // The scope where the record will be stored
      table.value,      // The ID/name of the table within the current scope/code context
      payer.value,      // The account that is paying for this storage
      id.value,         // Id of the entry
      (void*)&data[0],  // Record to store
      data.size()       // Size of data
    );
    eosio::print("inject resp: ", resp , "\n");
};

void migrator::idxi(name table,name scope,name payer,name id, uint64_t secondary) {            
    eosio::print("idxi ", eosio::name(table), ":", eosio::name(scope), " <", eosio::name(id),":", eosio::name(payer) ,">\n");
    const auto resp = db_idx64_store(
      scope.value,      // The scope for the secondary index
      table.value,      // The ID/name of the table within the current scope/code context
      payer.value,      // The account that is paying for this storage
      id.value,         // Id of the entry
      &secondary  // Record to store
    );
    eosio::print("idxi resp: ", resp , "\n");
};

void migrator::idxii(name table,name scope,name payer,name id, uint128_t secondary) {            
    eosio::print("idxii ", eosio::name(table), ":", eosio::name(scope), " <", eosio::name(id),":", eosio::name(payer) ,">\n");
    const auto resp = db_idx128_store(
      scope.value,      // The scope for the secondary index
      table.value,      // The ID/name of the table within the current scope/code context
      payer.value,      // The account that is paying for this storage
      id.value,         // Id of the entry
      &secondary      // Record to store
    );  
    eosio::print("idxii resp: ", resp , "\n");
};

void migrator::idxc(name table,name scope,name payer,name id, checksum256 secondary) {
    eosio::print("idxc ", eosio::name(table), ":", eosio::name(scope), " <", eosio::name(id),":", eosio::name(payer) ,">\n");
    const auto ref = secondary.get_array();
    const auto resp = db_idx256_store(
      scope.value,      // The scope for the secondary index
      table.value,      // The ID/name of the table within the current scope/code context
      payer.value,      // The account that is paying for this storage
      id.value,         // Id of the entry
      ref.data(),       // Record to store
      2
    ); 
    eosio::print("idxc resp: ", resp , "\n"); 
};

void migrator::idxdbl(name table,name scope,name payer,name id, double secondary) {
    eosio::print("idxdbl ", eosio::name(table), ":", eosio::name(scope), " <", eosio::name(id),":", eosio::name(payer) ,">\n");
    const auto resp = db_idx_double_store(
      scope.value,      // The scope for the secondary index
      table.value,      // The ID/name of the table within the current scope/code context
      payer.value,      // The account that is paying for this storage
      id.value,         // Id of the entry
      &secondary      // Record to store
    );
    eosio::print("idxdbl resp: ", resp , "\n"); 
};

void migrator::idxldbl(name table,name scope,name payer,name id, long double secondary) {
    eosio::print("idxldbl ", eosio::name(table), ":", eosio::name(scope), " <", eosio::name(id),":", eosio::name(payer) ,">\n");
    const auto resp = db_idx_long_double_store(
      scope.value,      // The scope for the secondary index
      table.value,      // The ID/name of the table within the current scope/code context
      payer.value,      // The account that is paying for this storage
      id.value,         // Id of the entry
      &secondary      // Record to store
    );
    eosio::print("idxldbl resp: ", resp , "\n");  
};

void migrator::eject(name account,name table,name scope,name id) {
  eosio::print("delete ", eosio::name(account), ":", eosio::name(table), ":", eosio::name(scope) , " <", eosio::name(id), ">\n");
	int32_t itr = eosio::internal_use_do_not_use::db_find_i64(account.value, scope.value, table.value, id.value);
	db_remove_i64(itr);
  eosio::print("idxldbl resp: itr=", itr , "\n");  
};
