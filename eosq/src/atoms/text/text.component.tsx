import * as React from "react"
import { Link } from "react-router-dom"
import {
  alignSelf,
  color,
  display,
  fontFamily,
  fontSize,
  fontWeight,
  justifySelf,
  lineHeight,
  space,
  textAlign,
  borders,
  width,
  compose
} from "styled-system"
import { styled } from "../../theme"

const textStyle = compose(
  alignSelf,
  borders,
  color,
  display,
  fontFamily,
  fontSize,
  fontWeight,
  lineHeight,
  justifySelf,
  space,
  textAlign,
  width
)

export const Text: React.ComponentType<any> = styled.div`
  position: relative;
  text-transform: ${(props: any) => props.textTransform};
  text-overflow: ${(props: any) => props.textOverflow};
  white-space: ${(props: any) => props.whiteSpace};
  word-break: ${(props: any) => props.wordBreak};

  ${textStyle}
`

export const HoverableText: React.ComponentType<any> = styled(Text)`
  &:hover {
    cursor: pointer;
    color: ${(props) => props.theme.colors.linkHover};
  }
`

export const HoverableTextNoHighlight: React.ComponentType<any> = styled(Text)`
  &:hover {
    cursor: pointer;
  }
`

export const EllipsisText: React.ComponentType<any> = styled(Text)`
  text-overflow: ellipsis;
  white-space: nowrap;
  overflow: hidden;
`

export const CondensedBold: React.ComponentType<any> = styled.b`
  font-family: "Roboto Condensed", sans-serif;
  font-weight: 800;
`

export const BigTitle: React.ComponentType<any> = styled.h1`
  ${textStyle}
`

export const Title: React.ComponentType<any> = styled.h2`
  ${textStyle}
`

export const SubTitle: React.ComponentType<any> = styled.h3`
  ${textStyle}
`

Text.defaultProps = {
  color: "text"
}

BigTitle.defaultProps = {
  color: "text"
}

Title.defaultProps = {
  color: "text"
}

SubTitle.defaultProps = {
  color: "text",
  my: [2]
}

export interface TextLinkProps {
  whiteSpace?: string
  lineHeight?: string
  download?: string
  to: string
  fontSize?: any
  fontFamily?: any
  fontWeight?: any
  style?: any
  pt?: any
  pb?: any
  pr?: any
  p?: any
  textAlign?: any
  color?: any
  pl?: any
  width?: any
  mr?: any
  ml?: any
  my?: any
  mx?: any
}

export const LinkStyledText: React.ComponentType<any> = styled(HoverableText)`
  display: inline;
  ${textStyle}
`

export const StyledLink: React.ComponentType<any> = styled(Link)`
  ${fontSize};
`

export const TextLinkLight: React.SFC<TextLinkProps> = ({ to, children, ...rest }) => {
  return (
    <Link to={to}>
      <LinkStyledText color="link2" {...rest}>
        {children}
      </LinkStyledText>
    </Link>
  )
}

export const TextLink: React.SFC<TextLinkProps> = ({ to, children, ...rest }) => {
  return (
    <StyledLink fontSize={rest && rest.fontSize ? rest.fontSize : ""} to={to}>
      <LinkStyledText color="link" {...rest}>
        {children}
      </LinkStyledText>
    </StyledLink>
  )
}

export const ExternalTextLink: React.SFC<TextLinkProps> = ({ to, download, children, ...rest }) => {
  if (download) {
    return (
      <a href={to} target="_blank" rel="noopener noreferrer" download={download}>
        <LinkStyledText color="link" {...rest}>
          {children}
        </LinkStyledText>
      </a>
    )
  }
  return (
    <a href={to} target="_blank" rel="noopener noreferrer" {...download}>
      <LinkStyledText color="link" {...rest}>
        {children}
      </LinkStyledText>
    </a>
  )
}

export const ExternalTextLinkLight: React.SFC<TextLinkProps> = ({ to, children, ...rest }) => {
  return (
    <a href={to} target="_blank" rel="noopener noreferrer">
      <LinkStyledText color="link2" {...rest}>
        {children}
      </LinkStyledText>
    </a>
  )
}

export class KeyValueFormatEllipsis extends React.Component<{ content: string }> {
  render() {
    const regex: RegExp = /(\S*: )/g
    return (
      <EllipsisText fontFamily="Roboto Condensed" fontSize={[1]}>
        {this.props.content.split(regex).map((value: string, index: number) => {
          if (regex.test(value)) {
            return <CondensedBold key={index}>{value}</CondensedBold>
          }

          return value
        })}
      </EllipsisText>
    )
  }
}
