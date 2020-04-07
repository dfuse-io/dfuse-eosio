import * as React from "react"
import { SubTitle } from "../text/text.component"
import Box from "../ui-box/index"

type Props = {
  text: string
}

export const DataEmpty: React.SFC<Props> = ({ text, children }) => (
  <Box px={[4]} py={[4]} align="center" justify="center" flexDirection="column" w={["100%"]}>
    <SubTitle color="link" fontWeight={["normal"]} fontSize={[3]}>
      {text}
    </SubTitle>
    {children}
  </Box>
)
