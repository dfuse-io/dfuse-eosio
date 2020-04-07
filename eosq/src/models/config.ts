const windowTS = window as any

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
}

export const Config = windowTS.TopLevelConfig as EosqConfig
