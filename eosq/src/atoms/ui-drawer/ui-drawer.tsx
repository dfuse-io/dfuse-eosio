import * as React from "react"
import Hidden from "@material-ui/core/Hidden"
import { Cell } from "../ui-grid/ui-grid.component"
import { styled } from "../../theme"
import { observer } from "mobx-react"
import Drawer from "@material-ui/core/Drawer/Drawer"

interface State {
  opened: boolean
}

const NoFocus = styled.div`
  height: 100%;
  &:focus {
    outline: none !important;
  }
`

// The state of the drawer can be forced from parent
// with the parameters: onOpen, onClose, opened
interface Props {
  onOpen: () => any
  onClose: () => any
  mobileOpener: JSX.Element
  opener: JSX.Element
  content: JSX.Element
  renderMobileContent: (onClose: () => void) => JSX.Element
  opened?: boolean
}

const Container: React.ComponentType<any> = styled.div`
  text-align: center;
`

@observer
export class UiDrawer extends React.Component<Props, State> {
  state = {
    opened: false
  }

  componentDidUpdate(prevProps: Props): void {
    // Force state from parent if it decides to update the state
    if (this.props.opened !== undefined && prevProps.opened !== this.props.opened) {
      // eslint-disable-next-line react/no-did-update-set-state
      this.setState({
        opened: this.props.opened
      })
    }
  }

  componentDidMount(): void {
    if (this.props.opened !== undefined) {
      if (this.props.opened !== this.state.opened) {
        this.setState({
          opened: this.props.opened
        })
      }
    }
  }

  toggleDrawer = (open: boolean) => () => {
    if (open) {
      this.props.onOpen()
    } else {
      this.props.onClose()
    }
    this.setState({
      opened: open
    })
  }

  closeDrawer = () => {
    this.props.onClose()
    this.setState({
      opened: false
    })
  }

  render() {
    return (
      <Container>
        <Hidden xsDown={true} implementation="js">
          <Cell onClick={this.toggleDrawer(true)}>{this.props.opener}</Cell>
        </Hidden>
        <Hidden smUp={true} implementation="js">
          <Cell onClick={this.toggleDrawer(true)}>{this.props.mobileOpener}</Cell>
        </Hidden>
        <Hidden xsDown={true} implementation="js">
          <Drawer anchor="top" open={this.state.opened} onClose={this.toggleDrawer(false)}>
            <div
              tabIndex={0}
              role="button"
              onClick={this.toggleDrawer(false)}
              onKeyDown={this.toggleDrawer(false)}
            >
              {this.props.content}
            </div>
          </Drawer>
        </Hidden>
        <Hidden smUp={true} implementation="js">
          <Drawer
            PaperProps={{ style: { height: "100vh" } }}
            anchor="top"
            open={this.state.opened}
            onClose={this.toggleDrawer(false)}
          >
            <NoFocus tabIndex={0} role="button">
              {this.props.renderMobileContent(this.closeDrawer)}
            </NoFocus>
          </Drawer>
        </Hidden>
      </Container>
    )
  }
}
