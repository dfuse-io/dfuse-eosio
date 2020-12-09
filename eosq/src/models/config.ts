import { debugLog } from "../services/logger"

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
    version: 1,
    network_id:
      process.env.REACT_APP_EOSQ_NETWORK_ID ||
      process.env.REACT_APP_EOSQ_CURRENT_NETWORK ||
      "local",
    chain_core_symbol: "4,EOS",
    dfuse_auth_endpoint: process.env.REACT_APP_DFUSE_AUTH_URL || "null://",
    dfuse_io_api_key: process.env.REACT_APP_DFUSE_API_KEY || "web_1234567890abc",
    dfuse_io_endpoint: process.env.REACT_APP_DFUSE_API_NETWORK || "localhost:8080",
    secure: process.env.REACT_APP_DFUSE_API_NETWORK_SECURE === "true",
    display_price: false,

    available_networks: [
      {
        id: "local",
        is_test: true,
        name: "Local Network",
        url: "http://localhost:8080",
      },
      {
        id: "eos-kylin",
        is_test: true,
        name: "Kylin Testnet",
        url: "https://kylin.eosq.app",
      },
      {
        id: "wax-mainnet",
        is_test: false,
        name: "WAX Mainnet",
        url: "https://wax.eosq.app",
      },
    ],
  }

  if (isEnvSet(process.env.REACT_APP_EOSQ_CHAIN_CORE_SYMBOL)) {
    core.chain_core_symbol = process.env.REACT_APP_EOSQ_CHAIN_CORE_SYMBOL!
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
} else {
  // Config loaded from the server, to avoid having to refactor server config, we simply make a migration pass. If the
  // new `network_id` is not set but the old `current_network` variable exist and its a string type, use it.
  if (
    !typeof windowTS.TopLevelConfig.network_id &&
    typeof windowTS.TopLevelConfig.current_network === "string"
  ) {
    windowTS.TopLevelConfig.network_id = windowTS.TopLevelConfig.current_network
    delete windowTS.TopLevelConfig.current_network
  }
}

export interface EosqNetwork {
  id: string
  name: string
  url: string
  is_test?: boolean
  logo?: string
  logo_text?: string
  pageTitle?: string
  faviconTemplate?: string
}

interface EosqConfig {
  version: number
  isLocalhost: boolean

  dfuse_io_endpoint: string
  dfuse_io_api_key: string
  dfuse_auth_endpoint: string
  secure: boolean

  network_id: string
  network?: EosqNetwork
  available_networks: EosqNetwork[]

  chain_core_symbol: string
  chain_core_symbol_code: string
  chain_core_symbol_precision: number
  chain_core_asset_format: string

  display_price: boolean
  disable_segments: boolean
  disable_sentry: boolean
}

function newConfig() {
  const coreSymbolParts = windowTS.TopLevelConfig.chain_core_symbol.split(",")
  const coreSymbolPrecision = parseInt(coreSymbolParts[0])
  const coreSymbolCode = coreSymbolParts[1]

  const config = {
    ...windowTS.TopLevelConfig,
    chain_core_symbol_precision: coreSymbolPrecision,
    chain_core_symbol_code: coreSymbolCode,
    chain_core_asset_format: "0,0." + "0".repeat(coreSymbolPrecision),
    isLocalhost,
  } as EosqConfig

  config.network = config.available_networks.find((network) => network.id === config.network_id)

  debugLog("Loaded config %O", config)
  return config
}

export const Config = newConfig()
// ;(function init() {
//   debugLog("Performing init phase of config")
//   const { network } = Config
//   if (network?.pageTitle) {
//     document.title = network.pageTitle
//   }

//   if (network?.faviconTemplate) {
//     changeFavicon(network.faviconTemplate)
//   }
// })()

// function changeFavicon(src: string) {
//   const link = document.createElement("link")
//   link.rel = "shortcut icon"
//   link.href = src

//   const oldLinks = document.querySelectorAll('link[rel="shortcut icon"]')
//   debugLog("Removing all old favicon links (%s)", oldLinks.length)
//   oldLinks.forEach((element) => {
//     document.head.removeChild(element)
//   })

//   debugLog("Appending new favicon link to browser", link)
//   document.head.appendChild(link)
// }
