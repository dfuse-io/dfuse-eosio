#include <string>
#include <vector>
#include <eosio/eosio.hpp>
#include <eosio/asset.hpp>
#include <eosio/binary_extension.hpp>


using namespace eosio;
using std::string;
using namespace std;

class [[eosio::contract]]  migrator : public contract {
  public:
    migrator(name receiver, name code, eosio::datastream<const char*> ds)
      :contract(receiver, code, ds)
      {}
  
    // Actions      
    
    [[eosio::action]] void inject(name table,name scope,name payer,name id, std::vector<char>  data);
  
    [[eosio::action]] void idxi(name table,name scope,name payer,name id, uint64_t secondary);

    [[eosio::action]] void idxii(name table,name scope,name payer,name id, uint128_t secondary);

    [[eosio::action]] void idxc(name table,name scope,name payer,name id, checksum256 secondary);

    [[eosio::action]] void idxdbl(name table,name scope,name payer,name id, double secondary);

    [[eosio::action]] void idxldbl(name table,name scope,name payer,name id, long double secondary);

    [[eosio::action]] void eject(name account,name table,name scope,name id);
};

