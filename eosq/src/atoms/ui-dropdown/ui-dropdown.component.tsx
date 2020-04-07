import { faCaretDown } from "@fortawesome/free-solid-svg-icons"
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome"
import FormControl from "@material-ui/core/FormControl"
import InputLabel from "@material-ui/core/InputLabel"
import Select from "@material-ui/core/Select"
import MenuItem from "@material-ui/core/MenuItem"
import * as React from "react"
import { theme, styled } from "../../theme"
import { Text } from "../text/text.component"

const StyledFormControl: React.ComponentType<any> = styled(FormControl)`
  // min-width: 180px !important;
  width: 100%;
`

const StyledInputLabel: React.ComponentType<any> = styled(InputLabel)`
  font-size: 16px !important;
  width: 180px !important;
  padding: 0 12px !important;
  z-index: 3;
`

const StyledSelect: React.ComponentType<any> = styled(Select)`
  background-color: ${(props: any) => (props.bg ? props.bg : theme.colors.formSelectorBg)};
`

const StyledSelectSmall: React.ComponentType<any> = styled(StyledSelect)`
  width: auto;
  padding: 2px 2px 2px 2px;
`

const StyledSelectXSmall: React.ComponentType<any> = styled(StyledSelect)`
  width: auto;
  padding: 2px 2px 2px 2px;
`

const StyledIconComponent: React.ComponentType<any> = styled(FontAwesomeIcon)`
  top: calc(50% - 7px);
  right: 12px;
  position: absolute;
  pointer-events: none;
`

interface Props {
  options: DropDownOption[]
  onSelect: (label: string) => void
  placeholder?: string
  defaultValue?: string
  size?: string
  bg?: string
  selectorBg?: string
  noBorders?: boolean
  color?: string
  label?: string
  fontSize?: string
  id?: string
  value?: string
}

interface UiTextEventTarget extends EventTarget {
  index?: number
}

interface UiMouseEvent extends React.MouseEvent<HTMLElement> {
  target: UiTextEventTarget
}

export interface DropDownOption {
  label: string
  value: string
}

export class UiDropDown extends React.Component<Props, any> {
  button = null

  componentDidUpdate(prevProps: Props) {
    if (!this.props.value) {
      return
    }

    if (
      !this.optionValues.includes(this.props.value) &&
      this.props.defaultValue === this.state.selectedLabel
    ) {
      return
    }

    if (
      this.optionValues.includes(this.props.value) &&
      this.props.value !== this.state.selectedLabel
    ) {
      // eslint-disable-next-line react/no-did-update-set-state
      this.setState({ selectedLabel: this.props.value })
    } else if (this.props.value !== this.state.selectedLabel) {
      // eslint-disable-next-line react/no-did-update-set-state
      this.setState({ selectedLabel: this.props.defaultValue })
    }
  }

  get optionValues() {
    return this.props.options.map((option: DropDownOption) => option.value)
  }

  handleClick = (event: UiMouseEvent) => {}

  handleMenuItemClick = (event: any) => {
    this.setState({ selectedLabel: event.target.value })
    this.props.onSelect(event.target.value)
  }

  constructor(props: Props) {
    super(props)
    let selectedLabel = ""
    if (this.props.value && this.optionValues.includes(this.props.value)) {
      selectedLabel = this.props.value
    } else if (this.props.defaultValue) {
      selectedLabel = this.props.defaultValue
    }

    this.state = {
      selectedLabel
    }
  }

  render() {
    let SelectWrapper = StyledSelect as any
    if (this.props.size === "xs") {
      SelectWrapper = StyledSelectXSmall as any
    } else if (this.props.size === "sm") {
      SelectWrapper = StyledSelectSmall as any
    }
    let selectDisplayStyle: any = {
      paddingLeft: "12px"
    }

    if (this.props.size === "sm" || this.props.size === "xs") {
      selectDisplayStyle = {
        padding: "3px 28px 3px 3px"
      }
    }

    return (
      <StyledFormControl>
        {this.props.label ? (
          <StyledInputLabel htmlFor="table-dropdown">{this.props.label}</StyledInputLabel>
        ) : null}
        <SelectWrapper
          bg={this.props.bg}
          disableUnderline={true}
          value={this.state.selectedLabel}
          onChange={this.handleMenuItemClick}
          SelectDisplayProps={{ style: selectDisplayStyle }}
          IconComponent={() => (
            <StyledIconComponent
              icon={faCaretDown}
              color={this.props.color ? this.props.color : theme.colors.text}
            />
          )}
          style={{
            border: this.props.noBorders
              ? "none !important"
              : `1px solid ${theme.colors.formSelectorBorder} !important`
          }}
          inputProps={{
            name: this.props.id || Math.random().toString(),
            id: this.props.id || Math.random().toString(),
            style: {
              paddingRight: "30px"
            }
          }}
          MenuProps={{
            PaperProps: {
              style: {
                backgroundColor: this.props.selectorBg ? this.props.selectorBg : this.props.bg
              }
            }
          }}
        >
          {this.props.options.map((option) => (
            <MenuItem key={option.value} value={option.value}>
              <Text
                fontFamily="'Roboto Condensed', sans-serif;"
                color={this.props.color ? this.props.color : "text"}
                fontSize={this.props.fontSize ? this.props.fontSize : "16px"}
              >
                {option.label}
              </Text>
            </MenuItem>
          ))}
        </SelectWrapper>
      </StyledFormControl>
    )
  }
}
