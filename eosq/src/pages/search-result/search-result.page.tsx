import * as React from "react"
import { observer } from "mobx-react"
import { RouteComponentProps } from "react-router-dom"
import { Panel } from "../../atoms/panel/panel.component"
import queryString from "query-string"
import { styled } from "../../theme"
import { Text } from "../../atoms/text/text.component"
import { fontSize } from "styled-system"
import { Cell } from "../../atoms/ui-grid/ui-grid.component"
import { t } from "i18next"
import { PageContainer } from "../../components/page-container/page-container"

interface Props extends RouteComponentProps<any> {}

const BoldText: React.ComponentType<any> = styled.span`
  font-weight: bold;
  ${fontSize};
`

@observer
export class SearchResultPage extends React.Component<Props, any> {
  renderTitle() {
    const parsed = queryString.parse(this.props.location.search)
    let query: string = ""
    if (typeof parsed.query === "string") {
      query = parsed.query
    }

    return (
      <Text fontSize={[5]}>
        {t("search.result.noResultFoundFor")}{" "}
        <BoldText fontSize={[5]}>{decodeURIComponent(query)}</BoldText>
      </Text>
    )
  }

  render() {
    return (
      <PageContainer>
        <Panel title={this.renderTitle()}>
          <Cell minHeight="700px" />
        </Panel>
      </PageContainer>
    )
  }
}
