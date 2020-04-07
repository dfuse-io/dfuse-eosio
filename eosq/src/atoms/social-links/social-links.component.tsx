import { styled } from "../../theme"
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome"
import { IconProp } from "@fortawesome/fontawesome-svg-core"
import * as React from "react"
import { ExternalTextLink } from "../text/text.component"
import { faCheckCircle } from "@fortawesome/free-regular-svg-icons"

export const SocialIcon: React.ComponentType<any> = styled(FontAwesomeIcon)`
  color: ${(props) => props.theme.colors.grey5};
  width: 30px;
  height: 35px;

  display: inline-block;
  &:hover {
    color: ${(props) => props.theme.colors.text};
  }
`

export const SocialIconWrapper: React.ComponentType<any> = styled.div`
  display: inline-block;
  width: 30px;
  height: 30px;
  margin-right: 18px;
  position: relative;
`

export const BadgeCheck: React.ComponentType<any> = styled(FontAwesomeIcon)`
  color: ${(props) => props.theme.colors.primary};

  position: absolute;
  bottom: -10px;
  right: -16px;
  width: 20px;
  height: 20px;
  z-index: 30;
`

export const SocialIconBackground: React.ComponentType<any> = styled.div`
  position: absolute;
  bottom: -10px;
  right: -11px;
  width: 18px;
  height: 18px;
  background-color: ${(props) => props.theme.colors.ternary};
  border-radius: 10px;
  z-index: 2;
  box-shadow: 1px 2px 5px 1px ${(props) => props.theme.colors.grey5};
`

export const SocialLinksContainer: React.ComponentType<any> = styled.div`
  min-width: 100px;
  padding-top: 6px;
`

export interface SocialNetwork {
  url: string
  name: IconProp
  verified: boolean
}

interface Props {
  socialNetworks: SocialNetwork[]
  verifiedTitle: string
}

export const SocialLinks: React.SFC<Props> = ({ socialNetworks, verifiedTitle }) => {
  function renderSocialIcon(socialNetwork: SocialNetwork, index: number): JSX.Element {
    return (
      <ExternalTextLink key={index} to={socialNetwork.url}>
        <SocialIconWrapper title={socialNetwork.verified ? verifiedTitle : ""}>
          <SocialIcon size="3x" icon={socialNetwork.name as IconProp} />
          {socialNetwork.verified ? <BadgeCheck size="2x" icon={faCheckCircle as any} /> : null}
          {socialNetwork.verified ? <SocialIconBackground /> : null}
        </SocialIconWrapper>
      </ExternalTextLink>
    )
  }

  function renderSocialIcons(networks: SocialNetwork[]): JSX.Element[] {
    return networks.map((socialNetwork: SocialNetwork, index: number) =>
      renderSocialIcon(socialNetwork, index)
    ) as JSX.Element[]
  }

  return <SocialLinksContainer>{renderSocialIcons(socialNetworks)}</SocialLinksContainer>
}
