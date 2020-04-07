import Switch from "@material-ui/core/Switch"

import * as React from "react"
import { createStyles, withStyles, WithStyles } from "@material-ui/core/styles"
import { theme } from "../../theme"

const styles = () =>
  createStyles({
    iOSSwitchBase: {
      "&$iOSChecked": {
        color: theme.colors.primary,
        "& + $iOSBar": {
          backgroundColor: theme.colors.ternary
        }
      }
    },
    iOSSwitchBaseDark: {
      "&$iOSChecked": {
        color: theme.colors.primary,
        "& + $iOSBar": {
          backgroundColor: `${theme.colors.bleu11} !important`
        }
      }
    },

    iOSChecked: {
      transform: "translateX(14px)",
      "& + $iOSBar": {
        opacity: 1,
        border: "none"
      }
    },
    iOSBar: {
      borderRadius: 9,
      width: 32,
      height: 18,
      marginTop: -9,
      marginLeft: -16,
      border: "solid 1px",
      borderColor: theme.colors.border,
      backgroundColor: theme.colors.formFieldBg,
      opacity: "1 !important" as any
    },
    iOSBarDark: {
      borderRadius: 9,
      width: 32,
      height: 18,
      marginTop: -9,
      marginLeft: -16,
      border: "solid 1px",
      borderColor: `${theme.colors.bleu11} !important`,
      backgroundColor: `${theme.colors.bleu11} !important`,
      opacity: "1 !important" as any
    },

    iOSIcon: {
      width: 16,
      height: 16
    },
    iOSIconChecked: {}
  })

const decorate = withStyles(styles)

interface Props {
  onChange: (checked: boolean) => void
  checked?: boolean
  variant?: "light" | "dark"
}

export const UiSwitch = decorate<any>(
  class extends React.Component<Props & WithStyles<typeof styles>, {}> {
    state = { checked: false }

    constructor(props: Props & WithStyles<typeof styles>) {
      super(props)
      this.state = { checked: props.checked || false }
    }

    handleChange = (event: any) => {
      this.setState({ checked: event.target.checked })
      this.props.onChange(event.target.checked)
    }

    render() {
      return (
        <Switch
          classes={{
            switchBase:
              this.props.variant === "dark"
                ? this.props.classes.iOSSwitchBaseDark
                : this.props.classes.iOSSwitchBase,
            bar:
              this.props.variant === "dark"
                ? this.props.classes.iOSBarDark
                : this.props.classes.iOSBar,
            icon: this.props.classes.iOSIcon,
            iconChecked: this.props.classes.iOSIconChecked,
            checked: this.props.classes.iOSChecked
          }}
          color="primary"
          disableRipple={true}
          checked={this.state.checked}
          onChange={this.handleChange}
          value="checkedB"
        />
      )
    }
  }
)
