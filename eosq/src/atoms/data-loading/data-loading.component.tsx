import * as React from "react"
import { Text } from "../text/text.component"
import Box from "../ui-box/index"
import { styled } from "../../theme"
import { Spinner } from "../spinner/spinner"

const Loader = styled(Spinner)`
  width: 100%;
`

type Props = {
  text?: string
  color?: string
}

const renderText = (text?: string, color?: string) => {
  if (text === undefined) {
    return null
  }

  return (
    <Text
      wordBreak="break-all"
      whiteSapce="normal"
      py={[3]}
      fontSize={[3]}
      color={[color || "#6452b3"]}
    >
      {text}
    </Text>
  )
}

export const DataLoading: React.SFC<Props> = ({ text, color, children }) => (
  <Box
    pt={[4]}
    pb={[4]}
    textAlign="center"
    justify="center"
    flexDirection="column"
    width={["100%"]}
  >
    <Loader name="three-bounce" color={color || "#6452b3"} fadeIn="none" />
    {renderText(text, color)}
    {children}
  </Box>
)
