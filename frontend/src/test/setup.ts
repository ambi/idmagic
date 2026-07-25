// 2段目 preload: DOM 登録後に読み込まれる jest-dom matchers と RTL cleanup。
// bun:test には Vitest の global `afterEach` 自動登録が無いため、明示的に登録する。
import { afterEach } from 'bun:test'
import { cleanup } from '@testing-library/react'
import '@testing-library/jest-dom'

afterEach(() => {
  cleanup()
})

// jsdom の Blob/File は text() を実装しないため、FileReader 経由で補う。
if (typeof File !== 'undefined' && typeof File.prototype.text !== 'function') {
  File.prototype.text = function (this: File) {
    return new Promise<string>((resolve, reject) => {
      const reader = new FileReader()
      reader.onload = () => resolve(String(reader.result))
      reader.onerror = () => reject(reader.error)
      reader.readAsText(this)
    })
  }
}
