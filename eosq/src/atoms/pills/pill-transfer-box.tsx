import * as React from "react"
import Box from "../ui-box/index"
import { ArrowTo, MonospaceTextLink } from "../text-elements/misc"
import { NBSP } from "../../helpers/formatters"
import { faArrowRight } from "@fortawesome/free-solid-svg-icons"
import { Links } from "../../routes"
import { EllipsisText, Text } from "../text/text.component"

export interface TransferProps {
  from: string
  to: string
  amount?: string
  amounts?: { type: string; value: string }[]
  context?: string
  memo?: string
}

export const TransferBox: React.SFC<TransferProps> = ({
  context,
  from,
  to,
  amount,
  amounts,
  memo,
}): JSX.Element => {
  function renderAmount(amountName: { type: string; value: string }, index: number) {
    return (
      <Box key={index} mx={[2]} minHeight="26px" minWidth="10px" alignItems="center">
        <Text fontSize={[1]} fontWeight="bold">
          {amountName.type}:
        </Text>
        {NBSP}
        <Text fontSize={[1]} whiteSpace="nowrap">
          {amountName.value}
        </Text>
      </Box>
    )
  }

  function renderSimpleAmount(amountName: string) {
    return (
      <Box key="0" alignItems="center">
        <Text
          lineHeight="1em"
          whiteSpace="nowrap"
          key={0}
          ml={[2]}
          fontSize={[1]}
          fontWeight="bold"
        >
          {amountName}
        </Text>
      </Box>
    )
  }

  function renderAmounts(
    amountName?: string,
    amountNames?: { type: string; value: string }[]
  ): JSX.Element[] {
    if (amountName) {
      return [renderSimpleAmount(amountName)]
    }

    if (amountNames) {
      return amountNames.map((entry: { type: string; value: string }, index: number) =>
        renderAmount(entry, index)
      ) as JSX.Element[]
    }

    return []
  }

  if (!from || !to) {
    console.warn("Transfer pill cannot be display, empty content:")
    console.log("from:", from)
    console.log("to:", to)
    return <div />
  }

  return (
    <Box mx={[2]} minHeight="26px" minWidth="10px" alignItems="center">
      <MonospaceTextLink
        fontWeight={from === context ? "bold" : "normal"}
        fontSize={[1]}
        to={Links.viewAccount({ id: from.toString() })}
      >
        {from}
      </MonospaceTextLink>
      <ArrowTo icon={faArrowRight} />
      <MonospaceTextLink
        fontWeight={to === context ? "bold" : "normal"}
        fontSize={[1]}
        to={Links.viewAccount({ id: to.toString() })}
      >
        {to}
      </MonospaceTextLink>
      {renderAmounts(amount, amounts)}
      <Box alignItems="center" minWidth="10px" fontSize={[1]} pl={[2]} color="neutral">
        <EllipsisText lineHeight="1em" fontSize={[1]}>
          {memo}
        </EllipsisText>
      </Box>
    </Box>
  )
}
