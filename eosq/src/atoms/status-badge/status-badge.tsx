import * as React from "react"
import { Cell } from "../ui-grid/ui-grid.component"
import { theme, styled } from "../../theme"
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome"
import { faCheck, faBan, faClock } from "@fortawesome/free-solid-svg-icons"

const Container: React.ComponentType<any> = styled(Cell)`
  width: 70px;
  height: 70px;
  position: relative;
  padding: 3px;
`

const CircleContainer: React.ComponentType<any> = styled(Cell)`
  width: 64px;
  height: 64px;
  border-radius: 32px;
`

const CircleBorder: React.ComponentType<any> = styled(Cell)`
  border: 8px solid ${(props: any) => props.borderColor};
  border-radius: 35px;
  width: 70px;
  height: 70px;
  position: absolute;
  top: 0;
  left: 0;
`

const FontContainer: React.ComponentType<any> = styled(Cell)`
  position: absolute;
  top: 21px;
  left: 21px;
`

const BanContainer: React.ComponentType<any> = styled(Cell)`
  position: absolute;
  top: 0px;
  left: 0px;
`

export enum StatusBadgeVariant {
  BAN = "ban",
  CLOCK = "clock",
  CHECK = "check"
}

export const StatusBadge: React.SFC<{
  variant: StatusBadgeVariant
}> = ({ variant }) => {
  let color = theme.colors.statusBadgeCheck
  let background = theme.colors.statusBadgeCheckBg
  let icon: any = faCheck
  let size: any = "2x"
  let IconContainer = FontContainer

  switch (variant) {
    case StatusBadgeVariant.BAN:
      color = theme.colors.statusBadgeBan
      background = theme.colors.statusBadgeBanBg
      icon = faBan
      size = "5x"
      IconContainer = BanContainer
      break
    case StatusBadgeVariant.CLOCK:
      color = theme.colors.statusBadgeClock
      background = theme.colors.statusBadgeClockBg
      icon = faClock
      break
    case StatusBadgeVariant.CHECK:
      color = theme.colors.statusBadgeCheck
      background = theme.colors.statusBadgeCheckBg
      icon = faCheck
      break
    default:
      break
  }

  return (
    <Container>
      <CircleContainer bg={background} />
      {variant === "ban" ? null : <CircleBorder borderColor={color} />}
      <IconContainer>
        <FontAwesomeIcon size={size} icon={icon} color={color} />
      </IconContainer>
    </Container>
  )
}
