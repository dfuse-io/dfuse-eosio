import * as React from "react"
import { Text } from "../../atoms/text/text.component"
import { MonospaceTextLink } from "../../atoms/text-elements/misc"
import { Links } from "../../routes"
import { formatDateFromString } from "../../helpers/moment.helpers"
import { formatDateTime } from "../../helpers/formatters"

interface Props {
  value: string | number
  type: string
  label: string
}

export class FormattedContractElement extends React.Component<Props> {
  renderAccountName() {
    return (
      <MonospaceTextLink to={Links.viewAccount({ id: this.props.value.toString() })}>
        {this.props.value}
      </MonospaceTextLink>
    )
  }

  renderDate() {
    if (typeof this.props.value === "string") {
      return /[0-9]+/.test(this.props.value)
        ? this.renderDateFromInt(parseInt(this.props.value, 10))
        : this.renderDateFromString(this.props.value)
    }

    if (typeof this.props.value === "number") {
      return this.renderDateFromInt(this.props.value)
    }

    return this.renderDateFromString(this.props.value)
  }

  renderDateFromInt(value: number) {
    let valueInMilliseconds = value
    if (this.isMaybeInSeconds(value)) {
      valueInMilliseconds *= 1000
    }
    return <Text>{formatDateTime(valueInMilliseconds)}</Text>
  }

  isMaybeInSeconds(value: number) {
    return new Date(value).getUTCFullYear() < 2000
  }

  renderDateFromString(value: string) {
    return <Text>{formatDateFromString(value, true)}</Text>
  }

  render() {
    if (this.props.type === "account_name" || this.props.type === "name") {
      return this.renderAccountName()
    }

    if (["time_point", "time_point_sec"].includes(this.props.type)) {
      return this.renderDateFromString(this.props.value as string)
    }

    if (this.props.label.includes("_at")) {
      return this.renderDate()
    }

    return this.props.value.toString()
  }
}
