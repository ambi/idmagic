import { expect, test } from 'bun:test'
import {
  clickElementByAriaLabel,
  clickSelector,
  clickSummaryByText,
  navigateAndLogin,
  openWebView,
  setInputValue,
  setSelectValue,
  waitForPage,
  waitForText,
} from './fixtures'

const submitDisabled = `document.querySelector('form button[type="submit"]').disabled`

//spec:covers REQ-OAUTH2-090: 条件の文字数と承認の不足を送信前に示し、上限ちょうどの多バイトの式と承認、上限を保存後に読み戻せる。
test('administrator requires approval for a tool from the rule row', async () => {
  const view = await openWebView({ width: 1440, height: 1800 })
  try {
    const suffix = String(Date.now())
    const connection = `Approvals-${suffix}`
    await navigateAndLogin(
      view,
      '/realms/default/admin/mcp-resource-servers/new',
      'admin-mcp-resource-server-create',
    )
    await setInputValue(view, '#resource', `https://approvals-${suffix}.example/mcp`)
    await setInputValue(view, '#name', connection)
    await setSelectValue(view, '#evaluator-client', '')
    // 行に入力欄のある項目も JSON で入れられる。行の編集がそれらを保つことを確かめる。
    await clickSummaryByText(view, 'Edit conditions and limits as JSON')
    await setInputValue(
      view,
      '#access-rules',
      JSON.stringify([
        {
          resource_type: 'idmagic_mcp_operation',
          relation: 'invoke',
          scopes: ['mcp'],
          tool: { name: 'delete_document', display_name: 'Delete documents' },
          condition: { deny_when: 'operation.name == "never"' },
          limits: [{ by: ['agent'], max_requests: 7, window_seconds: 60 }],
        },
      ]),
    )
    // 値のある制約は開いて示す。
    expect(await view.evaluate(`document.querySelector('#rule-limit-max-0-0').value`)).toBe('7')

    const expression = `operation.name == '${'𠮷'.repeat(1024 - "operation.name == ''".length)}'`
    for (const selector of ['#rule-deny-when-0', '#rule-require-approval-when-0']) {
      await setInputValue(view, selector, `${expression} `)
      await waitForText(view, 'Enter an expression of 1024 characters or fewer.')
      expect(
        await view.evaluate(`document.querySelector('${selector}').getAttribute('aria-invalid')`),
      ).toBe('true')
      expect(
        await view.evaluate(`(() => {
          const input = document.querySelector('${selector}')
          return input.getAttribute('aria-describedby').split(' ')
            .map(id => document.getElementById(id)?.textContent)
        })()`),
      ).toContain('Enter an expression of 1024 characters or fewer.')
      expect(await view.evaluate(submitDisabled)).toBe(true)
      expect(
        await view.evaluate(`document.querySelector('${selector}').hasAttribute('maxlength')`),
      ).toBe(false)
      await waitForText(view, '1025 / 1024')
      await setInputValue(view, selector, selector.includes('require-approval') ? '' : expression)
    }
    expect(await view.evaluate(submitDisabled)).toBe(false)
    await waitForText(view, '1024 / 1024')

    // 制約の見出しから実キーで承認の欄へ進み、チェックを切り替える。
    await view.evaluate(
      `document.querySelector('#rule-approval-0').closest('details').querySelector('summary').focus()`,
    )
    await view.press('\t')
    await Bun.sleep(40)
    expect(await view.evaluate(`document.activeElement?.id`)).toBe('rule-approval-0')
    await view.press(' ')
    await waitForText(view, 'Enter an authorization detail type when human approval is required.')
    expect(
      await view.evaluate(
        `document.querySelector('#rule-detail-type-0').getAttribute('aria-invalid')`,
      ),
    ).toBe('true')
    expect(
      await view.evaluate(`(() => {
        const input = document.querySelector('#rule-detail-type-0')
        return input.getAttribute('aria-describedby').split(' ')
          .map(id => document.getElementById(id)?.textContent)
      })()`),
    ).toContain('Enter an authorization detail type when human approval is required.')
    expect(await view.evaluate(submitDisabled)).toBe(true)

    await view.press('\t')
    await Bun.sleep(40)
    expect(await view.evaluate(`document.activeElement?.id`)).toBe('rule-detail-type-0')
    await view.type('document_deletion')
    expect(await view.evaluate(submitDisabled)).toBe(false)
    await clickSelector(view, 'form button[type="submit"]')
    await waitForPage(view, 'admin-mcp-resource-servers')

    await waitForText(view, connection)
    await clickElementByAriaLabel(view, `Show details for ${connection}`)
    await clickSelector(view, `main a[href$="/edit"]`)
    await waitForPage(view, 'admin-mcp-resource-server-edit')
    expect(await view.evaluate(`document.querySelector('#rule-approval-0').checked`)).toBe(true)
    expect(await view.evaluate(`document.querySelector('#rule-detail-type-0').value`)).toBe(
      'document_deletion',
    )
    expect(await view.evaluate(`document.querySelector('#rule-limit-max-0-0').value`)).toBe('7')
    expect(await view.evaluate(`document.querySelector('#rule-deny-when-0').value`)).toBe(
      expression,
    )
  } finally {
    view.close()
  }
}, 90_000)
