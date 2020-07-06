import { t } from "i18next"
import * as React from "react"
import { styled } from "../../theme"
import { Box } from "@dfuse/explorer"

const Title: React.ComponentType<any> = styled.h1`
  display: inline-block;
  border-right: 1px solid ${(props) => props.theme.colors.text};
  margin: 0;
  margin-right: 20px;
  padding: 10px 23px 10px 0;
  font-size: 24px;
  font-weight: 500;
  vertical-align: top;
  color: ${(props) => props.theme.colors.text};
`

const Subtitle: React.ComponentType<any> = styled.h2`
  font-size: 14px;
  font-weight: normal;
  line-height: inherit;
  margin: 0;
  padding: 0;
  color: ${(props) => props.theme.colors.text};
`

const Content: React.ComponentType<any> = styled(Box)`
  display: inline-block;
  text-align: left;
  line-height: 49px;
  height: 49px;
  vertical-align: middle;
`

const Wrapper: React.ComponentType<any> = styled(Box)`
  justify-content: center;
  margin: 35vh 0;
`

export const NotFound = () => (
  <Wrapper>
    <Title>404</Title>
    <Content>
      <Subtitle>{t("core.notFoundErrorMessage")}</Subtitle>
    </Content>
  </Wrapper>
)
