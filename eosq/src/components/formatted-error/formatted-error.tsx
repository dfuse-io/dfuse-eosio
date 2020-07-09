import { Cell, Grid } from "../../atoms/ui-grid/ui-grid.component"
import { Text } from "../../atoms/text/text.component"
import * as React from "react"
import { faExclamationTriangle } from "@fortawesome/free-solid-svg-icons"
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome"
import { theme, styled } from "../../theme"
import { JsonWrapper } from "@dfuse/explorer"
import { ErrorData } from "@dfuse/client"

const BoldText: React.ComponentType<any> = styled(Text)`
  font-weight: bold;

  font-family: "Roboto Condensed", sans-serif;
  margin-bottom: 3px;
`

interface Props {
  error: ErrorData
  title: string
}

const ErrorWrapper = styled(Cell)`
  background-color: ${(props) => props.theme.colors.statusBadgeClockBg};
  border: 1px solid ${(props) => props.theme.colors.statusBadgeClock};
  margin: 20px;
  word-break: break-word;
`

export const FormattedError: React.FC<Props> = ({ title, error }) => (
  <ErrorWrapper px={[4]} py={[4]} alignItems="center">
    <Grid gridTemplateColumns={["50px auto"]}>
      <Cell fontSize={[4]} alignSelf="center">
        <FontAwesomeIcon icon={faExclamationTriangle} color={theme.colors.statusBadgeClock} />
      </Cell>

      <Cell>
        <BoldText display="inline-block" fontSize={[4]} color={theme.colors.statusBadgeClock}>
          {title}
        </BoldText>
      </Cell>
    </Grid>
    <Cell />
    <br />
    <Grid gridTemplateColumns={["50px auto"]}>
      <Cell />
      <Grid wordBreak="break-word" gridTemplateRows="auto">
        <BoldText fontSize={[3]}>Message</BoldText>
        <Text fontSize={[3]} mb={[3]}>
          {error.message}
        </Text>
        <BoldText fontSize={[3]}>Code</BoldText>
        <Text fontSize={[3]} mb={[3]}>
          {error.code}
        </Text>
        {error.trace_id
          ? [
              <BoldText key="trace-id-label" fontSize={[3]}>
                Trace_id
              </BoldText>,
              <Text key="trace-id-value" fontSize={[3]} mb={[3]}>
                {error.trace_id}
              </Text>
            ]
          : null}

        <BoldText fontSize={[3]}>Details</BoldText>

        <Text fontSize={[3]}>
          <JsonWrapper fontSize={[2]}>{JSON.stringify(error.details, null, 2)}</JsonWrapper>
        </Text>
      </Grid>
    </Grid>
  </ErrorWrapper>
)
