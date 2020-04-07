import { observable } from "mobx"

export enum ServiceWorkerStates {
  DEFAULT = "default",
  REGISTERED = "registered",
  UPDATEFOUND = "update_found",
  INSTALLING = "installing",
  INSTALLED = "installed"
}

export class ServiceWorkerStore {
  @observable state: ServiceWorkerStates

  constructor() {
    this.state = ServiceWorkerStates.DEFAULT
  }

  changeToState(newState: ServiceWorkerStates) {
    console.log("Updating serviceworker state to %s at time %s", newState, Date.now())
    this.state = newState
  }
}
