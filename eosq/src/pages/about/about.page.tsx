import * as React from "react"
import { Box, HomeWrapper } from "@dfuse/explorer"
import { Header } from "../../components/header"
import { Text } from "../../atoms/text/text.component"

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
