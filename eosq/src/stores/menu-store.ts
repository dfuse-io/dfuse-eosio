import { observable } from "mobx"

export class MenuStore {
  @observable opened: boolean

  constructor() {
    this.opened = false
  }

  open() {
    this.opened = true
  }

  close() {
    this.opened = false
  }
}
