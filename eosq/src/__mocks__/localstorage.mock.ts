export class MockStorage implements Storage {
  length = 0

  clearMock = jest.fn()
  getItemMock = jest.fn()
  keyMock = jest.fn()
  removeItemMock = jest.fn()
  setItemMock = jest.fn()

  clear(): void {
    this.clearMock()
  }

  getItem(key: string): string | null {
    return this.getItemMock(key)
  }

  key(index: number): string | null {
    return this.keyMock(index)
  }

  removeItem(key: string): void {
    this.removeItemMock(key)
  }

  setItem(key: string, value: string): void {
    this.setItemMock(key, value)
  }
}
