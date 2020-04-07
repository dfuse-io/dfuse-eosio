import * as React from "react"
import { styled } from "../../theme"
import { Text } from "../../atoms/text/text.component"
import Box from "../../atoms/ui-box"
import { Cell } from "../../atoms/ui-grid/ui-grid.component"

const Container: React.ComponentType<any> = styled(Cell)`
  background-color: ${(props) => props.theme.colors.banner};
  margin-bottom: 1px;
`
const BorderBannerContainer: React.ComponentType<any> = styled(Container)`
  border-style: solid;
  border-color: ${(props) => props.theme.colors.bleu6};
`

export const BannerTitle: React.ComponentType<any> = styled(Text)`
  color: ${(props) => props.theme.colors.bannerTitle};
  font-family: "Roboto", sans-serif;
  text-transform: uppercase;
  font-weight: 400;
`

export const BannerDetails: React.ComponentType<any> = styled(Text)`
  margin-top: 0.12em;
  font-family: "Roboto Condensed", sans-serif;
  font-weight: 700;
`

export interface BannerContainerProps {
  titleTip?: string
  containerProps?: any
}

export const BannerContainer: React.SFC<BannerContainerProps> = ({
  titleTip,
  containerProps,
  children
}) => (
  <BorderBannerContainer
    borderTop={["0px"]}
    borderBottom={["1px", "0px", "0px"]}
    borderLeft={["0px"]}
    borderRight={["0px", "1px", "1px"]}
    px={[2, 3]}
    py={[2, 3]}
    title={titleTip || ""}
    {...(containerProps || {})}
  >
    {children}
  </BorderBannerContainer>
)

export interface BannerItemProps extends BannerContainerProps {
  align?: string[]
  title: string
  details: JSX.Element | string
}

export const BannerItem: React.SFC<BannerItemProps> = ({
  align,
  title,
  details,
  titleTip,
  containerProps
}) => (
  <BannerContainer titleTip={titleTip} containerProps={containerProps}>
    <Box flexDirection="column">
      <BannerTitle textAlign={align} fontSize={[1]} fontWeight="400">
        {title}
      </BannerTitle>
      <BannerDetails textAlign={align} color="white" fontSize={[4, 5, 5]}>
        {details}
      </BannerDetails>
    </Box>
  </BannerContainer>
)

BannerItem.defaultProps = {
  align: ["left", "left", "right"]
}
