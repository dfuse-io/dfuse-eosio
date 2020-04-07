import * as React from "react"
import { Header } from "../../components/header"
import { Text } from "../../atoms/text/text.component"
import Box, { HomeWrapper } from "../../atoms/ui-box"

const About = () => {
  return (
    <HomeWrapper width="100%" align="center" flexDirection="column">
      <Header />
      <Box
        width={["100%", "100%", "95%", "1440px"]}
        flexDirection={["row", "row", "row"]}
        py={[4]}
        textAlign="center"
        justify="center"
      >
        <Text>About</Text>
      </Box>
    </HomeWrapper>
  )
}

export default About
