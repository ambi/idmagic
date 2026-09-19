// REQ-SYSTEM-021: ゲートウェイ越しに、実行時の経路表が持つ経路へ実際に届くことを確かめる。
//
// backend/cmd/idmagic-gateway-routes は設定の字面と経路表を突き合わせる。突き合わせが
// 通っても、設定を読むゲートウェイが実際にその経路を中継するとは限らない。ここは開発用
// ゲートウェイ (Vite 開発サーバー) を実際に通し、応答が API のものであって SPA の
// `index.html` ではないことを観測する。
//
// このずれは静かに失敗する。外れた経路は 404 ではなく `index.html` の 200 になるので、
// 状態番号ではなく本文が届いたかどうかの印になる。
//
// ホスト形式は `localhost` をテナントへ解決できないので、API は `tenant_not_found` を
// 返す。それも API の応答なので、到達の証拠としては足りる。経路そのものの応答まで
// 確かめられるのはパス形式であり、両方の形式を見るのは `@realmBackend` と `@backend` が
// 別の列挙だからである。
import { expect, test } from 'bun:test'
import { uiOrigin } from './fixtures'

// SPA の `index.html` だけが持つ印。Vite 開発サーバーの fallback はこの本文を返す。
const spaMarker = '<div id="root"></div>'
// ホスト形式がテナントを解決できなかったときに API が返す拒否。
const hostTenantRefusal = '"error":"tenant_not_found"'

async function read(path: string, init?: RequestInit) {
  const response = await fetch(`${uiOrigin}${path}`, { redirect: 'manual', ...init })
  return { status: response.status, body: await response.text() }
}

function postSecurityEvent(path: string) {
  return read(path, {
    method: 'POST',
    headers: { 'Content-Type': 'application/secevent+jwt' },
    body: 'not-a-set',
  })
}

test('gateway proxies the runtime routes the SPA fallback would otherwise swallow', async () => {
  // Shared Signals の受信経路はこれ 1 本しかない。存在しないストリームへの POST に
  // RFC 8935 §2.3 が定める拒否の本体が返ることが、要求が API まで届いた証拠である。
  const missingStream = 'streams/00000000-0000-4000-8000-000000000000/events'
  const hostFormEvent = await postSecurityEvent(`/ssf/${missingStream}`)
  expect(hostFormEvent.body).not.toContain(spaMarker)
  expect(hostFormEvent.body).toContain(hostTenantRefusal)

  const realmFormEvent = await postSecurityEvent(`/realms/default/ssf/${missingStream}`)
  expect(realmFormEvent.body).not.toContain(spaMarker)
  expect(realmFormEvent.body).toMatch(/"err":"(ssf_stream_not_found|security_event_rejected)"/)

  // OIDC Session Management の check_session_iframe。RP が hidden iframe で読み込む。
  // ここも HTML を返すので、SPA と区別できる印は本文の中にしかない。
  const hostFormIframe = await read('/session/check')
  expect(hostFormIframe.body).not.toContain(spaMarker)
  expect(hostFormIframe.body).toContain(hostTenantRefusal)

  const iframe = await read('/realms/default/session/check')
  expect(iframe.status).toBe(200)
  expect(iframe.body).toContain('check_session_iframe')
  expect(iframe.body).not.toContain(spaMarker)

  // 管理コンソールがアプリケーションアイコンを組み立てる URL。存在しない ID なので
  // API は 404 を返す。中継されていなければ SPA の 200 になる。
  const missingIcon = '00000000-0000-4000-8000-000000000000/missing'
  const hostFormIcon = await read(`/application-icons/${missingIcon}`)
  expect(hostFormIcon.body).not.toContain(spaMarker)
  expect(hostFormIcon.body).toContain(hostTenantRefusal)

  const icon = await read(`/realms/default/application-icons/${missingIcon}`)
  expect(icon.body).not.toContain(spaMarker)
  expect(icon.status).toBe(404)
}, 60_000)

test('gateway keeps the unauthenticated metrics endpoint off the public entry point', async () => {
  // `/metrics` は認証を持たない。docs/domain/system/decisions.md が公開先を限っている
  // ので、ゲートウェイは中継してはならない。中継していなければ SPA へ落ちる。
  const received = await read('/metrics')
  expect(received.body).toContain(spaMarker)
  expect(received.body).not.toContain('go_goroutines')
}, 30_000)
