import { Config } from "../models/config"

export const LOGO_PLACEHOLDER =
  "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/placeholder.png"

export const LOGO_LG_PLACEHOLDER =
  "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/placeholder-lg.png"

export interface TokenInfo {
  name: string
  logo?: string
  logo_lg?: string
  symbol: string
  account: string
  chain: "bos" | "eos" | "jungle" | "worbli" | "wax" | "snax" | "telos"
}

export function getTokenInfosForNetwork(network: string): TokenInfo[] {
  const chain = networkToName[network]
  if (!chain) {
    return []
  }

  return AIRDROPS.filter((element) => element.chain === chain)
}

export type TokenInfoKey = string

export function getTokenInfoKey(info: TokenInfo): TokenInfoKey {
  return info.account + info.symbol
}

export function getTokenInfosByKeyMap(): Record<TokenInfoKey, TokenInfo> {
  const mappings: ReturnType<typeof getTokenInfosByKeyMap> = {}
  getTokenInfosForNetwork(Config.current_network).forEach((info) => {
    mappings[getTokenInfoKey(info)] = info
  })

  return mappings
}

const networkToName: Record<string, TokenInfo["chain"]> = {
  "eos-mainnet": "eos",
  "eos-jungle": "jungle",
  "eos-worbli": "worbli",
  "wax-mainnet": "wax"
}

const eosCafeList: TokenInfo[] = [
  {
    name: "VIG",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/VIG.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/VIG-lg.png",
    symbol: "VIG",
    account: "vig111111111",
    chain: "eos"
  },
  {
    name: "UPD",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/upd-symbol-icon.png",
    logo_lg:
      "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/upd-symbol-icon.png",
    symbol: "UPD",
    account: "updtokenacct",
    chain: "eos"
  },
  {
    name: "AdderalCoin",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/placeholder.png",
    logo_lg:
      "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/placeholder-lg.png",
    symbol: "ADD",
    account: "eosadddddddd",
    chain: "eos"
  },
  {
    name: "CADEOS.io",
    logo:
      "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/ADE-logo-225x225.jpg",
    logo_lg:
      "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/ADE-logo-225x225.jpg",
    symbol: "ADE",
    account: "cadeositoken",
    chain: "eos"
  },
  {
    name: "EOSNOW",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/eosnow.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/eosnow-lg.png",
    symbol: "ANTE",
    account: "eosnowbpower",
    chain: "eos"
  },
  {
    name: "ANOX",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/anx-sm.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/anx-256.png",
    symbol: "ANX",
    account: "anoxanoxanox",
    chain: "eos"
  },
  {
    name: "Atidium",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/atidium.jpg",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/atidium-lg.png",
    symbol: "ATD",
    account: "eosatidiumio",
    chain: "eos"
  },
  {
    name: "ATMOS",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/atmos.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/atmos.png",
    symbol: "ATMOS",
    account: "novusphereio",
    chain: "eos"
  },
  {
    name: "Banker.Bet",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/BBC.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/BBC-lg.png",
    symbol: "BBC",
    account: "bbctokencode",
    chain: "eos"
  },
  {
    name: "BEAN",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/bean.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/bean-lg.png",
    symbol: "BEAN",
    account: "thebeantoken",
    chain: "eos"
  },
  {
    name: "EOS BET",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/eosbet.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/eosbet-lg.png",
    symbol: "BET",
    account: "betdividends",
    chain: "eos"
  },
  {
    name: "BetKing.io",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/betking.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/betking-lg.png",
    symbol: "BKT",
    account: "betkingtoken",
    chain: "eos"
  },
  {
    name: "eosBLACK",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/eosblack.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/eosblack-lg.png",
    symbol: "BLACK",
    account: "eosblackteam",
    chain: "eos"
  },
  {
    name: "BNT",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/bancor.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/bancor-lg.png",
    symbol: "BNT",
    account: "bntbntbntbnt",
    chain: "eos"
  },
  {
    name: "BNTEOS",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/bnteos.png",
    logo_lg:
      "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/placeholder-lg.png",
    symbol: "BNTEOS",
    account: "bnt2eosrelay",
    chain: "eos"
  },
  {
    name: "BNTUSDT",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/bntusdt.jpeg",
    logo_lg:
      "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/placeholder-lg.png",
    symbol: "BNTUSDT",
    account: "bancorr11232",
    chain: "eos"
  },
  {
    name: "BOID",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/boidlogo.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/BoidLogo-lg.png",
    symbol: "BOID",
    account: "boidcomtoken",
    chain: "eos"
  },
  {
    name: "BOS",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/bos.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/bos-lg.png",
    symbol: "BOS",
    account: "eosio.token",
    chain: "bos"
  },
  {
    name: "BOS",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/bos.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/bos-lg.png",
    symbol: "BOS",
    account: "bosibc.io",
    chain: "eos"
  },
  {
    name: "Bitcoin",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/eosbetbtc.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/eosbetbtc-lg.png",
    symbol: "BTC",
    account: "eosbettokens",
    chain: "eos"
  },
  {
    name: "Bitcoin Cash",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/eosbetbch.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/eosbetbch-lg.png",
    symbol: "BCH",
    account: "eosbettokens",
    chain: "eos"
  },
  {
    name: "GrandpaBTC",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/grandpa-btc.png",
    logo_lg:
      "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/grandpa-btc-lg.png",
    symbol: "BTC",
    account: "grandpacoins",
    chain: "eos"
  },
  {
    name: "The EOS Button",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/placeholder.png",
    logo_lg:
      "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/placeholder-lg.png",
    symbol: "BTN",
    account: "eosbuttonbtn",
    chain: "eos"
  },
  {
    name: "CARMEL",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/carmel.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/carmel-lg.png",
    symbol: "CARMEL",
    account: "carmeltokens",
    chain: "eos"
  },
  {
    name: "Chaince",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/chaince.jpg",
    logo_lg:
      "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/placeholder-lg.png",
    symbol: "CET",
    account: "eosiochaince",
    chain: "eos"
  },
  {
    name: "Chintai",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/chintai-chex.png",
    logo_lg:
      "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/chintai-chex-lg.png",
    symbol: "CHEX",
    account: "chexchexchex",
    chain: "eos"
  },
  {
    name: "Challenge DAC",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/CHLnewPNG500.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/CHLnewPNG500.png",
    symbol: "CHL",
    account: "challengedac",
    chain: "eos"
  },
  {
    name: "Carbon",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/carbonlogo-64.png",
    logo_lg:
      "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/carbonlogo-128.png",
    symbol: "CUSD",
    account: "stablecarbon",
    chain: "eos"
  },
  {
    name: "DABBLE",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/dabble.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/dabble-lg.png",
    symbol: "DAB",
    account: "eoscafekorea",
    chain: "eos"
  },
  {
    name: "DAPP Network",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/dapp.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/dapp-lg.png",
    symbol: "DAPP",
    account: "dappservices",
    chain: "eos"
  },
  {
    name: "DEOS Games",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/deosgames.png",
    logo_lg:
      "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/placeholder-lg.png",
    symbol: "DEOS",
    account: "thedeosgames",
    chain: "eos"
  },
  {
    name: "DICE",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/dice.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/dice-lg.png",
    symbol: "DICE",
    account: "betdicetoken",
    chain: "eos"
  },
  {
    name: "Dig Coin",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/dig.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/dig-lg.png",
    symbol: "DIG",
    account: "digcoinsmine",
    chain: "eos"
  },
  {
    name: "Dragon Option",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/dragon.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/dragon-lg.png",
    symbol: "DRAGON",
    account: "eosdragontkn",
    chain: "eos"
  },
  {
    name: "DS",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/DS.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/DS.png",
    symbol: "DS",
    account: "dsdsdsdsdsds",
    chain: "eos"
  },
  {
    name: "GrandpaDOGE",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/grandpa-doge.png",
    logo_lg:
      "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/grandpa-doge-lg.png",
    symbol: "DOGE",
    account: "grandpacoins",
    chain: "eos"
  },
  {
    name: "EOS AUCTION PLATFORM",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/eap.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/eap-lg.png",
    symbol: "EAP",
    account: "eosauctionpt",
    chain: "eos"
  },
  {
    name: "EBTC",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/placeholder.png",
    logo_lg:
      "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/placeholder-lg.png",
    symbol: "EBTC",
    account: "bitpietokens",
    chain: "eos"
  },
  {
    name: "eosCASH",
    logo: "",
    logo_lg: "",
    symbol: "ECASH",
    account: "horustokenio",
    chain: "eos"
  },
  {
    name: "EDNA",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/edna.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/edna-lg.png",
    symbol: "EDNA",
    account: "ednazztokens",
    chain: "eos"
  },
  {
    name: "EETH",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/placeholder.png",
    logo_lg:
      "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/placeholder-lg.png",
    symbol: "EETH",
    account: "ethsidechain",
    chain: "eos"
  },
  {
    name: "EETH",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/placeholder.png",
    logo_lg:
      "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/placeholder-lg.png",
    symbol: "EETH",
    account: "bitpietokens",
    chain: "eos"
  },
  {
    name: "Effect.AI",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/efx.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/efx-lg.png",
    symbol: "EFX",
    account: "effecttokens",
    chain: "eos"
  },
  {
    name: "Emanate MNX",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/emanate-mnx.png",
    logo_lg:
      "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/emanate-mnx-lg.png",
    symbol: "MNX",
    account: "emanatenekot",
    chain: "eos"
  },
  {
    name: "Emanate",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/emanate.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/emanate-lg.png",
    symbol: "EMT",
    account: "emanateoneos",
    chain: "eos"
  },
  {
    name: "eosDAC",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/eosdac.jpg",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/eosdac-lg.jpg",
    symbol: "EOSDAC",
    account: "eosdactokens",
    chain: "eos"
  },
  {
    name: "EOSN",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/eosn.jpg",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/eosn-lg.jpg",
    symbol: "EOSN",
    account: "eosnationinc",
    chain: "eos"
  },
  {
    name: "EOX Commerce",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/eoxcommerce.png",
    logo_lg:
      "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/eoxcommerce-lg.png",
    symbol: "EOX",
    account: "eoxeoxeoxeox",
    chain: "eos"
  },
  {
    name: "ERO",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/ero.png",
    logo_lg:
      "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/placeholder-lg.png",
    symbol: "ERO",
    account: "eoslandadmin",
    chain: "eos"
  },
  {
    name: "EOSLAND RARE ORE",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/ero.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/ero-lg.png",
    symbol: "ERO",
    account: "eoslandadmin",
    chain: "eos"
  },
  {
    name: "Ethereum",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/eosbeteth.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/eosbeteth-lg.png",
    symbol: "ETH",
    account: "eosbettokens",
    chain: "eos"
  },
  {
    name: "GrandpaETH",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/grandpa-eth.png",
    logo_lg:
      "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/grandpa-eth-lg.png",
    symbol: "ETH",
    account: "grandpacoins",
    chain: "eos"
  },
  {
    name: "Europechain",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/europechain.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/europechain.png",
    symbol: "XEC",
    account: "europe.chain",
    chain: "eos"
  },
  {
    name: "EUSD",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/placeholder.png",
    logo_lg:
      "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/placeholder-lg.png",
    symbol: "EUSD",
    account: "bitpietokens",
    chain: "eos"
  },
  {
    name: "FairEOS",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/fair.jpg",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/fair-lg.jpg",
    symbol: "FAIR",
    account: "faireostoken",
    chain: "eos"
  },
  {
    name: "FastWin",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/fast.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/fast-lg.png",
    symbol: "FAST",
    account: "fastecoadmin",
    chain: "eos"
  },
  {
    name: "UXfyre",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/uxfyre.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/uxfyre-lg.png",
    symbol: "FYRE",
    account: "uxfyretoken1",
    chain: "eos"
  },
  {
    name: "Horus Pay",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/horuspay.jpg",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/horuspay-lg.jpg",
    symbol: "HORUS",
    account: "horustokenio",
    chain: "eos"
  },
  {
    name: "HASHFUN",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/HFC.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/HFC.png",
    symbol: "HFC",
    account: "hashfuncoins",
    chain: "eos"
  },
  {
    name: "HireVibes",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/hvt.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/hvt-lg.png",
    symbol: "HVT",
    account: "hirevibeshvt",
    chain: "eos"
  },
  {
    name: "Infinicoin",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/infiniverse.png",
    logo_lg:
      "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/infiniverse-lg.png",
    symbol: "INF",
    account: "infinicoinio",
    chain: "eos"
  },
  {
    name: "IPOS",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/placeholder.png",
    logo_lg:
      "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/placeholder-lg.png",
    symbol: "IPOS",
    account: "oo1122334455",
    chain: "eos"
  },
  {
    name: "Everipedia",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/everipedia.jpg",
    logo_lg:
      "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/everipedia-lg.png",
    symbol: "IQ",
    account: "everipediaiq",
    chain: "eos"
  },
  {
    name: "EOSJacks",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/eosjacks.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/eosjacks-lg.png",
    symbol: "JKR",
    account: "eosjackscoin",
    chain: "eos"
  },
  {
    name: "KARMA",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/karma.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/karma-lg.png",
    symbol: "KARMA",
    account: "therealkarma",
    chain: "eos"
  },
  {
    name: "KEOS",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/placeholder.png",
    logo_lg:
      "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/placeholder-lg.png",
    symbol: "KEOS",
    account: "keoskorea111",
    chain: "eos"
  },
  {
    name: "KROWN",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/KROWN.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/KROWN-lg.png",
    symbol: "KROWN",
    account: "krowndactokn",
    chain: "eos"
  },
  {
    name: "LED",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/led.png",
    logo_lg:
      "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/placeholder-lg.png",
    symbol: "LED",
    account: "okkkkkkkkkkk",
    chain: "eos"
  },
  {
    name: "LICC",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/licc.png",
    logo_lg:
      "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/placeholder-lg.png",
    symbol: "LICC",
    account: "liccommunity",
    chain: "eos"
  },
  {
    name: "LuckyGo",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/lkg.jpg",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/lkg-lg.jpg",
    symbol: "LKG",
    account: "luckygotoken",
    chain: "eos"
  },
  {
    name: "Lelego",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/lelego.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/lelego-lg.png",
    symbol: "LLG",
    account: "llgonebtotal",
    chain: "eos"
  },
  {
    name: "Litecoin",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/eosbetltc.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/eosbetltc-lg.png",
    symbol: "LTC",
    account: "eosbettokens",
    chain: "eos"
  },
  {
    name: "LUCKY",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/placeholder.png",
    logo_lg:
      "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/placeholder-lg.png",
    symbol: "LUCKY",
    account: "eoslucktoken",
    chain: "eos"
  },
  {
    name: "Lumeos",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/lumeos.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/lumeos-lg.png",
    symbol: "LUME",
    account: "lumetokenctr",
    chain: "eos"
  },
  {
    name: "LYNX",
    logo:
      "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/worktokenbviLYNX.png",
    logo_lg:
      "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/worktokenbviLYNX-lg.png",
    symbol: "LYNX",
    account: "worktokenbvi",
    chain: "eos"
  },
  {
    name: "dmail",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/dmail.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/dmail-lg.png",
    symbol: "MAIL",
    account: "d.mail",
    chain: "eos"
  },
  {
    name: "MAX",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/placeholder.png",
    logo_lg:
      "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/placeholder-lg.png",
    symbol: "MAX",
    account: "eosmax1token",
    chain: "eos"
  },
  {
    name: "MEET.ONE",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/meetone.jpg",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/meetone-lg.jpg",
    symbol: "MEETONE",
    account: "eosiomeetone",
    chain: "eos"
  },
  {
    name: "Royal Online Vegas",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/mev.jpg",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/mev-lg.jpg",
    symbol: "MEV",
    account: "eosvegascoin",
    chain: "eos"
  },
  {
    name: "MORTYS",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/mortys.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/mortys-lg.png",
    symbol: "MORTYS",
    account: "mrpoopybutt1",
    chain: "eos"
  },
  {
    name: "Nebula",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/nebula.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/nebula-lg.png",
    symbol: "NEB",
    account: "nebulatokenn",
    chain: "eos"
  },
  {
    name: "Effect Network",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/nfx.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/nfx-lg.png",
    symbol: "NFX",
    account: "effecttokens",
    chain: "eos"
  },
  {
    name: "NUTS",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/nuts.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/nuts-lg.png",
    symbol: "NUTS",
    account: "nutscontract",
    chain: "eos"
  },
  {
    name: "Oracle Chain",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/oraclechain.png",
    logo_lg:
      "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/oraclechain-lg.png",
    symbol: "OCT",
    account: "octtothemoon",
    chain: "eos"
  },
  {
    name: "OnePlay",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/oneplay.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/oneplay-lg.png",
    symbol: "ONE",
    account: "oneplaytoken",
    chain: "eos"
  },
  {
    name: "PEOS",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/peos.png",
    logo_lg:
      "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/placeholder-lg.png",
    symbol: "PEOS",
    account: "thepeostoken",
    chain: "eos"
  },
  {
    name: "pixEOS",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/pixeos.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/pixeos-lg.png",
    symbol: "PIXEOS",
    account: "pixeos1token",
    chain: "eos"
  },
  {
    name: "PIZZA",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/PIZZA.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/PIZZA-lg.png",
    symbol: "PIZZA",
    account: "pizzatotoken",
    chain: "eos"
  },
  {
    name: "EOS Poker",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/poker.jpg",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/poker-lg.jpg",
    symbol: "POKER",
    account: "eospokercoin",
    chain: "eos"
  },
  {
    name: "Poorman Token",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/poorman.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/poorman-lg.png",
    symbol: "POOR",
    account: "poormantoken",
    chain: "eos"
  },
  {
    name: "Crypto Peso",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/cryptopeso.png",
    logo_lg:
      "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/cryptopeso-lg.png",
    symbol: "PSO",
    account: "cryptopesosc",
    chain: "eos"
  },
  {
    name: "PUBLYTO",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/pub.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/pub-lg.png",
    symbol: "PUB",
    account: "publytoken11",
    chain: "eos"
  },
  {
    name: "CryptoPIX PXS",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/pixels.jpeg",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/pixels_lg.jpeg",
    symbol: "PXS",
    account: "pxstokensapp",
    chain: "eos"
  },
  {
    name: "RAMtoken",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/ramtoken.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/ramtoken-lg.png",
    symbol: "RAM",
    account: "ramtokenmoon",
    chain: "eos"
  },
  {
    name: "RIDL",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/ridl.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/ridl-lg.png",
    symbol: "RIDL",
    account: "ridlridlcoin",
    chain: "eos"
  },
  {
    name: "Rocket Battles",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/rocketbattle.png",
    logo_lg:
      "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/rocketbattle-lg.png",
    symbol: "ROCKET",
    account: "rocketbattle",
    chain: "eos"
  },
  {
    name: "ROJI",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/roji.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/roji-lg.png",
    symbol: "ROJI",
    account: "rojirojiroji",
    chain: "eos"
  },
  {
    name: "Real World Coupon",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/rwc.png",
    logo_lg:
      "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/placeholder-lg.png",
    symbol: "RWC",
    account: "realworldcpn",
    chain: "eos"
  },
  {
    name: "Parsl Seed",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/seed.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/seed-lg.png",
    symbol: "SEED",
    account: "parslseed123",
    chain: "eos"
  },
  {
    name: "Sense",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/sense.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/sense-400.png",
    symbol: "SENSE",
    account: "sensegenesis",
    chain: "eos"
  },
  {
    name: "Sprtshubcoin",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/sprtshubcoin.png",
    logo_lg:
      "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/sprtshubcoin-lg.png",
    symbol: "SHC",
    account: "sprtshubcoin",
    chain: "eos"
  },
  {
    name: "SOLIT",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/solit.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/solit-lg.png",
    symbol: "SLT",
    account: "nblabtokenss",
    chain: "eos"
  },
  {
    name: "Nebula Stable",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/nebula.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/nebula-lg.png",
    symbol: "SNEB",
    account: "nebulatokenn",
    chain: "eos"
  },
  {
    name: "SNAX",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/SNAX.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/SNAX.png",
    symbol: "SNAX",
    account: "snax.token",
    chain: "snax"
  },
  {
    name: "SOV",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/sov.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/sov-lg.png",
    symbol: "SOV",
    account: "sovmintofeos",
    chain: "eos"
  },
  {
    name: "eoseven",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/placeholder.png",
    logo_lg:
      "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/placeholder-lg.png",
    symbol: "SVN",
    account: "eoseventoken",
    chain: "eos"
  },
  {
    name: "Telos",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/telos.png",
    logo_lg:
      "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/placeholder-lg.png",
    symbol: "TLOS",
    account: "eosio.token",
    chain: "telos"
  },
  {
    name: "TOOKTOOK",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/took.jpg",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/took-lg.jpg",
    symbol: "TOOK",
    account: "taketooktook",
    chain: "eos"
  },
  {
    name: "TokenPocket",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/placeholder.png",
    logo_lg:
      "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/placeholder-lg.png",
    symbol: "TPT",
    account: "eosiotptoken",
    chain: "eos"
  },
  {
    name: "TokenPocket",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/placeholder.png",
    logo_lg:
      "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/placeholder-lg.png",
    symbol: "TPT",
    account: "bosibc.io",
    chain: "bos"
  },
  {
    name: "TRIV",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/triv-token-logo.png",
    logo_lg:
      "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/triv-token-logo.png",
    symbol: "TRIV",
    account: "triviatokens",
    chain: "eos"
  },
  {
    name: "TRYBE",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/trybe.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/trybe-lg.png",
    symbol: "TRYBE",
    account: "trybenetwork",
    chain: "eos"
  },
  {
    name: "USDE",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/USDE.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/USDE-lg.png",
    symbol: "USDE",
    account: "usdetotokens",
    chain: "eos"
  },
  {
    name: "USDT",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/USDT.png",
    logo_lg:
      "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/placeholder-lg.png",
    symbol: "USDT",
    account: "tethertether",
    chain: "eos"
  },
  {
    name: "USDT",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/USDT.png",
    logo_lg:
      "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/placeholder-lg.png",
    symbol: "USDT",
    account: "bosibc.io",
    chain: "bos"
  },
  {
    name: "WhaleEx",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/whaleex.png",
    logo_lg:
      "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/placeholder-lg.png",
    symbol: "WAL",
    account: "whaleextoken",
    chain: "eos"
  },
  {
    name: "WECASH",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/wecash.jpg",
    logo_lg:
      "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/placeholder-lg.png",
    symbol: "WECASH",
    account: "weosservices",
    chain: "eos"
  },
  {
    name: "WiZZ",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/wizz.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/wizz-lg.png",
    symbol: "WIZZ",
    account: "wizznetwork1",
    chain: "eos"
  },
  {
    name: "Worbli",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/worbli.png",
    logo_lg:
      "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/placeholder-lg.png",
    symbol: "WBI",
    account: "eosio.token",
    chain: "worbli"
  },
  {
    name: "Gamblr",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/gamblr.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/gamblr.png",
    symbol: "GAMBLR",
    account: "gamblrtokens",
    chain: "eos"
  },
  {
    name: "WRK",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/worktokenbviWRK.png",
    logo_lg:
      "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/worktokenbviWRK-lg.png",
    symbol: "WRK",
    account: "worktokenbvi",
    chain: "eos"
  },
  {
    name: "Billionaire Token",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/billionaire.png",
    logo_lg:
      "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/billionaire-lg.png",
    symbol: "XBL",
    account: "billionairet",
    chain: "eos"
  },
  {
    name: "ZKS",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/zks.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/zks-lg.png",
    symbol: "ZKS",
    account: "zkstokensr4u",
    chain: "eos"
  },
  {
    name: "Qubicles",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/qbe.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/qbe-lg.png",
    symbol: "QBE",
    account: "qubicletoken",
    chain: "telos"
  },
  {
    name: "Beatitude",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/beatitude.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/beatitude-lg.png",
    symbol: "HEART",
    account: "revelation21",
    chain: "telos"
  },
  {
    name: "Cards & Tokens",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/cnt.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/cnt.png",
    symbol: "CNT",
    account: "vapaeetokens",
    chain: "telos"
  },
  {
    name: "Viitasphere Token",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/viitasphere.png",
    logo_lg:
      "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/viitasphere-lg.png",
    symbol: "VIITA",
    account: "viitasphere1",
    chain: "telos"
  },
  {
    name: "VIITA Certificate Token",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/viitasphere.png",
    logo_lg:
      "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/viitasphere-lg.png",
    symbol: "VIICT",
    account: "viitasphere1",
    chain: "telos"
  },
  {
    name: "Acorn UBI",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/acorn.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/acorn-lg.png",
    symbol: "ACORN",
    account: "acornaccount",
    chain: "telos"
  },
  {
    name: "EDNA",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/edna.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/edna-lg.png",
    symbol: "EDNA",
    account: "ednazztokens",
    chain: "telos"
  },
  {
    name: "Teachology",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/teach.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/teach-lg.png",
    symbol: "TEACH",
    account: "teachology14",
    chain: "telos"
  },
  {
    name: "Proxibots",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/proxibots.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/proxibots-lg.png",
    symbol: "ROBO",
    account: "proxibotstkn",
    chain: "telos"
  },
  {
    name: "TelosDAC",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/telosdac.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/telosdac-lg.png",
    symbol: "TLOSDAC",
    account: "telosdacdrop",
    chain: "telos"
  },
  {
    name: "Anudit Coin Test",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/anudit-test.png",
    logo_lg:
      "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/anudit-test-lg.png",
    symbol: "ANT",
    account: "antestacc111",
    chain: "jungle"
  },
  {
    name: "MyCryptoVegas Token",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/cts.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/cts-lg.png",
    symbol: "CTS",
    account: "cryptovgscts",
    chain: "eos"
  },
  {
    name: "WORD",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/word.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/word.png",
    symbol: "WORD",
    account: "wordtokeneos",
    chain: "eos"
  },
  {
    name: "WORD",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/word.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/word.png",
    symbol: "WORD",
    account: "wordtokeneos",
    chain: "telos"
  },
  {
    name: "WORD",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/word.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/word.png",
    symbol: "WORD",
    account: "wordtokeneos",
    chain: "jungle"
  },
  {
    name: "Yakee chain",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/YKC.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/YKC.png",
    symbol: "YKC",
    account: "okkkkkkkkkkk",
    chain: "eos"
  },
  {
    name: "NUT",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/nut_225x225.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/nut_424x424.png",
    symbol: "NUT",
    account: "eosdtnutoken",
    chain: "eos"
  },
  {
    name: "San Diego City Token",
    logo:
      "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/city-seal-Blue-and-Gold-small-300x300.png",
    logo_lg:
      "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/city-seal-Blue-and-Gold-small-300x300.png",
    symbol: "SAND",
    account: "sandiegocoin",
    chain: "eos"
  },
  {
    name: "POW",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/pow.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/pow-lg.png",
    symbol: "POW",
    account: "powhcontract",
    chain: "eos"
  },
  {
    name: "POWX",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/powx.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/powx-lg.png",
    symbol: "POWX",
    account: "powxtokenpow",
    chain: "eos"
  },
  {
    name: "GoldenChip",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/goldenchip.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/goldenchip.png",
    symbol: "GCHIP",
    account: "goldenchipio",
    chain: "eos"
  },
  {
    name: "PINK",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/pink.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/pink-lg.png",
    symbol: "PINK",
    account: "pinknettoken",
    chain: "wax"
  },
  {
    name: "WAX",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/wax.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/wax.png",
    symbol: "WAX",
    account: "eosio.token",
    chain: "wax"
  },
  {
    name: "One Thousand Coin",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/OTCeoslogo1.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/OTCeoslogo1.png",
    symbol: "OTC",
    account: "thousandcoin",
    chain: "eos"
  },
  {
    name: "STEEMP on EOS",
    logo:
      "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/EOSSTEEMpFULLres-527x504.png",
    logo_lg:
      "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/EOSSTEEMpFULLres-527x504.png",
    symbol: "STEEMP",
    account: "steemoneosio",
    chain: "eos"
  },
  {
    name: "SQRL",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/sqrl.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/sqrl-lg.png",
    symbol: "SQRL",
    account: "sqrlwalletio",
    chain: "telos"
  }
]

const extraList: TokenInfo[] = [
  {
    name: "EDNA",
    logo: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/edna.png",
    logo_lg: "https://raw.githubusercontent.com/eoscafe/eos-airdrops/master/logos/edna-lg.png",
    symbol: "EDNA",
    account: "ednazztokens",
    chain: "worbli"
  },
  {
    name: "PLO",
    symbol: "PLO",
    account: "playeronetkn",
    chain: "eos"
  },
  {
    name: "BRM",
    symbol: "BRM",
    account: "openbrmeos11",
    chain: "eos"
  },
  {
    name: "DAPPHDL",
    symbol: "DAPPHDL",
    account: "dappairhodl1",
    chain: "eos"
  },
  {
    name: "EFOR",
    symbol: "EFOR",
    account: "theforcegrou",
    chain: "eos"
  },
  {
    name: "BITI",
    symbol: "BITI",
    account: "biteyebiteye",
    chain: "eos"
  },
  {
    name: "RWC",
    symbol: "RWC",
    account: "realworldcpn",
    chain: "eos"
  },
  {
    name: "SOV",
    symbol: "SOV",
    account: "sovmintofeos",
    chain: "eos"
  },
  {
    name: "ESB",
    symbol: "ESB",
    account: "esbcointoken",
    chain: "eos"
  }
]

// List of tokens that do not actual work correctly anymore
const removedTokens = [
  { chain: "eos", account: "nutscontract" },
  { chain: "eos", account: "uxfyretoken1" },
  { chain: "eos", account: "triviatokens" }
]

function isRemovedToken(tokenInfo: TokenInfo): boolean {
  return removedTokens.some(
    (removedToken) =>
      tokenInfo.chain === removedToken.chain && tokenInfo.account === removedToken.account
  )
}

export const AIRDROPS: TokenInfo[] = [...eosCafeList, ...extraList]
  .filter((element) => !isRemovedToken(element))
  .map((element) => {
    if (element.logo === LOGO_PLACEHOLDER) {
      element.logo = undefined
    }

    if (element.logo_lg === LOGO_LG_PLACEHOLDER) {
      element.logo_lg = undefined
    }

    return element
  })
