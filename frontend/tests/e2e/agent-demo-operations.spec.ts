import { expect, test } from 'bun:test'
import { tmpdir } from 'node:os'
import { dirname, join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { spawn, spawnSync } from 'bun'
import {
  clickButtonByText,
  clickEnabledButtonByText,
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

type DemoState = { public_key: string; read_scope: string; write_scope: string }
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

function fieldValue(view: Bun.WebView, label: string): Promise<string> {
  return view.evaluate(`(() => {
    const node = [...document.querySelectorAll('label')].find((item) => item.textContent?.trim() === ${JSON.stringify(label)})
    const box = node?.parentElement
    return box?.querySelector('code')?.textContent?.trim() ?? box?.querySelector('input')?.value ?? ''
  })()`) as Promise<string>
}

// 行の必要なスコープを一つだけ残す。新しい行は登録済みのスコープをすべて選んだ状態で始まる。
async function keepOnlyScope(view: Bun.WebView, row: number, scope: string) {
  await view.evaluate(`(() => {
    // 入力欄の親が項目、その親が一つのツールの行である。
    const box = document.querySelector('#tool-name-${row}').parentElement.parentElement
    for (const label of box.querySelectorAll('fieldset label')) {
      const input = label.querySelector('input[type=checkbox]')
      if (input.checked !== (label.textContent.trim() === ${JSON.stringify(scope)})) input.click()
    }
  })()`)
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
      `[...document.querySelector('#resource-connection').options].find(option=>option.textContent==='社内文書 MCP')?.value`,
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

//spec:covers REQ-OAUTH2-084, REQ-OAUTH2-083, REQ-MCPGATEWAY-010: デモで閲覧、更新、削除を個別に割り当て、同じスコープの未割当の削除を拒否する。
test('administrator enables writing and deletion independently in the agent demo', async () => {
  // go run は子プロセスを残すので、kill が本体へ届くように先にビルドして直接起動する。
  const binary = join(tmpdir(), `idmagic-agent-demo-operations-${process.pid}`)
  const built = spawnSync(['go', 'build', '-o', binary, './examples/agent-demo'], { cwd: root })
  expect(built.stderr.toString()).toBe('')
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

    // Gateway の評価用サービスを登録する。
    await navigateAndLogin(
      view,
      '/realms/default/admin/applications/new',
      'admin-application-create',
    )
    await setInputValue(view, '#app-name', `文書ゲートウェイ-${Date.now()}`)
    await view.evaluate(
      `[...document.querySelectorAll('button')].find((node) => node.textContent?.includes('Service (M2M)'))?.click()`,
    )
    await clickButtonByText(view, 'Create')
    await waitForText(view, 'Client secret')
    const gatewayClientID = await fieldValue(view, 'Client ID')
    const gatewaySecret = await fieldValue(view, 'Client secret')

    // 三つのツールを登録する。閲覧は tools:read、更新と削除は同じ tools:write に対応させる。
    await view.navigate(`${uiOrigin}/realms/default/admin/mcp-resource-servers/new`)
    await setInputValue(view, '#resource', resource)
    await setInputValue(view, '#name', '社内文書 MCP')
    await setInputValue(view, '#scopes', `${state.read_scope} ${state.write_scope}`)
    await setInputValue(view, '#client-id', gatewayClientID)
    const tools = [
      ['read_document', '文書の閲覧', state.read_scope],
      ['write_document', '文書の作成・更新', state.write_scope],
      ['delete_document', '文書の削除', state.write_scope],
    ] as const
    for (const [index, [tool, label, scope]] of tools.entries()) {
      await clickButtonByText(view, 'Register tool')
      await setInputValue(view, `#tool-name-${index}`, tool)
      await setInputValue(view, `#tool-label-${index}`, label)
      await keepOnlyScope(view, index, scope)
    }
    await view.evaluate(`document.querySelector('form button[type="submit"]').click()`)
    await waitForPage(view, 'admin-mcp-resource-servers')

    // Agent と認証用クライアントを作る。認証上限は登録済みの二つのスコープになる。
    const name = `社内文書アシスタント-${Date.now()}`
    await view.navigate(`${uiOrigin}/realms/default/admin/agents/new`)
    await setInputValue(view, '#agent-name', name)
    await clickButtonByText(view, 'Register')
    await waitForPage(view, 'admin-agent-detail')
    const agentPath = new URL(view.url).pathname
    await view.navigate(`${uiOrigin}${agentPath}/credentials`)
    await setInputValue(view, '#agent-public-key', state.public_key)
    await clickEnabledButtonByText(view, 'Create and bind')
    await waitForText(view, 'The credential has been added.')
    const agentClientID = String(
      await view.evaluate(`document.querySelector('li code')?.textContent ?? ''`),
    )
    expect(agentClientID).not.toBe('')

    // 閲覧だけを割り当てて接続する。スコープを省略した発行は閲覧に絞られる。
    await toggleOperation(view, agentPath, '文書の閲覧')
    const connected = await post('/api/connect', {
      catalog: true,
      agent_client_id: agentClientID,
      gateway_client_id: gatewayClientID,
      gateway_client_secret: gatewaySecret,
    })
    expect(connected.steps[0]).toMatchObject({ outcome: 'allowed' })
    expect(connected.steps[0]?.evidence).toBe(`発行されたスコープ: ${state.read_scope}`)
    expect(await last('read-public')).toMatchObject({ outcome: 'allowed', executed: true })
    expect(await last('read-secret')).toMatchObject({ outcome: 'allowed', executed: true })
    expect(await last('write')).toMatchObject({ outcome: 'denied', executed: false })
    expect(await last('delete')).toMatchObject({ outcome: 'denied', executed: false })

    // 更新を追加し、新しいスコープのためにトークンを取り直す。同じスコープの削除は拒否のまま。
    await toggleOperation(view, agentPath, '文書の作成・更新')
    const reissued = await last('reissue')
    expect(reissued).toMatchObject({ outcome: 'allowed' })
    expect(reissued?.evidence).toContain(state.write_scope)
    expect(await last('write')).toMatchObject({ outcome: 'allowed', executed: true })
    expect(await last('delete')).toMatchObject({ outcome: 'denied', executed: false })

    // 削除の追加は同じスコープ内なので、取り直さずに次の依頼から効く。
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
