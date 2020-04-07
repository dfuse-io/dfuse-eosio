import { Config } from "../models/config"
import {
  GenericPillComponent,
  PillComponentClass
} from "../components/action-pills/templates/generic-pill.component"
import { translate } from "react-i18next"
import { i18n } from "../i18n"
import { Action } from "@dfuse/client"

export class TemplateStore {
  registry: PillComponentClass[] = []

  constructor(networkPills: PillComponentClass[]) {
    this.registry = networkPills.filter(
      (networkPill: PillComponentClass) =>
        networkPill.contextForRendering().networks.includes(Config.current_network) ||
        networkPill.contextForRendering().networks.includes("all")
    )

    this.registry.forEach((networkPill: PillComponentClass) => {
      if (networkPill.i18n) {
        const languageResources = networkPill.i18n()
        Object.keys(languageResources).forEach((lang: string) => {
          i18n.addResourceBundle(
            lang,
            "translations",
            { pillTemplates: languageResources[lang] },
            true
          )
        })
      }
    })
  }

  getComponent(action: Action<any>): PillComponentClass | typeof GenericPillComponent {
    if (!action.data) {
      return GenericPillComponent
    }

    const templatedComponent = this.findTemplatedPillComponent(action)
    if (!templatedComponent) {
      return GenericPillComponent
    }

    if (templatedComponent.contextForRendering().needsTranslate) {
      return translate()(templatedComponent) as PillComponentClass
    }

    return templatedComponent
  }

  /**
   * We want to find the best matching pill class for this action. This means
   * we need to use a two pass algorithm.
   *
   * In the first pass, we try to find the action that specifically matches
   * both the `contract` and the `action` part. Indeed, some action could matches
   * first only the action which is shared by multiple pill. The `contract`
   * being the discriminator, hence when both matches (for example `specialaccnt`
   * and `issue`), it must takes precedences over the `issue` one that matches
   * any contract.
   *
   * Only when the first matching did not find anything, which means there is
   * not pill that matches both the contract and the action, then we try to
   * match the action alone.
   *
   * The action alone case is required for example for token contracts that
   * are not `eosio.token` so that their transfer methods is correctly templated
   * because for those, we only want to check that the action is `transfer` and that
   * the action has all required fields.
   */
  findTemplatedPillComponent(action: Action<any>) {
    let template = this.registry.find(
      (pillClass: PillComponentClass) =>
        this.matchesBothContractAndAction(action, pillClass) &&
        this.dataHasAllRequiredFields(action.data, pillClass.requireFields)
    )

    if (!template) {
      template = this.registry.find(
        (pillClass: PillComponentClass) =>
          this.matchesOnlyAction(action, pillClass) &&
          this.dataHasAllRequiredFields(action.data, pillClass.requireFields)
      )
    }

    return template
  }

  matchesBothContractAndAction(action: Action<any>, pillClass: PillComponentClass) {
    return pillClass
      .contextForRendering()
      .validActions.some(
        (candidate) => candidate.contract === action.account && candidate.action === action.name
      )
  }

  matchesOnlyAction(action: Action<any>, pillClass: PillComponentClass) {
    return pillClass
      .contextForRendering()
      .validActions.some(
        (candidate) => candidate.contract === undefined && candidate.action === action.name
      )
  }

  dataHasAllRequiredFields(data: Record<string, unknown>, fields: string[]) {
    // eslint-disable-next-line no-prototype-builtins
    return fields.every((field: string) => data.hasOwnProperty(field))
  }
}
