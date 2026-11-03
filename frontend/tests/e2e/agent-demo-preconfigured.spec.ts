import { expect, test } from 'bun:test'
import { tmpdir } from 'node:os'
import { dirname, join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { spawn, spawnSync } from 'bun'
import {
  clickButtonByText,
  clickLinkByText,
  demo as demoAccount,
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
const demoOrigin = 'http://127.0.0.1:5187'
// ほかのデモの例と同じテナントへ登録するため、接続先の URI を分ける。
const resource = 'https://docs-mcp-preconfigured.agent-demo.example/'

type Step = { title: string; outcome: string; executed: boolean }

async function last(scenario: string): Promise<Step | undefined> {
  const response = await fetch(`${demoOrigin}/api/run`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ scenario }),
  })
  expect(response.status).toBe(200)
  return ((await response.json()) as { steps: Step[] }).steps.at(-1)
}

//spec:covers REQ-OAUTH2-088, REQ-OAUTH2-087, REQ-IDMANAGEMENT-105, REQ-AUTHORIZATION-017, REQ-OAUTH2-083: 管理者のログインだけで設定済みの開始点を作り、作ったものと出所を示し、文書ごとに絞ると未指定の文書を止める。
test('administrator starts the agent demo preconfigured and narrows it to one document', async () => {
  // go run は子プロセスを残すので、kill が本体へ届くように先にビルドして直接起動する。
  const binary = join(tmpdir(), `idmagic-agent-demo-preconfigured-${process.pid}`)
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
      '127.0.0.1:5187',
      '-resource',
      resource,
    ],
    { cwd: root, stdout: 'ignore', stderr: 'ignore' },
  )
  const view = await openWebView({ width: 1440, height: 2000 })
  try {
    await waitForUp(`${demoOrigin}/api/state`)
    await view.navigate(`${demoOrigin}/`)
    await waitForText(view, '設定済みで始める')

    // 自分で設定する場合の量を固定する。増えたら見直す。
    expect(await view.evaluate(`({ ...document.querySelector('#setup-counts').dataset })`)).toEqual(
      { steps: '4', decided: '3', copied: '3' },
    )

    // 管理者として一度ログインすると、デモが同じ管理 API で一式を作って接続する。
    await setInputValue(view, '#admin-username', demoAccount.username)
    await setInputValue(view, '#admin-password', demoAccount.password)
    await clickButtonByText(view, 'ログインして設定する')
    // 本文の文字列はスクリプトにも含まれるので、一覧の要素が現れるのを待つ。
    for (let attempt = 0; attempt < 120; attempt++) {
      if (
        await view.evaluate(
          `Boolean(document.querySelector('#prepared-items, #prepared-error .error'))`,
        )
      )
        break
      await Bun.sleep(500)
    }
    expect(
      await view.evaluate(`document.querySelector('#prepared-error .error')?.textContent ?? ''`),
    ).toBe('')
    const prepared = String(
      await view.evaluate(`document.querySelector('#prepared-items').innerText`),
    )
    for (const expected of [
      '評価用のクライアント',
      'ゲートウェイが報告',
      'IdMagic が作成',
      '文書の閲覧',
    ]) {
      expect(prepared).toContain(expected)
    }
    expect(await view.evaluate(`document.querySelector('#admin-password').value`)).toBe('')

    expect(await last('read-public')).toMatchObject({ outcome: 'allowed', executed: true })
    expect(await last('read-secret')).toMatchObject({ outcome: 'allowed', executed: true })
    expect(await last('delete')).toMatchObject({ outcome: 'denied', executed: false })

    // 二度目の設定は作り直さずに断る。
    await clickButtonByText(view, 'ログインして設定する')
    for (let attempt = 0; attempt < 40; attempt++) {
      if (await view.evaluate(`Boolean(document.querySelector('#prepared-error .error'))`)) break
      await Bun.sleep(250)
    }
    expect(
      await view.evaluate(`document.querySelector('#prepared-error .error')?.textContent ?? ''`),
    ).toContain('すでに設定済み')

    // 文書ごとに絞る：閲覧のツールを対象ごとにし、エージェントの範囲を就業規則だけにする。
    // 準備した設定の「管理画面で開く」から、作った接続先の ID を読む。
    const agentResourcesURL = String(
      await view.evaluate(
        `[...document.querySelectorAll("#prepared-items a")].map((node) => node.href).find((href) => href.endsWith("/resources"))`,
      ),
    )
    const serverID = String(
      await view.evaluate(
        `[...document.querySelectorAll('#prepared-items a')].map((node) => node.href).find((href) => href.endsWith('/edit')).split('/').at(-2)`,
      ),
    )
    await navigateAndLogin(
      view,
      '/realms/default/admin/mcp-resource-servers',
      'admin-mcp-resource-servers',
    )
    await view.navigate(`${uiOrigin}/realms/default/admin/mcp-resource-servers/${serverID}/edit`)
    await waitForPage(view, 'admin-mcp-resource-server-edit')
    const readRow = Number(
      await view.evaluate(
        `[...document.querySelectorAll('[id^="tool-name-"]')].findIndex((node) => node.value === 'read_document')`,
      ),
    )
    await view.evaluate(
      `document.querySelector('#tool-name-${readRow}').parentElement.parentElement.querySelector('input[type=checkbox]').click()`,
    )
    await clickButtonByText(view, 'Update')
    await waitForPage(view, 'admin-mcp-resource-servers')
    expect(await last('read-public')).toMatchObject({ outcome: 'denied', executed: false })

    await view.navigate(agentResourcesURL)
    await waitForPage(view, 'admin-agent-resources')
    await clickLinkByText(view, 'Edit connections and operations')
    await waitForPage(view, 'admin-agent-resources-edit')
    await setSelectValue(view, '#resource-connection', serverID)
    await view.evaluate(
      `[...document.querySelectorAll('input[type=radio]')].find((node) => node.parentElement.textContent.trim() === 'Only the listed targets').click()`,
    )
    await view.evaluate(
      `(() => { const area = document.querySelector('textarea[id^="targets-"]'); const set = Object.getOwnPropertyDescriptor(HTMLTextAreaElement.prototype, 'value').set; set.call(area, 'handbook'); area.dispatchEvent(new Event('input', { bubbles: true })) })()`,
    )
    await clickButtonByText(view, 'Save')
    await waitForPage(view, 'admin-agent-detail')

    expect(await last('read-public')).toMatchObject({ outcome: 'allowed', executed: true })
    expect(await last('read-secret')).toMatchObject({ outcome: 'denied', executed: false })
  } finally {
    await view.close?.()
    demo.kill()
  }
}, 240_000)
