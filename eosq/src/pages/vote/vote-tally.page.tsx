import { t } from "i18next"
import { observer } from "mobx-react"
import * as React from "react"
import { Banner } from "../../components/banner/banner.component"
import { ListVotedProducers } from "../../components/voted-producers/list-voted-producers.component"
import { Vote } from "../../models/vote"
import { metricsStore, voteStore } from "../../stores"
import { PageContainer } from "../../components/page-container/page-container"
import { registerVoteTallyStream } from "../../streams/vote-listener"
import { DataError, Box } from "@dfuse/explorer"
import { Stream, ErrorData } from "@dfuse/client"

interface State {
  fetchError: boolean
}

@observer
export class VoteTally extends React.Component<any, State> {
  state: State = { fetchError: false }
  voteTallyStream?: Stream

  async registerStreams() {
    this.setState({ fetchError: false })
    this.voteTallyStream = await registerVoteTallyStream((error: ErrorData) => {
      this.setState({ fetchError: true })
    })
  }

  async unregisterStreams() {
    if (this.voteTallyStream) {
      await this.voteTallyStream.close()
      this.voteTallyStream = undefined
    }
  }

  async componentDidMount() {
    await this.registerStreams()
  }

  async componentWillUnmount() {
    await this.unregisterStreams()
  }

  renderEmpty() {
    return <Box>{t("vote.list.empty")}</Box>
  }

  renderTable(votes: Vote[], headBlockProducer: string) {
    return <ListVotedProducers headBlockProducer={headBlockProducer} votes={votes || []} />
  }

  render() {
    const { votes } = voteStore
    const { headBlockProducer } = metricsStore
    const isEmpty = votes.length <= 0

    if (this.state.fetchError) {
      return (
        <PageContainer>
          <DataError />
        </PageContainer>
      )
    }

    return (
      <PageContainer>
        <Banner />
        {isEmpty ? this.renderEmpty() : this.renderTable(votes, headBlockProducer)}
      </PageContainer>
    )
  }
}
