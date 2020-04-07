import { BlockStore } from "./block-store"
import { MetricsStore } from "./metrics-store"
import { VoteStore } from "./voter-store"
import { ServiceWorkerStore } from "./service-worker-store"
import { SearchStore } from "./search-store"
import { MenuStore } from "./menu-store"
import { ContractTableStore } from "./contract-table-store"
import { TemplateStore } from "./template-store"
import { ALL_TEMPLATES } from "../components/action-pills/templates/all-templates"

export const blockStore = new BlockStore()
export const metricsStore = new MetricsStore()
export const voteStore = new VoteStore()
export const serviceWorkerStore = new ServiceWorkerStore()
export const searchStore = new SearchStore()
export const menuStore = new MenuStore()
export const contractTableStore = new ContractTableStore()
export const templateStore = new TemplateStore(ALL_TEMPLATES)
