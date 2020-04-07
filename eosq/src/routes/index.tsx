import { compile } from "path-to-regexp"

export const Paths = {
  home: "/",
  about: "/about",
  vote: "/vote",
  viewAccountTabs: "/account/:id/:currentTab",
  contact: "/contact",

  searchResults: "/search-results",

  serverError: "/server-error",
  notFound: "/not-found",
  viewAccount: "/account/:id",
  viewBlock: "/block/:id",
  transactions: "/transactions",
  blocks: "/blocks",
  viewTransaction: "/tx/:id",
  producers: "/producers",
  viewTransactionSearch: "/search"
}

export const Links = {
  blocks: compile(Paths.blocks),
  home: compile(Paths.home),
  about: compile(Paths.about),
  vote: compile(Paths.vote),
  viewAccountTabs: compile(Paths.viewAccountTabs),
  searchResults: compile(Paths.searchResults),
  contact: compile(Paths.contact),
  // Inject equivalent here;..

  serverError: compile(Paths.serverError),
  notFound: compile(Paths.notFound),

  viewAccount: compile(Paths.viewAccount),
  viewBlock: compile(Paths.viewBlock),

  transactions: compile(Paths.transactions),
  viewTransaction: compile(Paths.viewTransaction),
  viewTransactionSearch: compile(Paths.viewTransactionSearch)
}
