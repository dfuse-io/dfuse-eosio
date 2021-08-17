import * as React from "react"
import { color as color_, fontSize } from "styled-system"
import { Link } from "react-router-dom"
import { Cell, Grid } from "../../atoms/ui-grid/ui-grid.component"
import { Links } from "../../routes"
import { t } from "i18next"
import { styled } from "../../theme"
import { Config } from "../../models/config"
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
    </Grid>
  )
}

const Logo: React.FC = () => {
  const { network } = Config
  if (network?.logo) {
    if (network.logo_text) {
      return <LogoImageAndText image={network.logo} text={network.logo_text} />
    }

    return <LogoImage image={network.logo} />
  }

  return <LogoUltra />
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

const LogoUltra: React.FC = () => (
  <>
    <LogoElement>
      

<svg width="200px" height="80px" viewBox="0 0 200 80">
    <g id="Page-1" stroke="none" stroke-width="1" fill="none" fill-rule="evenodd">
        <rect id="Rectangle" x="0" y="0" width="200" height="80"></rect>
        <g id="Group" transform="translate(8.000000, 28.000000)" fill="#FFFFFF">
            <polygon id="Path" fill-rule="nonzero" points="91 12 95.000453 12 95.000453 26.1999959 107 26.1999959 107 30 91 30"></polygon>
            <polygon id="Path" fill-rule="nonzero" points="111 12 129 12 129 15.8000041 121.999933 15.8000041 121.999933 30 118.000067 30 118.000067 15.8000041 111 15.8000041"></polygon>
            <path d="M154.621726,13.785611 C153.410669,12.6627104 151.666264,12 149.516563,12 L139,12 L139,30 L142.999809,30 L142.999809,23.7510026 L149.590253,23.7510026 L153.048258,30 L158,30 L153.672813,22.7760835 C155.405137,21.7412512 156.436197,20.0411433 156.436197,18 C156.436197,16.4297347 155.828554,14.9041143 154.621726,13.785611 Z M142.999809,19.9999796 L142.999809,15.8000041 L149.42898,15.8000041 C150.60138,15.8000041 151.296605,16.1421999 151.703109,16.5347815 C152.126526,16.9435476 152.328268,17.4954093 152.328268,18.0351784 C152.328268,18.5536329 152.140418,19.0179149 151.752639,19.3547973 C151.367275,19.6900918 150.667218,19.9999796 149.42898,19.9999796 L142.999809,19.9999796 Z" id="Shape"></path>
            <polygon id="Path" fill-rule="nonzero" points="164 30 168.249947 30 173.999698 16.9999796 179.750053 30 184 30 176.000242 12 171.999758 12"></polygon>
            <path d="M62,21.0621275 L62,13 L66.0003383,13 L66.0003383,21.1250067 C66.0106068,23.1756779 66.6049766,24.5552148 67.5889486,25.5124765 C68.5970819,26.4826644 69.9434745,27.0439262 72.0002114,27.0439262 C74.0611766,27.0439262 75.422066,26.4801879 76.4271791,25.5124161 C77.4032986,24.5631879 78.0000846,23.1689128 78.0000846,21.0621275 L78.0000846,13 L82,13 L82,21.0621275 C82,23.9563221 81.1422712,26.477349 79.220838,28.3437315 C77.3277943,30.1678926 74.8917238,31 72.0002114,31 C69.1044708,31 66.6859172,30.1654161 64.7952897,28.3437315 C62.8841251,26.486651 62.0128055,23.9218322 62,21.0621275 Z" id="Path" fill-rule="nonzero"></path>
            <path d="M41,20.5 C41,2.41176252 38.5883118,0 20.5,0 C2.41173156,0 0,2.41176252 0,20.5 C0,38.5882499 2.41173156,41 20.5,41 C38.5883118,41 41,38.5882499 41,20.5 Z M10.250031,20.6555249 L10.250031,12.3000248 L14.3500186,12.3000248 L14.3500186,20.7206571 C14.3605437,22.8459369 14.9701369,24.2756885 15.9788198,25.2677788 C17.0120821,26.2733041 18.3920558,26.8549746 20.5,26.8549746 C22.6124638,26.8549746 24.0071728,26.2707038 25.0378347,25.2677169 C26.0379118,24.2839229 26.6499814,22.8389407 26.6499814,20.6555249 L26.6499814,12.3000248 L30.750031,12.3000248 L30.750031,20.6555249 C30.750031,23.655013 29.8706219,26.267732 27.9013624,28.2020727 C25.9609542,30.0926411 23.4635168,30.9549622 20.5,30.9549622 C17.5320874,30.9549622 15.0530381,30.0900408 13.1152922,28.2020108 C11.1558767,26.2774523 10.2631565,23.6192274 10.250031,20.6555249 Z M22.5499938,20.5 L22.5499938,12.3000248 L18.4500062,12.3000248 L18.4500062,20.5227839 C18.4534114,21.2961363 18.6566091,21.8162655 18.9929193,22.177217 C19.337402,22.5428738 19.79729,22.7543679 20.5,22.7543679 C21.2041959,22.7543679 21.669037,22.5418832 22.0125909,22.1771551 C22.3459293,21.8194231 22.5499938,21.2939694 22.5499938,20.5 Z" id="Shape"></path>
        </g>
    </g>
</svg>
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
    <Img src={image} alt="Logo" title={text} maxWidth="48px" maxHeight="48px"></Img>
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
