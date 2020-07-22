const windowTS = window as any

// Extracted from React register service worker part to detect localhost
const isLocalhost = Boolean(
  window.location.hostname === "localhost" ||
    // [::1] is the IPv6 localhost address.
    window.location.hostname === "[::1]" ||
    // 127.0.0.1/8 is considered localhost for IPv4.
    window.location.hostname.match(/^127(?:\.(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)){3}$/)
)

const isEnvSet = (value: string | undefined): boolean => value != null && value !== ""

const newDefaultConfig = () => {
  const core = {
    current_network: process.env.REACT_APP_EOSQ_CURRENT_NETWORK || "local",
    dfuse_auth_endpoint: process.env.REACT_APP_DFUSE_AUTH_URL || "null://",
    dfuse_io_api_key: process.env.REACT_APP_DFUSE_API_KEY || "web_1234567890abc",
    dfuse_io_endpoint: process.env.REACT_APP_DFUSE_API_NETWORK || "localhost:8080",
    display_price: false,
    price_ticker_name: "EOS",
    version: 1,
    available_networks: [
      {
        id: "local",
        is_test: true,
        logo: "/images/eos-mainnet.png",
        name: "Local Network",
        url: "http://localhost:8080"
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

  if (isEnvSet(process.env.REACT_APP_EOSQ_DISPLAY_PRICE)) {
    core.display_price = process.env.REACT_APP_EOSQ_DISPLAY_PRICE === "true"
  }

  if (isEnvSet(process.env.REACT_APP_EOSQ_AVAILABLE_NETWORKS)) {
    try {
      core.available_networks = JSON.parse(process.env.REACT_APP_EOSQ_AVAILABLE_NETWORKS!)
    } catch (error) {
      console.error("Invalid available networks environemnt variable, it's not valid JSON", error)
    }
  }

  return core
}

if (!windowTS.TopLevelConfig) {
  windowTS.TopLevelConfig = newDefaultConfig()
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

  dfuse_io_endpoint: string
  dfuse_io_api_key: string
  dfuse_auth_endpoint: string

  current_network: string
  available_networks: EosqNetwork[]
  display_price: boolean
  price_ticker_name: string

  secure: boolean
  disable_segments: boolean
  disable_sentry: boolean
}

export const Config = { ...windowTS.TopLevelConfig, isLocalhost } as EosqConfig
