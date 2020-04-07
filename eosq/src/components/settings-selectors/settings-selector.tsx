import * as React from "react"

import { withRouter, RouteComponentProps } from "react-router"
import ListItem from "@material-ui/core/ListItem/ListItem"
import ListItemText from "@material-ui/core/ListItemText/ListItemText"
import List from "@material-ui/core/List/List"
import { theme, styled } from "../../theme"
import { Text } from "../../atoms/text/text.component"
import { faCheck, faCircle } from "@fortawesome/free-solid-svg-icons"
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome"
import { Cell } from "../../atoms/ui-grid/ui-grid.component"

export const UpperText: React.ComponentType<any> = styled(Text)`
  text-transform: uppercase;
`

interface Props extends RouteComponentProps<{}> {
  variant: "dark" | "light"
  options: { label: string; value: string }[]
  currentOption: string
  onSelect: (value: string) => void
}

class SelectorContainer extends React.Component<Props> {
  renderActiveIcon() {
    if (this.props.variant === "dark") {
      return <FontAwesomeIcon color={theme.colors.bleu8} icon={faCircle as any} size="lg" />
    }
    return <FontAwesomeIcon color={theme.colors.ternary} icon={faCheck as any} size="lg" />
  }

  renderInactiveIcon() {
    if (this.props.variant === "dark") {
      return <FontAwesomeIcon color={theme.colors.grey4} icon={faCircle as any} size="lg" />
    }
    return <Cell width="19px" />
  }

  onSelect(event: Event, value: string) {
    event.preventDefault()
    this.props.onSelect(value)
  }

  renderTextLink(option: { label: string; value: string }) {
    let color = theme.colors.primary
    if (this.props.variant === "dark") {
      color = option.value !== this.props.currentOption ? theme.colors.grey5 : theme.colors.bleu8
    }
    return (
      <UpperText fontSize={[3]} color={color}>
        {option.label}
      </UpperText>
    )
  }

  renderActiveNetworkIcon(option: { label: string; value: string }) {
    if (option.value !== this.props.currentOption) {
      return this.renderInactiveIcon()
    }
    return this.renderActiveIcon()
  }

  render() {
    return (
      <List style={{ width: "100%" }}>
        {this.props.options.map((option: { label: string; value: string }) => {
          return (
            <Cell key={option.value} onClick={(event: Event) => this.onSelect(event, option.value)}>
              <ListItem button={true}>
                {this.renderActiveNetworkIcon(option)}
                <ListItemText primary={this.renderTextLink(option)} />
              </ListItem>
            </Cell>
          )
        })}
      </List>
    )
  }
}

export const SettingsSelector = withRouter(SelectorContainer)
