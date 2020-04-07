import { t } from "i18next"
import { observer } from "mobx-react"
import * as React from "react"
import { MonospaceTextLink, WrappingText } from "../../../atoms/text-elements/misc"
import { DetailLine } from "../../../atoms/pills/detail-line"
import {
  formatBytes,
  formatDateTime,
  formatMicroseconds,
  formatNumber,
  formatTransactionID,
  LONGDASH,
  NBSP,
  secondsToTime,
} from "../../../helpers/formatters"
import { ExtDTrxOp, Authorization } from "@dfuse/client"
import {
  computeTransactionTrustPercentage,
  TransactionReceiptStatus,
} from "../../../models/transaction"
import { Links } from "../../../routes"
import { LinkStyledText, Text, TextLink } from "../../../atoms/text/text.component"
import { Cell, Grid } from "../../../atoms/ui-grid/ui-grid.component"
import { BlockProgressPie } from "../../blocks/block-progress-pie"
import { metricsStore } from "../../../stores"
import { formatDateFromString } from "../../../helpers/moment.helpers"
import { Age } from "../../../atoms/age/age.component"
import { StatusBadge } from "../../../atoms/status-badge/status-badge"
import {
  getStatusBadgeVariant,
  getTransactionStatusColor,
} from "../../../helpers/transaction.helpers"
import { translate, Trans } from "react-i18next"
import { JsonWrapper } from "../../../atoms/json-wrapper/json-wrapper"
import { UiHrDotted } from "../../../atoms/ui-hr/ui-hr"
import { UiModal } from "../../../atoms/ui-modal/ui-modal"
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome"
import { faPlusCircle } from "@fortawesome/free-solid-svg-icons"
import { TransactionLifecycleWrap } from "../../../services/transaction-lifecycle"
import { AutorizationBox } from "../../../components/authorization-box/authorization-box.component"
import { SearchShortcut } from "../../../components/search-shortcut/search-shortcut"

interface Props {
  gridRow?: any
  lifecycleWrap: TransactionLifecycleWrap
}

@observer
class BaseTransactionDetailHeader extends React.Component<Props> {
  get isStale() {
    return !this.props.lifecycleWrap.lifecycle.execution_irreversible && this.trustPercentage >= 1
  }

  get hasRecentMetrics() {
    return (
      metricsStore.lastIrreversibleBlockNum > 0 &&
      metricsStore.headBlockNum > this.props.lifecycleWrap.blockNum
    )
  }

  get trustPercentage() {
    if (metricsStore.lastIrreversibleBlockNum <= 0) {
      return 0.0
    }
    return computeTransactionTrustPercentage(
      this.props.lifecycleWrap.blockNum,
      metricsStore.headBlockNum,
      metricsStore.lastIrreversibleBlockNum
    )
  }

  get blockHeader() {
    return this.props.lifecycleWrap.lifecycle.execution_block_header
  }

  extractException(): any | undefined {
    const { status, executionTrace } = this.props.lifecycleWrap
    if (!executionTrace) return undefined
    if (
      status !== TransactionReceiptStatus.SOFT_FAIL &&
      status !== TransactionReceiptStatus.HARD_FAIL
    ) {
      return undefined
    }

    if (executionTrace.except) {
      return executionTrace.except
    }

    return undefined
  }

  renderTransactionStatusValue(): React.ReactChild {
    const { status } = this.props.lifecycleWrap
    const exception = this.extractException()
    const color = getTransactionStatusColor(status)

    return (
      <Cell>
        <Text color={color} display="inline-block" fontWeight="bold">
          {t(`transaction.status.${status}`)}
        </Text>
        {this.renderIrreversibleText()}
        {exception ? this.renderException(exception) : null}
      </Cell>
    )
  }

  renderIrreversibleText() {
    if (!this.props.lifecycleWrap.lifecycle.execution_irreversible && this.trustPercentage >= 1) {
      return null
    }

    if (this.trustPercentage >= 1) {
      return null
    }

    if (this.props.lifecycleWrap.blockNum && this.props.lifecycleWrap.blockNum > 0) {
      if (!this.hasRecentMetrics) {
        return null
      }

      return (
        <Text ml={[2]} display="inline-block" fontWeight="normal">
          {t("transaction.detailPanel.statuses.accepted")} (
          <Text display="inline-block" color="secondHighlight">
            {metricsStore.headBlockNum - this.props.lifecycleWrap.blockNum || 0}
            {t("transaction.detailPanel.statuses.blockDeep")}
          </Text>
          )
        </Text>
      )
    }

    return null
  }

  renderUsageValue(value: number | undefined, formatter: (value: any) => string) {
    return this.renderValue(value, (rawValue: number) => {
      if (rawValue <= 0) return t("transaction.detailPanel.noUsage")

      return formatter(rawValue)
    })
  }

  renderExpirationDate(expirationDate?: Date) {
    return this.renderValue(expirationDate, formatDateTime)
  }

  renderCpuUsage(cpuUsage?: number) {
    return this.renderUsageValue(cpuUsage, formatMicroseconds)
  }

  renderNetworkUsage(networkUsage?: number) {
    return this.renderUsageValue(networkUsage || -1, formatBytes)
  }

  renderAuthorizations() {
    if (this.props.lifecycleWrap.executionTrace) {
      const { authorizations } = this.props.lifecycleWrap
      return (
        <Text>
          {authorizations.map((auth: Authorization, index: number) => {
            return this.renderAuthorization(auth, index === authorizations.length - 1)
          })}
        </Text>
      )
    }

    return null
  }

  renderAuthorization(auth: Authorization, isLast: boolean) {
    return (
      <Cell display="inline-block" key={auth.actor} mr={[2]}>
        <AutorizationBox authorization={auth} />
      </Cell>
    )
  }

  renderSignedBy() {
    return (this.props.lifecycleWrap.lifecycle.pub_keys || []).map(
      (publicKey: string, index: number) => {
        const query = `(data.auth.keys.key:${publicKey} OR data.active.keys.key:${publicKey} OR data.owner.keys.key:${publicKey})`
        return (
          <WrappingText key={index}>
            <SearchShortcut fixed={true} query={query}>
              {publicKey}
            </SearchShortcut>
          </WrappingText>
        )
      }
    )
  }

  renderValue(value: any | undefined, formatter: (value: any) => string) {
    if (value === undefined) {
      return ""
    }

    return <WrappingText>{formatter(value)}</WrappingText>
  }

  renderBlockValue(): React.ReactChild {
    if (this.props.lifecycleWrap.noBlockInfo) {
      return <Text> {t("transaction.detailPanel.producer.unknown")}</Text>
    }
    return (
      <TextLink to={Links.viewBlock({ id: this.props.lifecycleWrap.blockId })}>
        {this.props.lifecycleWrap.blockNum > 0
          ? formatNumber(this.props.lifecycleWrap.blockNum)
          : t("transaction.detailPanel.producer.unknown")}
      </TextLink>
    )
  }

  renderBlockStatusValue = (): JSX.Element => {
    if (!this.props.lifecycleWrap.lifecycle.execution_irreversible && this.trustPercentage >= 1) {
      return <Text color="secondHighlight">{t("block.stale")}</Text>
    }

    if (this.trustPercentage >= 1) {
      return (
        <Text color="ternary" fontWeight="bold">
          {t("block.irreversible")}
        </Text>
      )
    }
    return <Text color="secondHighlight">{t("block.reversible")}</Text>
  }

  renderProducerValue(): React.ReactChild {
    if (!this.blockHeader || this.blockHeader.producer === "") {
      return <span />
    }

    return (
      <DetailLine compact={true} label={t("transaction.blockPanel.producer")}>
        <MonospaceTextLink to={Links.viewAccount({ id: this.blockHeader!.producer })}>
          {this.blockHeader!.producer}
        </MonospaceTextLink>
      </DetailLine>
    )
  }

  renderTimeStamp(timestamp: string) {
    if (!timestamp || timestamp === "") {
      return null
    }

    return (
      <div title={formatDateFromString(timestamp, true)}>
        {formatDateFromString(timestamp, false)}
      </div>
    )
  }

  renderBlockDetail() {
    if (this.props.lifecycleWrap.blockId) {
      return (
        <Cell pt={[2]}>
          <DetailLine compact={true} label={t("transaction.blockPanel.block")}>
            {this.renderBlockValue()}
          </DetailLine>
          {this.props.lifecycleWrap.blockTimestamp ? (
            <DetailLine compact={true} label={t("transaction.blockPanel.age")}>
              <Age date={this.props.lifecycleWrap.blockTimestamp} />
            </DetailLine>
          ) : null}
          <DetailLine compact={true} label={t("transaction.blockPanel.blockId")}>
            <TextLink
              to={
                this.props.lifecycleWrap.blockId
                  ? Links.viewBlock({ id: this.props.lifecycleWrap.blockId })
                  : ""
              }
            >
              {this.props.lifecycleWrap.blockId}
            </TextLink>
          </DetailLine>
          <DetailLine compact={true} label={t("transaction.blockPanel.status")}>
            {this.renderBlockStatusValue()}
          </DetailLine>
          {this.renderProducerValue()}
        </Cell>
      )
    }

    return null
  }

  renderDeferredTemplate(ref: string, i18nKey: string) {
    const refObject: ExtDTrxOp = this.props.lifecycleWrap.lifecycle[ref]

    const i18nKeyLabel = `transaction.deferred.${i18nKey}.label`
    let i18nKeyContent = `transaction.deferred.${i18nKey}.content`

    if (refObject) {
      if (
        refObject.src_trx_id === this.props.lifecycleWrap.lifecycle.id &&
        refObject.op !== "PUSH_CREATE"
      ) {
        return null
      }

      if (i18nKey === "creationMethod") {
        i18nKeyContent = `transaction.deferred.${i18nKey}.${refObject.op}`
      }

      return (
        <DetailLine key={`${ref}-${i18nKey}`} compact={true} label={t(i18nKeyLabel)}>
          <Trans
            i18nKey={i18nKeyContent}
            values={{
              transactionId: formatTransactionID(refObject.src_trx_id).join(""),
              blockNum: formatNumber(refObject.block_num),
            }}
            components={[
              <TextLink key="1" to={Links.viewTransaction({ id: refObject.src_trx_id })}>
                {formatTransactionID(refObject.src_trx_id).join("")}
              </TextLink>,
              <TextLink key="2" to={Links.viewBlock({ id: refObject.block_id })}>
                {formatNumber(refObject.block_num)}
              </TextLink>,
            ]}
          />
        </DetailLine>
      )
    }
    return null
  }

  renderDelayedFor() {
    if (
      this.props.lifecycleWrap.transaction &&
      this.props.lifecycleWrap.transaction.delay_sec > 0
    ) {
      return (
        <DetailLine key="0" compact={true} label={t("transaction.deferred.delayedFor")}>
          {secondsToTime(this.props.lifecycleWrap.lifecycle.transaction!.delay_sec)}
        </DetailLine>
      )
    }
    return null
  }

  renderDeferredInfo() {
    return [
      this.renderDelayedFor(),
      this.renderDeferredTemplate("created_by", "createdBy"),
      this.renderDeferredTemplate("canceled_by", "canceledBy"),
      this.renderDeferredTemplate("created_by", "creationMethod"),
    ]
  }

  renderException(except: any) {
    const message = except.message as string

    if (message.length > 0) {
      return [
        <Cell key="0" display="inline-block">
          :{NBSP}
          {message}
        </Cell>,
        <UiModal
          key="1"
          opener={
            <span>
              {NBSP}
              {LONGDASH}
              {NBSP}
              <LinkStyledText color="link">
                {t("transaction.detailPanel.fullTrace")}{" "}
                <FontAwesomeIcon icon={faPlusCircle as any} />
              </LinkStyledText>
            </span>
          }
        >
          <Cell>
            <JsonWrapper>{JSON.stringify(except, null, " ")}</JsonWrapper>
          </Cell>
        </UiModal>,
      ]
    }
    return null
  }

  renderFailureTraceDetail() {
    if (
      !this.props.lifecycleWrap.executionTrace ||
      !this.props.lifecycleWrap.executionTrace.failed_dtrx_trace
    ) {
      return null
    }

    const i18nKeyLabel = `transaction.deferred.triggeredBy.label`
    const i18nKeyContent = `transaction.deferred.triggeredBy.content`

    const failureTrace = this.props.lifecycleWrap.executionTrace.failed_dtrx_trace

    return (
      <DetailLine key="failure-trace" compact={true} label={t(i18nKeyLabel)}>
        <Trans
          i18nKey={i18nKeyContent}
          values={{
            transactionId: formatTransactionID(failureTrace.id).join(""),
            blockNum: formatNumber(failureTrace.block_num),
          }}
          components={[
            <TextLink key="1" to={Links.viewTransaction({ id: failureTrace.id })}>
              {formatTransactionID(failureTrace.id).join("")}
            </TextLink>,
            <TextLink key="2" to={Links.viewBlock({ id: failureTrace.producer_block_id })}>
              {formatNumber(failureTrace.block_num)}
            </TextLink>,
          ]}
        />
      </DetailLine>
    )
  }

  renderExecutionDetails(): (JSX.Element | null)[] {
    const { executionTrace } = this.props.lifecycleWrap
    if (!executionTrace) {
      return []
    }

    return [
      this.renderFailureTraceDetail(),

      <DetailLine key="1" compact={true} label={t("transaction.detailPanel.cpuUsage")}>
        {this.renderCpuUsage(executionTrace.receipt && executionTrace.receipt.cpu_usage_us)}
      </DetailLine>,

      <DetailLine key="2" compact={true} label={t("transaction.detailPanel.networkUsage")}>
        {this.renderNetworkUsage(
          executionTrace.receipt && executionTrace.receipt.net_usage_words * 8
        )}
      </DetailLine>,
    ]
  }

  renderTransactionDetail() {
    return (
      <Grid gridTemplateColumns={["1fr", "1fr"]}>
        <Cell pt={[2]} wordBreak="break-all">
          <DetailLine compact={true} label={t("transaction.detailPanel.status")}>
            {this.renderTransactionStatusValue()}
          </DetailLine>

          {this.renderDeferredInfo()}
          {this.renderExecutionDetails()}
          <DetailLine compact={true} label={t("transaction.detailPanel.authorizations")}>
            {this.renderAuthorizations()}
          </DetailLine>
          <DetailLine compact={true} label={t("transaction.detailPanel.signedBy")}>
            {this.renderSignedBy()}
          </DetailLine>
        </Cell>
      </Grid>
    )
  }

  renderStatusBadge() {
    const variant = getStatusBadgeVariant(this.props.lifecycleWrap.status)
    return variant ? <StatusBadge variant={variant} /> : null
  }

  renderBlockProgressPie() {
    return (
      <Grid maxWidth={["150px", "none"]} mx={["auto", 0]} pb={[1, 2]}>
        <BlockProgressPie
          headBlockNum={metricsStore.headBlockNum}
          blockNum={this.props.lifecycleWrap.blockNum}
          lastIrreversibleBlockNum={metricsStore.lastIrreversibleBlockNum}
        />
      </Grid>
    )
  }

  renderBlockHeader() {
    return [
      <Cell key="0" px={[3, 4]}>
        <UiHrDotted />
      </Cell>,
      <Cell key="1">
        <Grid px={[3, 4]} gridTemplateColumns={["1fr", "8fr 100px"]}>
          <Cell pt={[1, 2]} pb={[2, 3]} gridRow={[2, 1]}>
            {this.renderBlockDetail()}
          </Cell>

          <Grid mt={[4, 0]}>
            {this.props.lifecycleWrap.blockNum > 0 && !this.isStale && this.hasRecentMetrics
              ? this.renderBlockProgressPie()
              : null}
          </Grid>
        </Grid>
      </Cell>,
    ]
  }

  render() {
    return (
      <Grid>
        <Grid px={[3, 4]} pt={[2, 3]} pb={[1, 2]} gridTemplateColumns={["1fr", "8fr 100px"]}>
          <Cell gridRow={[2, 1]}>{this.renderTransactionDetail()}</Cell>
          <Cell p="15px" mt={[4, 4]}>
            <Cell maxWidth={["70px", "none"]} mx={["auto", 0]} pb={[1, 2]}>
              {this.renderStatusBadge()}
            </Cell>
          </Cell>
        </Grid>
        {this.props.lifecycleWrap.lifecycle.execution_block_header
          ? this.renderBlockHeader()
          : null}
      </Grid>
    )
  }
}

export const TransactionDetailHeader = translate()(BaseTransactionDetailHeader)
