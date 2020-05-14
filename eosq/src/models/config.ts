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
    version: 1,

    current_network: "eos-mainnet",
    on_demand: false,

    dfuse_io_endpoint: "mainnet.eos.dfuse.io",
    dfuse_io_api_key: "",
    dfuse_auth_endpoint: "https://auth.dfuse.io",
    display_price: true,
    price_ticker_name: "EOS",

    available_networks: []
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
