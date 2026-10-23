import { expect, test } from 'bun:test'
import {
  openWebView,
  navigateAndLogin,
  clickButtonByText,
  setInputValue,
  waitForText,
} from './fixtures'

//spec:covers REQ-SYSTEM-011: 正式な Entra の追加画面から送った構成の拒否を、日本語の辞書で表示すること。
//spec:covers REQ-WSFEDERATION-001: 定義のない sourceAnchor 属性の構成を拒否し、属性の設定を直す案内を表示すること。
test('Entra configuration displays a localized sourceAnchor refusal', async () => {
  const view = await openWebView({ width: 1280, height: 1200 })
  try {
    await navigateAndLogin(view, '/admin/federation/entra/new', 'admin-entra-federation-add')
    await clickButtonByText(view, '日本語')
    await setInputValue(view, '#entra_domain', 'contoso.com')
    await setInputValue(view, '#entra_source_anchor', 'undefined_source_anchor')
    await clickButtonByText(view, '保存')
    await waitForText(view, 'sourceAnchor 属性を一意の属性に設定してください。')
    const alert = await view.evaluate(`document.querySelector('[role="alert"]')?.textContent ?? ''`)
    expect(alert).toBe('sourceAnchor 属性を一意の属性に設定してください。')
  } finally {
    view.close()
  }
}, 60_000)

//spec:covers REQ-WSFEDERATION-001: 正式な追加画面が構成 API と署名器につながり、Graph の作成と更新のコマンドと制限の日本語を表示すること。
test('Entra configuration displays Graph commands with the real signing certificate', async () => {
  const view = await openWebView({ width: 1280, height: 1200 })
  let originalAttributes: string | undefined
  try {
    await navigateAndLogin(view, '/admin/federation/entra/new', 'admin-entra-federation-add')
    await clickButtonByText(view, '日本語')
    originalAttributes = (await view.evaluate(`(async () => {
      const api = await import('/src/api.ts')
      const account = await api.request('/api/auth/account')
      const schema = await api.getTenantUserAttributeSchema()
      const rps = await api.listWsFedRelyingParties()
      if (rps.some(rp => rp.entra_profile)) throw new Error('Entra fixture already exists')
      await api.updateTenantUserAttributeSchema(account.csrf_token, [...schema.attributes, {
        key: 'entra_graph_anchor', type: 'string', multi_valued: false, required: false,
        editable_by_user: false, unique: true, visibility: 'admin_readable', pii: false,
      }])
      await api.request('/api/admin/v1/keys/rotate', {
        method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': account.csrf_token },
        body: JSON.stringify({ usage: 'XmlFederationSigning', action: 'prepare' }),
      })
      return JSON.stringify(schema.attributes)
    })()`)) as string
    await setInputValue(view, '#entra_domain', 'entra-graph.example.test')
    await setInputValue(view, '#entra_source_anchor', 'entra_graph_anchor')
    await clickButtonByText(view, '保存')
    await waitForText(view, 'New-MgDomainFederationConfiguration')
    const commands = (await view.evaluate(
      `Array.from(document.querySelectorAll('pre code'), node => node.textContent)`,
    )) as string[]
    expect(commands).toHaveLength(2)
    expect(commands[0]).toContain(
      "New-MgDomainFederationConfiguration -DomainId 'entra-graph.example.test'",
    )
    expect(commands[1]).toContain(
      "Update-MgDomainFederationConfiguration -DomainId 'entra-graph.example.test'",
    )
    expect(commands[1]).toContain('Get-MgDomainFederationConfiguration')
    for (const command of commands) {
      expect(command).toContain("-PreferredAuthenticationProtocol 'wsFed'")
      expect(command).toContain("-FederatedIdpMfaBehavior 'acceptIfMfaDoneByFederatedIdp'")
      expect(command).toContain("/realms/default/wsfed?wa=wsignout1.0'")
      expect(command).toMatch(/-SigningCertificate '[A-Za-z0-9+/=]{500,}'/)
      expect(command).toMatch(/-NextSigningCertificate '[A-Za-z0-9+/=]{500,}'/)
    }
    await waitForText(view, '作成コマンドをコピー')
    await waitForText(view, '更新コマンドをコピー')
    await waitForText(view, 'Hybrid Azure AD Join のデバイス登録は未提供です。')
    expect(
      await view.evaluate(`document.body.textContent.includes('hybrid_join_not_supported')`),
    ).toBe(false)
  } finally {
    try {
      if (originalAttributes !== undefined) {
        await view.evaluate(`(async () => {
          const api = await import('/src/api.ts')
          const account = await api.request('/api/auth/account')
          await api.deleteWsFedRelyingParty(account.csrf_token, 'urn:federation:MicrosoftOnline')
          const keys = await api.request('/api/admin/v1/keys?usage=XmlFederationSigning')
          for (const key of keys.keys.filter(key => key.prepared)) {
            await api.request('/api/admin/v1/keys/' + encodeURIComponent(key.kid) + '/disable?usage=XmlFederationSigning', {
              method: 'POST', headers: { 'X-CSRF-Token': account.csrf_token },
            })
          }
          await api.updateTenantUserAttributeSchema(account.csrf_token, ${originalAttributes})
        })()`)
      }
    } finally {
      view.close()
    }
  }
}, 60_000)
