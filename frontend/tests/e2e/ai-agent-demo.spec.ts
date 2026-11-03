import { expect, test } from 'bun:test'
import { tmpdir } from 'node:os'
import { dirname, join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { spawn, spawnSync } from 'bun'
import {
  clickButtonByText,
  clickElementByAriaLabel,
  clickEnabledButtonByText,
  clickLinkByText,
  clickMenuItemByText,
  navigateAndLogin,
  openWebView,
  setInputValue,
  setSelectValue,
  uiOrigin,
  waitForPage,
  waitForText,
  waitForUp,
  waitForUrl,
} from './fixtures'

const root = resolve(dirname(fileURLToPath(import.meta.url)), '../../..')
const demoOrigin = 'http://127.0.0.1:5185'

type DemoState = {
  public_key: string
  resource: string
  scope: string
  read_scope: string
  write_scope: string
  public_document: string
  secret_document: string
}
type Step = { title: string; outcome: string; executed: boolean }

async function run(scenario: string): Promise<Step[]> {
  const response = await fetch(`${demoOrigin}/api/run`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ scenario }),
  })
  const body = (await response.json()) as { steps: Step[] }
  expect(response.status).toBe(200)
  return body.steps
}

async function action(path: string): Promise<unknown> {
  const response = await fetch(`${demoOrigin}${path}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: '{}',
  })
  const body = await response.json()
  expect(response.status, JSON.stringify(body)).toBe(200)
  return body
}

function fieldValue(view: Bun.WebView, label: string): Promise<string> {
  return view.evaluate(`(() => {
    const node = [...document.querySelectorAll('label')].find((item) => item.textContent?.trim() === ${JSON.stringify(label)})
    const box = node?.parentElement
    return box?.querySelector('code')?.textContent?.trim() ?? box?.querySelector('input')?.value ?? ''
  })()`) as Promise<string>
}

async function moveDemoStep(view: Bun.WebView, direction: string, heading: string) {
  await view.evaluate(`(() => {
    const card = document.querySelector('#setup .card:not([hidden])')
    const button = [...card.querySelectorAll('.nav button')].find((node) => node.textContent === ${JSON.stringify(direction)})
    if (!button || button.disabled) throw new Error('The step navigation is unavailable')
    button.click()
  })()`)
  expect(
    await view.evaluate(
      `Array.from(document.querySelectorAll('#setup .card:not([hidden]) h3'), (node) => node.textContent)`,
    ),
  ).toEqual([heading])
}

//spec:covers REQ-IDMANAGEMENT-103, REQ-IDMANAGEMENT-104, REQ-AUTHORIZATION-014: 画面で資格情報と直接付与を設定し、実在のゲートウェイ越しの許可、拒否、停止を観測する。
test('administrator governs a real agent from the console', async () => {
  // go run は子プロセスを残すので、kill が本体へ届くように先にビルドして直接起動する。
  const binary = join(tmpdir(), `idmagic-agent-demo-${process.pid}`)
  const built = spawnSync(['go', 'build', '-o', binary, './examples/agent-demo'], { cwd: root })
  expect(built.stderr.toString()).toBe('')
  const demo = spawn(
    [binary, '-realm', `${uiOrigin}/realms/default`, '-listen', '127.0.0.1:5185'],
    { cwd: root, stdout: 'ignore', stderr: 'ignore' },
  )
  const view = await openWebView({ width: 1440, height: 2000 })
  try {
    await waitForUp(`${demoOrigin}/api/state`)
    const state = (await (await fetch(`${demoOrigin}/api/state`)).json()) as DemoState

    // 統制なしでは、機密の読み取りも削除も上流で実行される。
    const unmanaged = await run('unmanaged')
    expect(unmanaged.map((item) => item.executed)).toEqual([true, true, true])

    // 手順 1：テンプレートから認可モデルを発行する。JSON は使わない。
    await navigateAndLogin(
      view,
      '/realms/default/admin/authorization/model/edit',
      'admin-authorization-model-edit',
    )
    await clickButtonByText(view, 'Template: document viewers')
    await clickEnabledButtonByText(view, 'Review publication')
    await waitForText(view, 'Differences from the latest version')
    await clickEnabledButtonByText(view, 'Publish')
    await waitForPage(view, 'admin-authorization-model')

    // 手順 2：ゲートウェイのサービスを登録する。スコープは空のまま。
    await view.navigate(`${uiOrigin}/realms/default/admin/applications/new`)
    await setInputValue(view, '#app-name', `文書ゲートウェイ-${Date.now()}`)
    await view.evaluate(
      `[...document.querySelectorAll('button')].find((node) => node.textContent?.includes('Service (M2M)'))?.click()`,
    )
    await clickButtonByText(view, 'Create')
    await waitForText(view, 'Client secret')
    const gatewayClientID = await fieldValue(view, 'Client ID')
    const gatewaySecret = await fieldValue(view, 'Client secret')
    expect(gatewayClientID).not.toBe('')
    expect(gatewaySecret).not.toBe('')

    // 作成したサービスの詳細を開ける（セッション切れの誤表示の回帰）。
    await view.navigate(`${uiOrigin}/realms/default/admin/applications`)
    await waitForPage(view, 'admin-applications')
    const serviceID = String(
      await view.evaluate(`fetch('/realms/default/api/admin/v1/applications?limit=200')
        .then((response) => response.json())
        .then((body) => body.applications.find((app) => app.protocol?.client_id === ${JSON.stringify(gatewayClientID)})?.id)`),
    )
    await view.navigate(`${uiOrigin}/realms/default/admin/applications/${serviceID}`)
    await waitForPage(view, 'admin-application-detail')

    // 手順 3：保護する MCP サーバーを行の入力で登録する。
    await view.navigate(`${uiOrigin}/realms/default/admin/mcp-resource-servers/new`)
    await setInputValue(view, '#resource', state.resource)
    await setInputValue(view, '#name', '社内文書 MCP')
    await setInputValue(view, '#scopes', state.read_scope)
    await setInputValue(view, '#client-id', gatewayClientID)
    await view.evaluate(
      `[...document.querySelectorAll('summary')].find(node=>node.textContent==='Add a legacy evaluation rule')?.click()`,
    )
    await clickButtonByText(view, 'Add a legacy evaluation rule')
    await setInputValue(view, '#rule-type-0', 'document')
    await setInputValue(view, '#rule-relation-0', 'viewer')
    await view.evaluate(`document.querySelector('form button[type="submit"]').click()`)
    await waitForUrl(view, /\/admin\/mcp-resource-servers$/)

    // 手順 4：エージェントを登録する。
    const name = `社内文書アシスタント-${Date.now()}`
    await view.navigate(`${uiOrigin}/realms/default/admin/agents/new`)
    await setInputValue(view, '#agent-name', name)
    await clickButtonByText(view, 'Register')
    await waitForPage(view, 'admin-agent-detail')
    const detailURL = view.url
    const agentID = decodeURIComponent(new URL(detailURL).pathname.split('/').pop() ?? '')

    // 手順 5：PEM の公開鍵を貼り、登録済みの認証上限を使ってクレデンシャルを作る。
    await view.navigate(`${detailURL}/credentials`)
    await setInputValue(view, '#agent-public-key', state.public_key)
    await clickEnabledButtonByText(view, 'Create and bind')
    await waitForText(view, 'The credential has been added.')
    const agentClientID = String(
      await view.evaluate(`document.querySelector('li code')?.textContent ?? ''`),
    )
    expect(agentClientID).not.toBe('')

    // 手順 6：対象と Agent の選択で付与する。ID は入力しない。
    await view.navigate(detailURL)
    await waitForPage(view, 'admin-agent-detail')
    await clickLinkByText(view, 'Resource connections')
    await waitForPage(view, 'admin-agent-resources')
    await clickLinkByText(view, 'Resource assignments and diagnostics')
    await waitForPage(view, 'admin-agent-access')
    await clickLinkByText(view, 'Add access')
    await waitForPage(view, 'admin-agent-access-create')
    const serverID = String(
      await view.evaluate(
        `Array.from(document.querySelector('#access-connection').options).find(option => option.text === '社内文書 MCP')?.value`,
      ),
    )
    await setSelectValue(view, '#access-connection', serverID)
    await setSelectValue(view, '#access-permission', 'document:viewer')
    await setInputValue(view, '#access-resource', state.public_document)
    expect(await view.evaluate(`document.querySelector('#access-agent').value`)).toBe(agentID)
    await clickEnabledButtonByText(view, 'Grant access')
    await waitForPage(view, 'admin-agent-access')
    await waitForText(view, 'Access assignment added.')
    await waitForText(view, state.public_document)

    // 手順 7：デモの画面から接続し、画面の結果を確かめる。
    await view.navigate(`${demoOrigin}/?example=documents`)
    await waitForText(view, '手順 1')
    await clickButtonByText(view, '統制なしで 3 つの依頼を実行')
    await waitForText(view, '給与テーブル', 15_000)
    await moveDemoStep(view, '次へ', '手順 2　ゲートウェイ')
    await setInputValue(view, '#gateway-client', gatewayClientID)
    await setInputValue(view, '#gateway-secret', gatewaySecret)
    await moveDemoStep(view, '次へ', '手順 3　MCP サーバー')
    expect(
      await view.evaluate(`document.querySelector('#setup .card:not([hidden])').innerText`),
    ).toContain(gatewayClientID)
    await moveDemoStep(view, '次へ', '手順 4　エージェント')
    await setInputValue(view, '#agent-id', agentID)
    await moveDemoStep(view, '次へ', '手順 5　クレデンシャル')
    await setInputValue(view, '#agent-client', agentClientID)
    await moveDemoStep(view, '次へ', '手順 6　閲覧の許可')
    expect(
      await view.evaluate(`document.querySelector('#setup .card:not([hidden])').innerText`),
    ).toContain('エージェントを名前で選んで')
    await moveDemoStep(view, '前へ', '手順 5　クレデンシャル')
    expect(await view.evaluate(`document.querySelector('#agent-client').value`)).toBe(agentClientID)
    await clickButtonByText(view, '7. 接続')
    expect(await view.evaluate(`document.querySelector('#connect').checkVisibility()`)).toBe(true)
    await clickButtonByText(view, '接続して確かめる')
    await waitForText(view, '共有の API キーはどこにもありません')
    await clickButtonByText(view, '給与テーブルを読む')
    await waitForText(view, 'ゲートウェイが止めました')

    const allowed = (await run('read-public')).at(-1)
    expect(allowed).toMatchObject({ outcome: 'allowed', executed: true })
    expect((await run('read-secret')).at(-1)).toMatchObject({ outcome: 'denied', executed: false })
    expect((await run('delete')).at(-1)).toMatchObject({ outcome: 'denied', executed: false })

    // デモの画面が示す値で関係を足すと、トークンを取り直さずに給与テーブルを読める。
    await waitForText(view, state.secret_document)
    await view.navigate(
      `${uiOrigin}/realms/default/admin/authorization/access/new?agent=${encodeURIComponent(agentID)}`,
    )
    await waitForPage(view, 'admin-agent-access-create')
    await setSelectValue(view, '#access-connection', serverID)
    await setSelectValue(view, '#access-permission', 'document:viewer')
    await setInputValue(view, '#access-resource', state.secret_document)
    await clickEnabledButtonByText(view, 'Grant access')
    await waitForPage(view, 'admin-agent-access')
    await waitForText(view, state.secret_document)
    const granted = await run('read-secret')
    expect(granted).toHaveLength(1)
    expect(granted[0]).toMatchObject({ outcome: 'allowed', executed: true })

    // 取消は給与テーブルだけに効き、就業規則の付与は残る。
    await view.evaluate(`(() => {
      const row = [...document.querySelectorAll('tbody tr')].find(node => node.textContent.includes(${JSON.stringify(state.secret_document)}))
      row.querySelector('button').click()
    })()`)
    await waitForText(view, 'Revoke access')
    await view.evaluate(`document.querySelector('[role="dialog"] button:last-child').click()`)
    await waitForText(view, 'Direct assignment revoked.')
    expect((await run('read-secret')).at(-1)).toMatchObject({ outcome: 'denied', executed: false })
    expect((await run('read-public')).at(-1)).toMatchObject({ outcome: 'allowed', executed: true })

    // 停止は発行済みのトークンと新しい発行の両方に効く。
    await view.navigate(detailURL)
    await waitForPage(view, 'admin-agent-detail')
    await clickElementByAriaLabel(view, 'Agent actions')
    await clickMenuItemByText(view, 'Disable')
    await waitForText(view, 'Disabled')
    const stopped = await run('after-stop')
    expect(stopped[0]).toMatchObject({ outcome: 'denied', executed: false })
    expect(stopped[1]).toMatchObject({ outcome: 'denied' })
    expect(stopped).toHaveLength(2)

    // 有効に戻して監督下へ変えると、人の承認を経て初めてトークンが出る。
    await clickElementByAriaLabel(view, 'Agent actions')
    await clickMenuItemByText(view, 'Enable')
    await waitForText(view, 'Active')
    await clickLinkByText(view, 'Edit basic information')
    await waitForPage(view, 'admin-agent-edit')
    await setSelectValue(view, '#agent-editor-kind', 'supervised')
    await clickButtonByText(view, 'Save')
    await waitForPage(view, 'admin-agent-detail')
    const started = (await action('/api/approval/start')) as { steps: Step[]; interval: number }
    expect(started.steps[0]).toMatchObject({ outcome: 'denied' })
    expect(await action('/api/approval/poll')).toMatchObject({ pending: true })
    await view.navigate(`${uiOrigin}/realms/default/account/approvals`)
    await waitForText(view, 'Approve')
    await clickButtonByText(view, 'Approve')
    await waitForText(view, 'Approved the request')
    await Bun.sleep((started.interval + 1) * 1000)
    const decided = (await action('/api/approval/poll')) as { pending: boolean; steps: Step[] }
    expect(decided.pending).toBe(false)
    expect(decided.steps[0]).toMatchObject({ outcome: 'allowed' })
  } finally {
    await view.close?.()
    demo.kill()
  }
}, 240_000)
