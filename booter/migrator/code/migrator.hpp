#include <string>
#include <vector>
#include <eosio/eosio.hpp>

using namespace eosio;
using std::string;  

class [[eosio::contract]]  migrator : public contract {
  public:
    migrator(name receiver, name code, datastream<const char*> ds)
      :contract(receiver, code, ds)
      {}
  
    // Actions      
    [[eosio::action]]
    void inject(name table,name scope,name payer,name id, std::vector<char>  data);
  private:
};