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



   struct resale_share {
      name       receiver;      // the receiver of the resale share
      uint16_t   basis_point;   // 1 means 0.0001

      EOSLIB_SERIALIZE( resale_share, (receiver)(basis_point) )
   };
   typedef vector<resale_share> resale_share_vector;


   struct [[eosio::table("factory.a"), eosio::contract("migrator")]] token_factory_v0 {
      uint64_t                      id;
      name                          asset_manager;
      name                          asset_creator;
      name                          conversion_rate_oracle_contract;
      vector<asset>                 chosen_rate;            // "10.0000 SECONDS", "20.6400 SECONDS", "4.0000 MINUTES", "5.1500 DAYS"
      asset                         minimum_resell_price;   // e.g. "2.00000000 UOS", "2.00000000 USD", etc
      vector<resale_share>          resale_shares;
      /**
       * if a window_start/window_end is missing, it means no limit; if window_end = minimum value, means it can never happen
       * mint window and trading window can only be absolute time values
       */
      optional<uint32_t>            mintable_window_start;
      optional<uint32_t>            mintable_window_end;
      optional<uint32_t>            trading_window_start;
      optional<uint32_t>            trading_window_end;

      /**
       * if a window_start and window_end is missing then the tokens are never recallable
       * recall window can only be relative to the mint date
       */
      optional<uint32_t>            recall_window_start;
      optional<uint32_t>            recall_window_end;

      optional<uint32_t>            lockup_time;               // lockup time since mint date, token cannot be transferred before lockup
      vector<name>                  conditionless_receivers;   // an NFT can be transfered to a conditionless receiver bypassing any restriction, like trading window, minimum resell price, etc
      uint8_t                       stat;                      // active by default after creation
      vector<string>                meta_uris;                 // can only append new URIs
      checksum256                   meta_hash;

      optional<uint32_t>            max_mintable_tokens;
      uint32_t                      minted_tokens_no;
      uint32_t                      existing_tokens_no;
      binary_extension<optional<uint32_t>> authorized_tokens_no;       // The current quantity of tokens that authorized minters can issue
      binary_extension<optional<uint32_t>> account_minting_limit; // number of minting limit per account

      uint64_t primary_key()const { return id; }

      EOSLIB_SERIALIZE( token_factory_v0, (id)(asset_manager)(asset_creator)
                  (conversion_rate_oracle_contract)(chosen_rate)(minimum_resell_price)(resale_shares)
                  (mintable_window_start)(mintable_window_end)(trading_window_start)(trading_window_end)
                  (recall_window_start)(recall_window_end)(lockup_time)(conditionless_receivers)(stat)(meta_uris)(meta_hash)
                  (max_mintable_tokens)(minted_tokens_no)(existing_tokens_no)(authorized_tokens_no)(account_minting_limit) )
   };

   typedef eosio::multi_index< "factory.a"_n, token_factory_v0 > token_factory_table;

   struct [[eosio::table("token.a"), eosio::contract("migrator")]] token_v0 {
      uint64_t                id;
      uint64_t                token_factory_id;
      time_point_sec          mint_date;     // the mint date relative to epoch
      uint32_t                serial_number; // the number which was assigned to this token when it was issues from token_factory_id

      uint64_t primary_key()const { return id; }

      EOSLIB_SERIALIZE( token_v0, (id)(token_factory_id)(mint_date)(serial_number) )
   };
   typedef eosio::multi_index< "token.a"_n, token_v0 > token_table;
};

