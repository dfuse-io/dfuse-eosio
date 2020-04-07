import * as React from "react"

import { withRouter, RouteComponentProps } from "react-router"
import * as Cookies from "js-cookie"
import { SettingsSelector } from "./settings-selector"
import { getCurrentLanguageValue, LANGUAGE_OPTIONS } from "./settings.helpers"

interface Props extends RouteComponentProps<{}> {
  variant: "dark" | "light"
}

class LanguageSelectorContainer extends React.Component<Props> {
  onSelectLanguage = (value: string) => {
    Cookies.set("i18next", value)

    window.location.reload()
  }

  render() {
    return (
      <SettingsSelector
        options={LANGUAGE_OPTIONS}
        currentOption={getCurrentLanguageValue()}
        variant={this.props.variant}
        onSelect={this.onSelectLanguage}
      />
    )
  }
}

export const LanguageSelector = withRouter(LanguageSelectorContainer)
