import * as React from "react"
import { Route, RouteComponentProps, Switch } from "react-router-dom"
import WrappedWithTheme from "../../hocs/with-theme"
import { TransactionDetailPage } from "../../pages/transactions/transaction-detail.page"
import { TransactionsPage } from "../../pages/transactions/transactions.page"
import { VoteTally } from "../../pages/vote/vote-tally.page"
import { Paths } from "../../routes"
import { menuStore, metricsStore, serviceWorkerStore } from "../../stores"
import { Header } from "../header"
import { NotFound } from "../not-found/not-found.component"
import { ServerError } from "../server-error/server-error.component"
import { Cell, Grid } from "../../atoms/ui-grid/ui-grid.component"
import { BlocksPage } from "../../pages/blocks/blocks.page"
import { BlockDetailPage } from "../../pages/blocks/block-detail.page"
import { SearchResultPage } from "../../pages/search-result/search-result.page"
import { LinkStyledText, Text } from "../../atoms/text/text.component"
import { Footer } from "../footer"
import { t } from "i18next"
import { streamPrice } from "../../clients/websocket/eosws"
import { InboundMessage, Stream, HeadInfoData } from "@dfuse/client"
import { AccountDetail } from "../../pages/account/account-detail.page"
import { TransactionSearchResultPage } from "../../pages/search-result/transaction-search-result.page"
import { NBSP } from "../../helpers/formatters"
import { observer } from "mobx-react"
import { theme, styled } from "../../theme"
import { ServiceWorkerStates } from "../../stores/service-worker-store"
import { handleVisibilityChange, VISIBILITYCHANGE } from "../../helpers/focus.helpers"
import { Config, EosqNetwork } from "../../models/config"
import { getDfuseClient } from "../../data/dfuse"

const SkewedCell = styled(Cell)`
  position: relative;
  &:before {
    content: " ";
    position: absolute;
    top: 0;
    left: -18px;
    width: 20px;
    height: 20px;
    z-index: 0;
    background: ${(props) => props.bg};
    border-radius: inherit;
    transform: skew(35deg);
  }

  @media (max-width: 767px) {
    &:before {
      content: "";
      display: none;
    }
  }
`

const NetworkContainer: React.ComponentType<any> = styled(Cell)`
  position: absolute;
  height: 20px;
  width: auto;
  right: 0px;
  top: 5px;
  z-index: 10000;

  @media (max-width: 767px) {
    &:before {
      content: "";
    }

    position: relative;
    top: 0px;
    height: auto;
  }
`

const HeaderWrapper: React.ComponentType<any> = styled(Cell)`
  position: fixed !important;
  z-index: 1000;
  width: 100%;
  top: 0px;

  background: #474793; /* Old browsers */
  background: -moz-linear-gradient(left, #474793 8%, #5e5ec2 93%); /* FF3.6-15 */
  background: -webkit-linear-gradient(left, #474793 8%, #5e5ec2 93%); /* Chrome10-25,Safari5.1-6 */
  background: linear-gradient(
    to right,
    #474793 8%,
    #5e5ec2 93%
  ); /* W3C, IE10+, FF16+, Chrome26+, Opera12+, Safari7+ */
  filter: progid:DXImageTransform.Microsoft.gradient( startColorstr='#474793', endColorstr='#5e5ec2',GradientType=1 ); /* IE6-9 */
`

const PageWrapper: React.ComponentType<any> = styled.div`
  background-color: ${(props) => props.theme.colors.primary};
  min-height: 100vh;
`

const MaintenanceWrapper: React.ComponentType<any> = styled(Grid)`
  background-color: #ffca28;
`

const Maintenance: React.ComponentType<any> = styled(Cell)`
  padding-top: 10px;
  padding-bottom: 10px;
`

interface Props extends RouteComponentProps<any> {
  currentTheme: string
  switchTheme: any
}

interface State {
  height: number
}

function onElementHeightChange(elm: any, callback: any) {
  let lastHeight = elm.clientHeight
  let newHeight
  ;(function run() {
    newHeight = elm.clientHeight
    if (lastHeight !== newHeight) callback(newHeight)
    lastHeight = newHeight

    if (elm.onElementHeightChangeTimer) clearTimeout(elm.onElementHeightChangeTimer)

    elm.onElementHeightChangeTimer = setTimeout(run, 150)
  })()
}

@observer
class AppContainer extends React.Component<Props, State> {
  headerElement: any

  headInfoStream?: Stream
  priceStream?: Stream

  state: State = {
    height: 167
  }

  focusListener = () => {
    this.registerStreams()
  }

  defocusListener = () => {
    this.unregisterStreams()
  }

  unregisterStreams = async () => {
    if (this.headInfoStream) {
      await this.headInfoStream.close()
      this.headInfoStream = undefined
    }

    if (this.priceStream) {
      await this.priceStream.close()
      this.priceStream = undefined
    }
  }

  registerStreams = async () => {
    if (!this.headInfoStream) {
      this.headInfoStream = await getDfuseClient().streamHeadInfo((message: InboundMessage) => {
        if (message.type === "head_info") {
          metricsStore.setBlockHeight(message.data as HeadInfoData)
        }
      })
    }

    if (!this.priceStream) {
      this.priceStream = await streamPrice(getDfuseClient(), (message: InboundMessage<any>) => {
        if ((message.type as string) === "price") {
          metricsStore.setPrice(message.data)
        }
      })
    }
  }

  componentDidUpdate(prevProps: Readonly<Props>, prevState: Readonly<State>, snapshot?: any): void {
    if (prevProps.location.pathname !== this.props.location.pathname) {
      this.changeDocumentTitle()
      if (menuStore.opened) {
        menuStore.close()
      }
    }
  }

  changeDocumentTitle() {
    if (this.isAListPage(this.props.location.pathname)) {
      document.title = "eosq: High-Precision Block Explorer"
    }
  }

  isAListPage(pathName: string) {
    return (
      !pathName.includes("/account/") && !pathName.includes("/tx/") && !pathName.includes("/block/")
    )
  }

  componentDidMount() {
    this.registerStreams()

    document.addEventListener(
      VISIBILITYCHANGE,
      handleVisibilityChange(this.focusListener, this.defocusListener),
      false
    )

    if (this.headerElement && this.headerElement.clientHeight) {
      const height = this.headerElement!.clientHeight

      this.setState({ height })

      onElementHeightChange(this.headerElement!, (newHeight: number) => {
        this.setState({ height: newHeight })
      })
    }
  }

  componentWillUnmount = async () => {
    clearTimeout(this.headerElement!.onElementHeightChangeTimer)
    document.removeEventListener(
      VISIBILITYCHANGE,
      handleVisibilityChange(this.focusListener, this.defocusListener),
      false
    )

    await this.unregisterStreams()
  }

  renderRoutes() {
    return (
      <Switch>
        <Route exact={true} path={Paths.home} component={TransactionsPage} />

        <Route exact={true} path={Paths.blocks} component={BlocksPage} />
        <Route exact={true} path={Paths.viewBlock} component={BlockDetailPage} />
        <Route exact={true} path={Paths.producers} component={VoteTally} />
        <Route exact={false} path={Paths.searchResults} component={SearchResultPage} />
        <Route exact={true} path={Paths.viewAccountTabs} component={AccountDetail} />
        <Route exact={true} path={Paths.viewAccount} component={AccountDetail} />
        <Route exact={true} path={Paths.transactions} component={TransactionsPage} />
        <Route exact={true} path={Paths.viewTransaction} component={TransactionDetailPage} />
        <Route exact={true} path={Paths.notFound} component={NotFound} />
        <Route
          exact={false}
          path={Paths.viewTransactionSearch}
          component={TransactionSearchResultPage}
        />

        <Route exact={true} path={Paths.serverError} component={ServerError} />

        <Route component={NotFound} />
      </Switch>
    )
  }

  renderTestNetWarning() {
    const network = Config.available_networks.find(
      (ref: EosqNetwork) => ref.id === Config.current_network
    )

    if (!network) {
      return null
    }

    const bg = !network.is_test ? theme.colors.bleu11 : theme.colors.testnet

    return (
      <Cell
        width="100%"
        display={[!network.is_test ? "none" : "block", "block"]}
        py={[1, 0]}
        height={["auto", "5px"]}
        bg={bg}
        textAlign="center"
      >
        <NetworkContainer
          onClick={() => menuStore.open()}
          px={[2]}
          justifySelf="left"
          alignSelf="left"
          color={theme.colors.primary}
          bg={bg}
        >
          <SkewedCell px={[3]} justifySelf="left" alignSelf="left" bg={bg}>
            <Text
              zIndex="10"
              fontFamily="Roboto Condensed"
              fontSize={[1]}
              color="primary"
              display="inline-block"
              fontWeight="bold"
            >
              {t(`core.networkOptions.${Config.current_network.replace("-", "_")}`, {
                defaultValue: network ? network.name : Config.current_network
              })}
            </Text>
          </SkewedCell>
        </NetworkContainer>
      </Cell>
    )
  }

  reloadPage() {
    window.location.reload(true)
  }

  renderNewVersionAvailable() {
    return (
      <MaintenanceWrapper pt={[0]} pb={[0]} px={[1, 0]}>
        <Maintenance mx="auto" maxWidth={["1800px"]} px={[2, 3, 4]}>
          <Text zIndex="10" fontSize="14px" color="black" fontWeight="bold" display="inline-block">
            {t("core.newVersionAvailable")} {NBSP}
            <LinkStyledText fontWeight="bold" onClick={this.reloadPage}>
              {t("core.refresh")}
            </LinkStyledText>
          </Text>
        </Maintenance>
      </MaintenanceWrapper>
    )
  }

  renderTitleBar() {
    return (
      <Cell height={`${this.state.height}px`}>
        <HeaderWrapper
          mx="auto"
          ref={(headerElement: any) => {
            this.headerElement = headerElement
          }}
        >
          {serviceWorkerStore.state === ServiceWorkerStates.INSTALLED
            ? this.renderNewVersionAvailable()
            : null}

          {this.renderTestNetWarning()}
          <Cell mx="auto" px={[2, 3, 4]}>
            <Header />
          </Cell>
        </HeaderWrapper>
      </Cell>
    )
  }

  render() {
    return (
      <PageWrapper id="outer-container">
        {this.renderTitleBar()}
        <Cell minHeight="820px">{this.renderRoutes()}</Cell>
        <Footer />
      </PageWrapper>
    )
  }
}

export default WrappedWithTheme(AppContainer)
