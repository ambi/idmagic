// 2段目 preload: DOM 登録後に読み込まれる jest-dom matchers と RTL cleanup。
// bun:test には Vitest の global `afterEach` 自動登録が無いため、明示的に登録する。
import { afterEach } from 'bun:test'
import { cleanup } from '@testing-library/react'
import '@testing-library/jest-dom'

// 画面シェルは表示時にブランド設定の公開エンドポイントを読む。各テストが意図していない実通信を
// localhost へ送らないよう、既定値だけをテスト環境で返す。それ以外の通信は個別テストが
// 明示的に差し替えるか、未差し替えなら従来どおり失敗させる。
const networkFetch = globalThis.fetch
globalThis.fetch = Object.assign(
  (input: RequestInfo | URL, init?: RequestInit) => {
    const requestURL = new URL(input instanceof Request ? input.url : input, window.location.href)
    const method = init?.method ?? (input instanceof Request ? input.method : 'GET')
    if (requestURL.pathname === '/api/branding' && method === 'GET') {
      return Promise.resolve(
        new Response('{}', { status: 200, headers: { 'content-type': 'application/json' } }),
      )
    }
    // 委譲先は Bun 本来の fetch なので、文書を基準に相対 URL を解決しない。画面コードが
    // `/api/...` を渡す経路を保つため、ここで解決済みの絶対 URL を渡す。Request は自身が
    // 絶対 URL を持つのでそのまま通す。
    return networkFetch(input instanceof Request ? input : requestURL, init)
  },
  { preconnect: networkFetch.preconnect },
)

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
