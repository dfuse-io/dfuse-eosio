import * as React from "react"
import { withRouter, RouteComponentProps } from "react-router"
import { t } from "i18next"
import ListItem from "@material-ui/core/ListItem/ListItem"
import ListItemText from "@material-ui/core/ListItemText/ListItemText"
import List from "@material-ui/core/List/List"
import { theme, styled } from "../../theme"
import { PictoTransactions } from "./svg/picto-transactions-01"
import { PictoProducer } from "./svg/picto-producer-01"
import { PictoBlocks } from "./svg/picto-blocks-01"
import { TextLink } from "../../atoms/text/text.component"
import { Cell } from "../../atoms/ui-grid/ui-grid.component"

export const UpperTextLink: React.ComponentType<any> = styled(TextLink)`
  text-transform: uppercase;
`

interface Props {
  variant: "dark" | "light"
}

class MainMenuContainer extends React.Component<RouteComponentProps<any> & Props, any> {
  matchesCurrentPath(pathnames: string[]): boolean {
    let isMatch = false

    pathnames.forEach((pathName: string) => {
      if (pathName === "/") {
        isMatch = isMatch || this.props.location.pathname === pathName
      } else {
        isMatch = isMatch || this.props.location.pathname.includes(pathName)
      }
    })
    return isMatch
  }

  renderTextLink(text: string, path: string) {
    return (
      <UpperTextLink
        fontWeight={this.props.variant === "dark" ? "600" : "400"}
        fontSize={[3]}
        to={path}
        color={this.color}
      >
        {text}
      </UpperTextLink>
    )
  }

  get color() {
    return this.props.variant === "dark" ? theme.colors.bleu8 : theme.colors.primary
  }

  navigate(path: string) {
    this.props.history.push(path)
  }

  render() {
    return (
      <List>
        <Cell key="transactions" onClick={() => this.navigate("/transactions")}>
          <ListItem button={true}>
            <PictoTransactions color={this.color} />
            <ListItemText
              primary={this.renderTextLink(t("navbar.transactions"), "/transactions")}
            />
          </ListItem>
        </Cell>
        <Cell key="blocks" onClick={() => this.navigate("/blocks")}>
          <ListItem button={true}>
            <PictoProducer color={this.color} />
            <ListItemText primary={this.renderTextLink(t("navbar.blocks"), "/blocks")} />
          </ListItem>
        </Cell>
        <Cell key="producers" onClick={() => this.navigate("/producers")}>
          <ListItem button={true}>
            <PictoBlocks color={this.color} />
            <ListItemText primary={this.renderTextLink(t("navbar.producers"), "/producers")} />
          </ListItem>
        </Cell>
      </List>
    )
  }
}

export const MainMenu = withRouter(MainMenuContainer)
