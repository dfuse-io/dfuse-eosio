import * as React from "react"
import { color as color_, fontSize } from "styled-system"
import { Link } from "react-router-dom"
import { Cell, Grid } from "../../atoms/ui-grid/ui-grid.component"
import { Links } from "../../routes"
import { t } from "i18next"
import { styled } from "../../theme"
import { getActiveNetworkConfig } from "../../models/config"
import { Box, Text } from "@dfuse/explorer"
import { Img } from "../../atoms/img"

const LogoElement: React.ComponentType<any> = styled.div`
  font-family: "Lato", sans-serif;
  font-weight: 600;
  ${color_};
  ${fontSize};
  top: -10px;
  position: relative;

  @media (max-width: 767px) {
    top: -6px;
  }
`

const Tagline: React.ComponentType<any> = styled.span`
  font-family: "Lato", sans-serif;
  font-weight: 600;
  color: ${(props) => props.theme.colors.logo1};
  ${fontSize};
  letter-spacing: 1px;
`

const LogoLink: React.ComponentType<any> = styled(Link)`
  display: block;
  display: flex;
  align-items: center;
  justify-content: center;
`

interface Props {
  variant: "dark" | "light"
}

export const HeaderLogo: React.FC<Props> = () => {
  return (
    <Grid gridTemplateColumns={["auto auto"]} gridRow={["1"]} gridColumn={["1"]} py={[1, 0]}>
      <Cell alignSelf="center" justifySelf="right">
        <LogoLink to={Links.home()}>
          <Logo />
        </LogoLink>
      </Cell>
      <Cell pl={[0, 1, 3]} mr={[1, 2]} alignSelf="center" justifySelf="left">
        <Tagline color="#fff" fontWeight="400">
          <Cell display={["none", "block"]} fontSize={[0, 1, 2]}>
            {t("core.tagline")}
          </Cell>
          <Cell display={["none", "block"]} fontSize={[0, 1, 2]}>
            {t("core.tagline2")}
          </Cell>
        </Tagline>
      </Cell>
    </Grid>
  )
}

const Logo: React.FC = () => {
  const networkConfig = getActiveNetworkConfig()
  if (networkConfig && networkConfig.logo && networkConfig.logo_text) {
    return <LogoImageAndText image={networkConfig.logo} text={networkConfig.logo_text} />
  }

  if (networkConfig && networkConfig.logo) {
    return <LogoImage image={networkConfig.logo} />
  }

  return <LogoDefault />
}

const LogoDefault: React.FC = () => (
  <>
    <LogoElement px={[0]} color="white" fontSize={["40px", "56px", "56px"]}>
      eos
    </LogoElement>
    <LogoElement px={[0]} color="logo2" fontSize={["40px", "56px"]}>
      q
    </LogoElement>
  </>
)

const LogoImage: React.FC<{ image: string }> = ({ image }) => (
  <Img src={image} alt="Logo" minWidth="70px" maxHeight="70px"></Img>
)

const LogoText = styled(Text)`
  font-family: "Lato", sans-serif;
  font-weight: 400;
`

const LogoImageAndText: React.FC<{ image: string; text?: string }> = ({ image, text }) => (
  <Box pa={[0]} alignItems="center" justifyContent="center" minWidth="150px" flexWrap="wrap">
    <Img src={image} alt="Logo" title={text} width="48px" height="48px"></Img>
    {text ? (
      text === "eosq" ? (
        <Box mx={[2]}>
          <LogoDefault />
        </Box>
      ) : (
        <LogoText color="white" mx={[2]} fontSize={[4]}>
          {text}
        </LogoText>
      )
    ) : null}
  </Box>
)
