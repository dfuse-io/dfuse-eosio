import * as React from "react"
import { createStyles, WithStyles, withStyles } from "@material-ui/core/styles"
import Tooltip from "@material-ui/core/Tooltip"
import { theme } from "../../theme"

const styles = () =>
  createStyles({
    lightTooltip: {
      background: theme.colors.bleu11,
      color: theme.colors.primary,
      fontSize: 12,
      maxWidth: "none"
    },
    arrowPopper: {
      opacity: 1,
      '&[x-placement*="bottom"] $arrowArrow': {
        opacity: 1,
        top: 0,
        left: 0,
        marginTop: "-0.9em",
        width: "3em",
        height: "1em",
        "&::before": {
          borderWidth: "0 1em 1em 1em",
          borderColor: `transparent transparent ${theme.colors.bleu11} transparent`
        }
      },
      '&[x-placement*="top"] $arrowArrow': {
        opacity: 1,
        bottom: 0,
        left: 0,
        marginBottom: "-0.9em",
        width: "3em",
        height: "1em",
        "&::before": {
          borderWidth: "1em 1em 0 1em",
          borderColor: `${theme.colors.bleu11} transparent transparent transparent`
        }
      },
      '&[x-placement*="right"] $arrowArrow': {
        opacity: 1,
        left: 0,
        marginLeft: "-0.9em",
        height: "3em",
        width: "1em",
        "&::before": {
          borderWidth: "1em 1em 1em 0",
          borderColor: `transparent ${theme.colors.bleu11} transparent transparent`
        }
      },
      '&[x-placement*="left"] $arrowArrow': {
        opacity: 1,
        right: 0,
        marginRight: "-0.9em",
        height: "3em",
        width: "1em",
        "&::before": {
          borderWidth: "1em 0 1em 1em",
          borderColor: `transparent transparent transparent ${theme.colors.bleu11}`
        }
      }
    },
    arrowArrow: {
      opacity: 1,
      position: "absolute",
      fontSize: 7,
      width: "3em",
      height: "3em",
      "&::before": {
        content: '""',
        margin: "auto",
        display: "block",
        width: 0,
        height: 0,
        borderStyle: "solid"
      }
    },
    button: {
      display: "table",
      position: "relative"
    },
    buttonFullWidth: {
      display: "table",
      width: "100%",
      position: "relative"
    }
  })

interface Props {
  fullWidth?: boolean
  placement?: any
}

const decorate = withStyles(styles)

export const UiToolTip = decorate<any>(
  class extends React.Component<Props & WithStyles<typeof styles>, {}> {
    state = {
      arrowRef: null
    }

    static get defaultProps() {
      return {
        fullWidth: false,
        placement: "top"
      }
    }

    handleArrowRef = (node: any) => {
      this.setState({
        arrowRef: node
      })
    }

    render() {
      const { classes } = this.props
      const children = React.Children.toArray(this.props.children)

      return (
        <Tooltip
          title={
            <>
              {children[1]}
              <span className={classes.arrowArrow} ref={this.handleArrowRef} />
            </>
          }
          enterTouchDelay={50}
          placement={this.props.placement}
          classes={{ popper: classes.arrowPopper, tooltip: classes.lightTooltip }}
          PopperProps={{
            popperOptions: {
              modifiers: {
                arrow: {
                  enabled: Boolean(this.state.arrowRef),
                  element: this.state.arrowRef
                }
              }
            }
          }}
        >
          <span
            id={Math.random().toString()}
            className={this.props.fullWidth ? classes.buttonFullWidth : classes.button}
          >
            {children[0]}
          </span>
        </Tooltip>
      )
    }
  }
)
