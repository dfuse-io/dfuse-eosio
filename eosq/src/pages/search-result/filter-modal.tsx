import * as React from "react"
import { UiModal } from "../../atoms/ui-modal/ui-modal"
import { LinkStyledText, Text } from "../../atoms/text/text.component"
import { t } from "i18next"
import { Cell } from "../../atoms/ui-grid/ui-grid.component"
import { HoverableIcon } from "../../atoms/hoverable/hoverable"
import { faCog } from "@fortawesome/free-solid-svg-icons"
import { theme } from "../../theme"
import { SearchFilters } from "../../components/search-filters/search-filters"
import { observer } from "mobx-react"
import { RouteComponentProps } from "react-router"

interface State {
  filtersOpened: boolean
}

interface Props extends RouteComponentProps<any> {
  title: string
  color?: string
}

@observer
export class FilterModal extends React.Component<Props, State> {
  state = { filtersOpened: false }

  onApplyFilters = () => {
    this.setState({ filtersOpened: false })
  }

  onOpenFilters = () => {
    this.setState({ filtersOpened: true })
  }

  render() {
    return (
      <UiModal
        opened={this.state.filtersOpened}
        onOpen={this.onOpenFilters}
        opener={
          <LinkStyledText
            display="inline-block"
            color={this.props.color ? this.props.color : "link"}
            fontSize={[3]}
            fontFamily="'Roboto Condensed', sans-serif;"
          >
            {this.props.title}
            <Cell display="inline-block" ml={[2]}>
              <HoverableIcon icon={faCog as any} size="lg" />
            </Cell>
          </LinkStyledText>
        }
        headerTitle={
          <Text fontSize={[4]} p={[2]} color={theme.colors.bleu8} fontWeight="600">
            {t("filters.queryParams")}
          </Text>
        }
      >
        <SearchFilters onApply={this.onApplyFilters} {...this.props} />
      </UiModal>
    )
  }
}
