import * as React from "react"
import { Trans, translate } from "react-i18next"
import { MonospaceTextLink } from "../../atoms/text-elements/misc"
import { ExternalTextLink, Text } from "../../atoms/text/text.component"
import Box from "../../atoms/ui-box/ui-box.component"
import { Links } from "../../routes"
import { SearchShortcut } from "../search-shortcut/search-shortcut"

interface Field {
  type: string
  value: string | number
  name: string
  query?: string
}

interface Props {
  i18nKey: string
  fields: Field[]
  fontSize: number[] | string
}

export class BaseFormattedText extends React.Component<Props> {
  renderAccountLink(value: string, index: number): JSX.Element {
    return (
      <MonospaceTextLink
        fontSize={this.props.fontSize}
        key={index}
        to={Links.viewAccount({ id: value })}
      >
        {value}
      </MonospaceTextLink>
    )
  }

  renderLink(value: string, index: number): JSX.Element {
    return (
      <ExternalTextLink fontSize={this.props.fontSize} key={index} to={value}>
        {value}
      </ExternalTextLink>
    )
  }

  renderBold(value: string | number, index: number): JSX.Element {
    return (
      <Text fontSize={this.props.fontSize} fontWeight="bold" key={index}>
        {value}
      </Text>
    )
  }

  renderPlain(value: string | number, index: number): JSX.Element {
    return (
      <Text fontSize={this.props.fontSize} key={index}>
        {value}
      </Text>
    )
  }

  renderSearchShortcut(value: any, query: string, index: number): JSX.Element {
    return (
      <SearchShortcut
        fixed={true}
        fontWeight="bold"
        lineHeight={this.props.fontSize}
        fontSize={this.props.fontSize}
        query={query}
        key={index}
      >
        {value}
      </SearchShortcut>
    )
  }

  render() {
    const values = {}
    this.props.fields.forEach((field) => {
      values[field.name] = field.value
    })

    const components = this.props.fields.map((field: Field, index: number) => {
      if (field.type === "accountLink") {
        return this.renderAccountLink(field.value as string, index)
      }

      if (field.type === "bold") {
        return this.renderBold(field.value, index)
      }

      if (field.type === "link") {
        return this.renderLink(field.value as string, index)
      }

      if (field.type === "searchShortcut") {
        return this.renderSearchShortcut(field.value, field.query!, index)
      }

      return this.renderPlain(field.value, index)
    })

    return (
      <Box fontSize={this.props.fontSize} whiteSpace="pre-wrap">
        <Trans i18nKey={this.props.i18nKey} values={values} components={components} />
      </Box>
    )
  }
}

export const FormattedText = translate()(BaseFormattedText)
