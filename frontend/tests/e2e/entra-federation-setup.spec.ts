import { expect, test } from 'bun:test'
import {
  clickButtonByText,
  navigateAndLogin,
  openWebView,
  setInputValue,
  uiOrigin,
  waitForPage,
  waitForText,
} from './fixtures'

const domain = 'entra-setup.example.test'
const tenantID = '0f1e2d3c-4b5a-6978-8796-a5b4c3d2e1f0'

//spec:covers REQ-WSFEDERATION-001: 保存の後にドメインの詳細へ移り、再読み込みしても未確認の状態と手順を示し、一覧も未確認と示し、拒否された観測文書を入力に残すこと。
test('administrator saves a domain and revisits its unverified guidance', async () => {
  const view = await openWebView({ width: 1280, height: 1400 })
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
        key: 'entra_setup_anchor', type: 'string', multi_valued: false, required: false,
        editable_by_user: false, unique: true, visibility: 'admin_readable', pii: false,
      }])
      return JSON.stringify(schema.attributes)
    })()`)) as string
    await setInputValue(view, '#entra_domain', domain)
    await setInputValue(view, '#entra_source_anchor', 'entra_setup_anchor')
    await setInputValue(view, '#entra_tenant_id', tenantID)
    await clickButtonByText(view, '保存')
    await waitForPage(view, 'admin-entra-federation-domain')
    expect(new URL(view.url).pathname.endsWith(`/admin/federation/entra/${domain}`)).toBe(true)
    await waitForText(view, `${domain} のドメインフェデレーションを保存しました。`)

    // 再訪問しても、保存し直さずに同じ状態と手順を読める。
    await view.navigate(`${uiOrigin}/admin/federation/entra/${domain}`)
    await waitForPage(view, 'admin-entra-federation-domain')
    await waitForText(view, '未確認')
    await waitForText(view, 'IdMagic に保存しただけで、Entra への反映はまだ確かめていません。')
    const scripts = (await view.evaluate(
      `Array.from(document.querySelectorAll('pre code'), node => node.textContent)`,
    )) as string[]
    expect(scripts.some((s) => s.includes(`Connect-MgGraph -TenantId '${tenantID}'`))).toBe(true)
    expect(scripts.some((s) => s.includes('idmagic.entra-federation-observation.v1'))).toBe(true)
    expect(
      scripts.some((s) =>
        s.startsWith(`New-MgDomainFederationConfiguration -DomainId '${domain}'`),
      ),
    ).toBe(true)

    // 形式の違う文書は拒否され、入力に残る。
    const rejected = JSON.stringify({ format: 'something-else' })
    await setInputValue(view, '#entra_observation', rejected)
    await clickButtonByText(view, '取り込んで照合')
    await waitForText(view, '取り込めませんでした')
    expect(await view.evaluate(`document.querySelector('#entra_observation').value`)).toBe(rejected)
    await waitForText(view, '未確認')

    await view.navigate(`${uiOrigin}/admin/federation/entra`)
    await waitForPage(view, 'admin-entra-federation')
    const row = (await view.evaluate(
      `document.querySelector('[data-domain="${domain}"]')?.textContent ?? ''`,
    )) as string
    expect(row).toContain('未確認')
  } finally {
    try {
      if (originalAttributes !== undefined) {
        await view.evaluate(`(async () => {
          const api = await import('/src/api.ts')
          const account = await api.request('/api/auth/account')
          await api.deleteWsFedRelyingParty(account.csrf_token, 'urn:federation:MicrosoftOnline')
          await api.updateTenantUserAttributeSchema(account.csrf_token, ${originalAttributes})
        })()`)
      }
    } finally {
      view.close()
    }
  }
}, 90_000)
