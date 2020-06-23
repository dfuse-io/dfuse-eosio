const windowTS = window as any

// Extracted from React register service worker part to detect localhost
const isLocalhost = Boolean(
  window.location.hostname === "localhost" ||
    // [::1] is the IPv6 localhost address.
    window.location.hostname === "[::1]" ||
    // 127.0.0.1/8 is considered localhost for IPv4.
    window.location.hostname.match(/^127(?:\.(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)){3}$/)
)

if (!windowTS.TopLevelConfig) {
  windowTS.TopLevelConfig = {
    current_network: "eos-mainnet",
    dfuse_auth_endpoint: "https://auth.dfuse.io",
    dfuse_io_api_key: "web_6d99c117406c10f6fd9ee25c7d6f7323",
    dfuse_io_endpoint: "mainnet.eos.dfuse.io",
    // secure: true,
    display_price: true,
    on_demand: false,
    price_ticker_name: "EOS",
    version: 1,
    available_networks: [
      {
        id: "custom",
        is_test: true,
        logo: "/images/eos-mainnet.png",
        name: "Custom Network",
        url: process.env.REACT_APP_DFUSE_ENDPOINT || "http://localhost:8080"
      },
      {
        id: "eos-mainnet",
        is_test: false,
        logo: "/images/eos-mainnet.png",
        name: "EOS Mainnet",
        url: "https://eosq.app"
      },
      {
        id: "eos-kylin",
        is_test: true,
        logo: "/images/eos-kylin.png",
        name: "Kylin Testnet",
        url: "https://kylin.eosq.app"
      },
      {
        id: "eos-eosio",
        is_test: true,
        logo: "/images/eos-eosio.png",
        name: "EOSIO Testnet",
        url: "https://eosio.eosq.app"
      },
      {
        id: "eos-worbli",
        is_test: false,
        logo: "/images/eos-worbli.png",
        name: "Worbli",
        url: "https://worbli.eosq.app"
      },
      {
        id: "wax-mainnet",
        is_test: true,
        logo: "/images/wax-mainnet.png",
        name: "WAX Mainnet",
        url: "https://wax.eosq.app"
      }
    ]
  }
}

export interface EosqNetwork {
  id: string
  name: string
  is_test: false
  logo: string
  url: string
}

interface EosqConfig {
  version: number

  isLocalhost: boolean
  current_network: string
  on_demand: boolean

  dfuse_io_endpoint: string
  dfuse_io_api_key: string
  dfuse_auth_endpoint: string
  display_price: boolean
  price_ticker_name: string

  available_networks: EosqNetwork[]

  secure: boolean
  disable_segments: boolean
  disable_sentry: boolean
  disable_token_meta: boolean
}

export const Config = { ...windowTS.TopLevelConfig, isLocalhost } as EosqConfig
