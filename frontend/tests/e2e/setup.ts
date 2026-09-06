// E2E のスタックは実行全体で 1 回だけ起動する。
//
// spec ごとに別プロセスへ分けていたときは、Go のビルド、API、Vite 開発サーバー、初期データの
// 投入を spec の数だけやり直していた。この起動はおよそ 5.5 秒あり、テストが 1 本しかない spec の
// 実測 5.6 秒はほぼ全部がそれだった。preload はテストファイルより前に 1 回だけ評価されるので、
// ここで宣言した beforeAll / afterAll が実行全体を包む。
//
// 分けるのをやめると、spec 間の状態が独立しているという暗黙の保証も無くなる。各 spec は自分が
// 操作する対象を自分で用意し、共有フィクスチャ (`demo`) を書き換えないこと。
import { afterAll, beforeAll } from 'bun:test'
import { startE2EEnvironment, stopE2EEnvironment } from './fixtures'

beforeAll(async () => {
  await startE2EEnvironment()
}, 180_000)

afterAll(async () => {
  await stopE2EEnvironment()
}, 30_000)
