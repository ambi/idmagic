// 1段目 preload: Happy DOM をグローバルへ登録する。ESM の import 巻き上げにより、
// DOM を必要とする他モジュール（jest-dom/RTL）より必ず先に評価される必要があるため、
// setup.ts とはファイルを分離している。
import { GlobalRegistrator } from '@happy-dom/global-registrator'

GlobalRegistrator.register({ url: 'http://localhost:3000' })

// React 19 は act() 境界外の更新を警告するが、テスト環境では RTL がラップするため抑止する。
;(globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true

// Happy DOM の Location はアクセサを prototype 側に持ち own-enumerable プロパティを持たない
// （`Object.keys(window.location)` が `[]` になる）。このため `{...window.location}` は
// pathname/origin 等すべてを失い、それを読むアプリコードがサイレントに壊れる。
// テストコードは `vi.stubGlobal('location', { ...originalLocation, assign: vi.fn() })`
// のようにスプレッド前提で書かれているため、スプレッド可能な列挙可能プロパティを持つ
// スナップショットに差し替える。
const original = window.location
const snapshot = {
  href: original.href,
  origin: original.origin,
  protocol: original.protocol,
  host: original.host,
  hostname: original.hostname,
  port: original.port,
  pathname: original.pathname,
  search: original.search,
  hash: original.hash,
  assign: original.assign.bind(original),
  replace: original.replace.bind(original),
  reload: original.reload.bind(original),
  toString: () => original.toString(),
}

Object.defineProperty(window, 'location', {
  value: snapshot,
  writable: true,
  configurable: true,
})
