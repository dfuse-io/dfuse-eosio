import * as React from "react"
import { styled } from "../../theme"
import { Text } from "../text/text.component"

const StyledBigButton: React.ComponentType<any> = styled.button`
  background-color: #dfe2e6;
  border: none;
  border-radius: 17px;
  cursor: pointer;

  &:hover {
    background-color: ${(props) => {
      console.log(props)
      return props.theme.colors.linkHover
    }};
  }
`

const StyledText: React.ComponentType<any> = styled(Text)`
  color: ${(props) => props.theme.colors.link};

  &:hover {
    color: white;
  }
`

type BigButtonProps = {
  text: string
  onClick: () => void
}

export const BigButton: React.SFC<BigButtonProps> = ({ children, text, onClick }) => (
  <StyledBigButton onClick={onClick}>
    <StyledText py={[2]}>{text}</StyledText>
  </StyledBigButton>
)
