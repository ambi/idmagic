import { expect, test } from 'bun:test'
import { tmpdir } from 'node:os'
import { dirname, join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { spawn, spawnSync } from 'bun'
import {
  clickButtonByText,
  clickElementByAriaLabel,
  clickLinkByText,
  navigateAndLogin,
  openWebView,
  setInputValue,
  setSelectValue,
  uiOrigin,
  waitForPage,
  waitForText,
  waitForUp,
} from './fixtures'

const root = resolve(dirname(fileURLToPath(import.meta.url)), '../../..')
const demoOrigin = 'http://127.0.0.1:5186'
// 文書別の例と同じテナントへ登録するため、接続先の URI を分ける。
const resource = 'https://docs-mcp-operations.agent-demo.example/'
// 全 spec が同じテナントを共有し、デモの事前設定も同じ表示名で登録するので、表示名で選ぶ接続先を実行ごとに一意にする。
const connectionName = `社内文書 MCP ${Date.now()}`

type DemoState = { public_key: string }
type Step = { title: string; outcome: string; executed: boolean; evidence?: string }

async function post(path: string, body: unknown): Promise<{ steps: Step[] }> {
  const response = await fetch(`${demoOrigin}${path}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  const result = (await response.json()) as { steps: Step[] }
  expect(response.status, JSON.stringify(result)).toBe(200)
  return result
}

async function last(scenario: string): Promise<Step | undefined> {
  return (await post('/api/run', { scenario })).steps.at(-1)
}

async function toggleOperation(view: Bun.WebView, agentPath: string, operation: string) {
  await view.navigate(`${uiOrigin}${agentPath}`)
  await waitForPage(view, 'admin-agent-detail')
  await clickLinkByText(view, 'Resource connections')
  await waitForPage(view, 'admin-agent-resources')
  await clickLinkByText(view, 'Edit connections and operations')
  await waitForPage(view, 'admin-agent-resources-edit')
  const serverID = String(
    await view.evaluate(
      `[...document.querySelector('#resource-connection').options].find(option=>option.textContent===${JSON.stringify(connectionName)})?.value`,
    ),
  )
  await setSelectValue(view, '#resource-connection', serverID)
  await view.evaluate(
    `[...document.querySelectorAll('label')].find(node=>node.querySelector('.font-medium')?.textContent===${JSON.stringify(operation)})?.querySelector('input[type=checkbox]')?.click()`,
  )
  await waitForText(view, `${operation}`)
  await clickButtonByText(view, 'Save')
  await waitForPage(view, 'admin-agent-detail')
  await waitForText(view, 'Connection and operation assignments saved.')
}

//spec:covers REQ-IDMANAGEMENT-105, REQ-OAUTH2-088, REQ-OAUTH2-087, REQ-MCPGATEWAY-011, REQ-OAUTH2-084, REQ-OAUTH2-083, REQ-MCPGATEWAY-010: 一つの画面で接続先と評価用クライアントを登録し、ゲートウェイが報告したツールを名前の入力なしに公開して、閲覧、更新、削除を個別に割り当てる。同じスコープでも未割当の削除は拒否する。
test('administrator enables writing and deletion independently in the agent demo', async () => {
  // go run は子プロセスを残すので、kill が本体へ届くように先にビルドして直接起動する。
  const binary = join(tmpdir(), `idmagic-agent-demo-operations-${process.pid}`)
  const built = spawnSync(['go', 'build', '-o', binary, './examples/agent-demo'], { cwd: root })
  if (built.exitCode !== 0) {
    throw new Error(`go build failed: ${built.stderr.toString()}`)
  }
  const demo = spawn(
    [
      binary,
      '-realm',
      `${uiOrigin}/realms/default`,
      '-listen',
      '127.0.0.1:5186',
      '-resource',
      resource,
    ],
    { cwd: root, stdout: 'ignore', stderr: 'ignore' },
  )
  const view = await openWebView({ width: 1440, height: 2000 })
  try {
    await waitForUp(`${demoOrigin}/api/state`)
    const state = (await (await fetch(`${demoOrigin}/api/state`)).json()) as DemoState

    // 接続先を登録する。評価用のクライアントは同じ送信で作られ、client_id もスコープも入力しない。
    await navigateAndLogin(
      view,
      '/realms/default/admin/mcp-resource-servers/new',
      'admin-mcp-resource-server-create',
    )
    await setInputValue(view, '#resource', resource)
    await setInputValue(view, '#name', connectionName)
    expect(await view.evaluate(`document.querySelector('#evaluator-client').value`)).toBe('__new__')
    await view.evaluate(`document.querySelector('form button[type="submit"]').click()`)
    await waitForText(view, 'Registered')
    const gatewayClientID = String(
      await view.evaluate(`document.querySelector('#result-client-id').textContent`),
    )
    const gatewaySecret = String(
      await view.evaluate(`document.querySelector('#result-client-secret').textContent`),
    )
    expect(
      await view.evaluate(`document.querySelector('#result-gateway-config').textContent`),
    ).toContain('catalog: true')

    // デモのゲートウェイを起動すると、実在のゲートウェイが上流の三つのツールを報告する。
    const started = await post('/api/gateway', {
      catalog: true,
      gateway_client_id: gatewayClientID,
      gateway_client_secret: gatewaySecret,
    })
    expect(started.steps.at(-1)).toMatchObject({ outcome: 'allowed' })

    // 報告されたツールを名前の入力なしで公開する。表示名は MCP の title から入る。
    await clickLinkByText(view, 'Publish reported tools')
    await waitForPage(view, 'admin-mcp-resource-server-edit')
    await waitForText(view, '文書の削除')
    for (const tool of ['read_document', 'write_document', 'delete_document']) {
      await clickElementByAriaLabel(view, `Publish ${tool}`)
    }
    expect(
      await view.evaluate(
        `[...document.querySelectorAll('[id^="tool-label-"]')].map((node) => node.value)`,
      ),
    ).toEqual(['文書の閲覧', '文書の作成・更新', '文書の削除'])
    await clickButtonByText(view, 'Update')
    await waitForPage(view, 'admin-mcp-resource-servers')

    // 一つの登録画面で Agent、公開鍵の認証、閲覧の割当を作る。Agent ID は控えない。
    const name = `社内文書アシスタント-${Date.now()}`
    await view.navigate(`${uiOrigin}/realms/default/admin/agents/new`)
    await waitForPage(view, 'admin-agent-create')
    await setInputValue(view, '#agent-name', name)
    await setInputValue(view, '#agent-create-public-key', state.public_key)
    await view.evaluate(
      `[...document.querySelectorAll('label')].find(node=>node.textContent.trim()==='文書の閲覧')?.querySelector('input[type=checkbox]')?.click()`,
    )
    await waitForText(view, 'Allow 文書の閲覧.')
    await clickButtonByText(view, 'Register')
    await waitForText(view, 'Registered')
    const agentClientID = String(
      await view.evaluate(
        `document.querySelector('#result-agent-client-id code')?.textContent ?? ''`,
      ),
    )
    expect(agentClientID).not.toBe('')
    expect(
      await view.evaluate(`document.querySelector('#result-agent-config').textContent`),
    ).toContain(resource)
    await clickLinkByText(view, 'Open agent details')
    await waitForPage(view, 'admin-agent-detail')
    const agentPath = new URL(view.url).pathname

    // 閲覧だけを割り当てた状態で接続する。スコープを省略した発行は IdMagic が決める。
    const connected = await post('/api/connect', { catalog: true, agent_client_id: agentClientID })
    expect(connected.steps.at(-1)).toMatchObject({ outcome: 'allowed' })
    expect(connected.steps.at(-1)?.evidence).toBe('発行されたスコープ: mcp')
    expect(await last('read-public')).toMatchObject({ outcome: 'allowed', executed: true })
    expect(await last('read-secret')).toMatchObject({ outcome: 'allowed', executed: true })
    expect(await last('write')).toMatchObject({ outcome: 'denied', executed: false })
    expect(await last('delete')).toMatchObject({ outcome: 'denied', executed: false })

    // 更新を追加すると取り直さずに更新できる。同じスコープでも未割当の削除は拒否のまま。
    await toggleOperation(view, agentPath, '文書の作成・更新')
    expect(await last('write')).toMatchObject({ outcome: 'allowed', executed: true })
    expect(await last('delete')).toMatchObject({ outcome: 'denied', executed: false })

    // 削除の追加も次の依頼から効く。
    await toggleOperation(view, agentPath, '文書の削除')
    expect(await last('delete')).toMatchObject({ outcome: 'allowed', executed: true })

    // 削除の取消も手元のトークンの次の依頼に効き、更新の割当は残る。
    await toggleOperation(view, agentPath, '文書の削除')
    expect(await last('delete')).toMatchObject({ outcome: 'denied', executed: false })
    expect(await last('write')).toMatchObject({ outcome: 'allowed', executed: true })
  } finally {
    await view.close?.()
    demo.kill()
  }
}, 240_000)
