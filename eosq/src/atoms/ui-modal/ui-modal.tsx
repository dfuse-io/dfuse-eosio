import Modal from "@material-ui/core/Modal/Modal"
import * as React from "react"
import { Cell, Grid } from "../ui-grid/ui-grid.component"
import { Text } from "../text/text.component"
import { styled } from "../../theme"
import { faTimes } from "@fortawesome/free-solid-svg-icons"
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome"

interface Props {
  opener: JSX.Element | string
  headerTitle?: JSX.Element | string
  onOpen?: () => void
  opened?: boolean
}

interface State {
  open: boolean
}

const HoverableIcon = styled(FontAwesomeIcon)`
  &:hover {
    cursor: pointer;
    color: ${(props) => props.theme.colors.linkHover};
  }
`

const ModalContainer: React.ComponentType<any> = styled(Grid)`
  position: relative;
  top: 8.5vh;

  @media (min-width: 767px) {
    top: 100px;
    max-height: calc(90vh - 100px);
  }

  margin-left: auto;
  margin-right: auto;
  overflow-y: scroll;
  width: 95vw;
  max-width: 800px;
  height: auto;
  max-height: 90vh;

  background-color: ${(props) => props.theme.colors.primary};
  box-shadow: 1px 2px 5px 1px ${(props) => props.theme.colors.grey7};
  outline: none;
`

const ModalHeader: React.ComponentType<any> = styled(Grid)`
  grid-template-columns: auto 20px;
  width: 100%;
  height: auto;
  background-color: white;
  border-bottom: 1px solid ${(props) => props.theme.colors.grey4};
`

export class UiModal extends React.Component<Props, State> {
  constructor(props: Props) {
    super(props)
    this.state = { open: this.props.opened !== undefined ? this.props.opened : false }
  }

  componentDidUpdate(prevProps: Props): void {
    if (
      this.props.opened !== undefined &&
      this.state.open !== this.props.opened &&
      this.props.opened !== prevProps.opened
    ) {
      // eslint-disable-next-line react/no-did-update-set-state
      this.setState({ open: this.props.opened })
    }
  }

  handleOpen = () => {
    this.setState({ open: true })
    if (this.props.onOpen) {
      this.props.onOpen()
    }
  }

  handleClose = () => {
    this.setState({ open: false })
  }

  render() {
    return (
      <Cell display="inline-block">
        <Text justifySelf="left" onClick={this.handleOpen}>
          {this.props.opener}
        </Text>

        <Modal onClose={this.handleClose} open={this.state.open}>
          <ModalContainer>
            {this.props.headerTitle ? (
              <ModalHeader p={[3]}>
                <Cell justifySelf="left">{this.props.headerTitle}</Cell>
                <Cell onClick={this.handleClose} justifySelf="center" p={[2]}>
                  <HoverableIcon icon={faTimes as any} size="2x" />
                </Cell>
              </ModalHeader>
            ) : null}
            <Cell p={[3]} bg="white">
              {this.props.children}
            </Cell>
          </ModalContainer>
        </Modal>
      </Cell>
    )
  }
}
