import { expect, test } from 'bun:test'
import {
  openWebView,
  navigateAndLogin,
  waitForPage,
  clickButtonByText,
  removeEntraFederationInPage,
  setInputValue,
  waitForText,
  uiOrigin,
} from './fixtures'

//spec:covers REQ-WSFEDERATION-001: 正式な追加と編集の経路で、内部キーの入力なしに保存し、既存の独自属性を保持すること。
test('Entra の通常設定と既存の独自属性を保持する', async () => {
  const view = await openWebView({ width: 1280, height: 1400 })
  let originalAttributes: string | undefined
  try {
    await navigateAndLogin(view, '/admin/federation/entra/new', 'admin-entra-federation-add')
    await clickButtonByText(view, '日本語')
    originalAttributes = (await view.evaluate(`(async () => {
      const api = await import('/src/api.ts')
      const schema = await api.getTenantUserAttributeSchema()
      const rps = await api.listWsFedRelyingParties()
      if (rps.some(rp => rp.entra_profile)) throw new Error('Entra fixture already exists')
      return JSON.stringify(schema.attributes)
    })()`)) as string
    expect(await view.evaluate(`document.querySelector('#entra_source_anchor') === null`)).toBe(
      true,
    )
    await waitForText(view, 'Entra のユーザー識別子（ImmutableID）')
    await waitForText(view, 'mS-DS-ConsistencyGuid')
    await waitForText(view, 'objectGUID')
    await waitForText(view, 'Get-MgUser')
    await setInputValue(view, '#entra_domain', 'immutable-default.example.test')
    await clickButtonByText(view, '保存')
    await waitForPage(view, 'admin-entra-federation-domain')
    expect(
      await view.evaluate(`(async () => {
      const api = await import('/src/api.ts')
      return (await api.getEntraFederationDomain('immutable-default.example.test')).profile.source_anchor_attribute
    })()`),
    ).toBe('on_premises_immutable_id')
    await view.evaluate(`(async () => {
      const api = await import('/src/api.ts')
      const account = await api.request('/api/auth/account')
      ${removeEntraFederationInPage}
      await api.updateTenantUserAttributeSchema(account.csrf_token, [...${originalAttributes}, {
        key: 'legacy_anchor', type: 'string', multi_valued: false, required: false,
        editable_by_user: false, unique: true, visibility: 'admin_readable', pii: false,
      }])
      await api.configureEntraFederation(account.csrf_token, {
        domain: 'immutable-legacy.example.test', source_anchor_attribute: 'legacy_anchor',
      })
    })()`)
    await view.navigate(`${uiOrigin}/admin/federation/entra/immutable-legacy.example.test/edit`)
    await waitForPage(view, 'admin-entra-federation-edit')
    expect(await view.evaluate(`document.querySelector('#entra_source_anchor') === null`)).toBe(
      true,
    )
    await waitForText(view, 'legacy_anchor')
    await setInputValue(view, '#entra_issuer', 'urn:immutable:updated')
    await clickButtonByText(view, '保存')
    await waitForPage(view, 'admin-entra-federation-domain')
    await waitForText(view, 'legacy_anchor')
    await waitForText(view, 'urn:immutable:updated')
    await view.navigate(`${uiOrigin}/admin/federation/entra/new`)
    await waitForPage(view, 'admin-entra-federation-add')
    await waitForText(view, 'legacy_anchor')
    await setInputValue(view, '#entra_domain', 'immutable-second.example.test')
    await clickButtonByText(view, '保存')
    await waitForPage(view, 'admin-entra-federation-domain')
    const profile = await view.evaluate(`(async () => {
      const api = await import('/src/api.ts')
      return (await api.getEntraFederationDomain('immutable-second.example.test')).profile
    })()`)
    expect(profile).toMatchObject({ source_anchor_attribute: 'legacy_anchor' })
    expect((profile as { domains: unknown[] }).domains).toHaveLength(2)
  } finally {
    try {
      if (originalAttributes !== undefined) {
        await view.evaluate(`(async () => {
          const api = await import('/src/api.ts')
          const account = await api.request('/api/auth/account')
          ${removeEntraFederationInPage}
          await api.updateTenantUserAttributeSchema(account.csrf_token, ${originalAttributes})
        })()`)
      }
    } finally {
      view.close()
    }
  }
}, 60_000)

//spec:covers REQ-WSFEDERATION-001: 実際の利用者の欠損と不正値による拒否で、件数、対象ユーザー、保存先と修復手順を示し、入力を保つこと。
test('ImmutableID の欠損と不正値を対象ユーザーから修復できるように案内する', async () => {
  const view = await openWebView({ width: 1280, height: 1400 })
  let userID: string | undefined
  try {
    await navigateAndLogin(view, '/admin/federation/entra/new', 'admin-entra-federation-add')
    await clickButtonByText(view, '日本語')
    userID = (await view.evaluate(`(async () => {
      const api = await import('/src/api.ts')
      const account = await api.request('/api/auth/account')
      return (await api.createAdminUser(account.csrf_token, {
        preferred_username: 'missing@immutable-missing.example.test',
        password: 'Immutable-test-90233!', email_verified: false, roles: [],
      })).id
    })()`)) as string
    await setInputValue(view, '#entra_domain', 'immutable-missing.example.test')
    await clickButtonByText(view, '保存')
    await waitForText(view, 'sourceAnchor の値がないユーザーが 1 人います。')
    const missingAlert = await view.evaluate(
      `document.querySelector('[role="alert"]')?.textContent ?? ''`,
    )
    expect(missingAlert).toContain('missing@immutable-missing.example.test')
    expect(missingAlert).toContain('on_premises_immutable_id')
    expect(missingAlert).toContain('ldap.attributes')
    expect(missingAlert).toContain('完全な再同期')
    expect(await view.evaluate(`document.querySelector('#entra_domain').value`)).toBe(
      'immutable-missing.example.test',
    )
    await view.evaluate(`(async () => {
      const api = await import('/src/api.ts')
      const account = await api.request('/api/auth/account')
      await api.updateAdminUser(account.csrf_token, ${JSON.stringify(userID)}, {
        attributes: { on_premises_immutable_id: { type: 'string', string: 'invalid@anchor' } },
      })
    })()`)
    await clickButtonByText(view, '保存')
    await waitForText(view, 'sourceAnchor を ImmutableID として使えないユーザーが 1 人います。')
    const invalidAlert = await view.evaluate(
      `document.querySelector('[role="alert"]')?.textContent ?? ''`,
    )
    expect(invalidAlert).toContain('missing@immutable-missing.example.test')
    expect(invalidAlert).toContain('on_premises_immutable_id')
    expect(invalidAlert).toContain('Entra の実値と比較')
    expect(
      await view.evaluate(`(async () => {
      const api = await import('/src/api.ts')
      return (await api.listEntraFederationDomains()).length
    })()`),
    ).toBe(0)
  } finally {
    try {
      if (userID !== undefined) {
        await view.evaluate(`(async () => {
          const api = await import('/src/api.ts')
          const account = await api.request('/api/auth/account')
          await api.deleteAdminUser(account.csrf_token, ${JSON.stringify(userID)})
        })()`)
      }
    } finally {
      view.close()
    }
  }
}, 60_000)

//spec:covers REQ-SYSTEM-011: 正式な Entra の追加画面から送った構成の拒否を、日本語の辞書で表示すること。
//spec:covers REQ-WSFEDERATION-001: 定義のない sourceAnchor 属性の構成を拒否し、属性の設定を直す案内を表示すること。
test('Entra configuration displays a localized sourceAnchor refusal', async () => {
  const view = await openWebView({ width: 1280, height: 1200 })
  try {
    await navigateAndLogin(view, '/admin/federation/entra/new', 'admin-entra-federation-add')
    await clickButtonByText(view, '日本語')
    await setInputValue(view, '#entra_domain', 'contoso.com')
    await clickButtonByText(view, 'ImmutableID の詳細設定')
    await setInputValue(view, '#entra_source_anchor', 'undefined_source_anchor')
    await clickButtonByText(view, '保存')
    await waitForText(view, 'sourceAnchor 属性を一意の属性に設定してください。')
    const alert = await view.evaluate(`document.querySelector('[role="alert"]')?.textContent ?? ''`)
    expect(alert).toBe('sourceAnchor 属性を一意の属性に設定してください。')
  } finally {
    view.close()
  }
}, 60_000)

//spec:covers REQ-WSFEDERATION-001: 正式な追加画面が構成 API と署名器につながり、保存後のドメインの詳細が Graph の作成と更新のコマンドと制限の日本語を表示すること。
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
    await clickButtonByText(view, 'ImmutableID の詳細設定')
    await setInputValue(view, '#entra_source_anchor', 'entra_graph_anchor')
    await clickButtonByText(view, '保存')
    await waitForPage(view, 'admin-entra-federation-domain')
    await waitForText(view, 'New-MgDomainFederationConfiguration')
    const commands = (
      (await view.evaluate(
        `Array.from(document.querySelectorAll('pre code'), node => node.textContent)`,
      )) as string[]
    ).filter((command) => /^(New|Update)-MgDomainFederationConfiguration/.test(command))
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
    for (const label of ['作成コマンドをコピー', '更新コマンドをコピー']) {
      expect(
        await view.evaluate(`Boolean(document.querySelector('button[aria-label="${label}"]'))`),
      ).toBe(true)
    }
    await waitForText(view, 'Microsoft Entra hybrid join の即時の登録は提供しません。')
    await waitForText(
      view,
      'Windows 10 1803 以降の端末は、Microsoft Entra Connect がコンピューターを同期した後に登録されます。',
    )
    expect(
      await view.evaluate(`document.body.textContent.includes('hybrid_join_not_supported')`),
    ).toBe(false)
  } finally {
    try {
      if (originalAttributes !== undefined) {
        await view.evaluate(`(async () => {
          const api = await import('/src/api.ts')
          const account = await api.request('/api/auth/account')
          ${removeEntraFederationInPage}
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
