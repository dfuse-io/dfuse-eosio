import * as React from "react"

import { withRouter, RouteComponentProps } from "react-router"
import { SettingsSelector } from "./settings-selector"
import { Config, EosqNetwork } from "../../models/config"
import { t } from "../../i18n"

interface Props extends RouteComponentProps<{}> {
  variant: "light" | "dark"
}

interface State {
  network: string
}

class NetworkSelectorContainer extends React.Component<Props, State> {
  availableNetworks = Config.available_networks.sort((network: EosqNetwork, ref: EosqNetwork) =>
    network.is_test || network.name > ref.name ? 1 : -1
  )
  constructor(props: Props) {
    super(props)
    this.state = { network: Config.network_id }
  }

  onSelectNetwork = (value: string) => {
    if (value === this.state.network) {
      return
    }

    const selectedNetwork = Config.available_networks.find(
      (network: EosqNetwork) => network.id === value
    )
    if (selectedNetwork) window.location.href = selectedNetwork.url
  }

  render() {
    const networks = this.availableNetworks.map((network: EosqNetwork) => {
      return {
        label: t(`core.networkOptions.${network.id.replace("-", "_")}`, {
          defaultValue: network.name,
        }),
        value: network.id,
      }
    })

    return (
      <SettingsSelector
        options={networks}
        currentOption={Config.network_id}
        onSelect={this.onSelectNetwork}
        variant={this.props.variant}
      />
    )
  }
}

export const NetworkSelector = withRouter(NetworkSelectorContainer)
