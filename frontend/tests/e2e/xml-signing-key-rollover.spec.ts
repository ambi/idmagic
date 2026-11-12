import { expect, test } from 'bun:test'
import {
  clickButtonByText,
  navigateAndLogin,
  openWebView,
  removeEntraFederationInPage,
  setInputValue,
  uiOrigin,
  waitForPage,
  waitForText,
} from './fixtures'

const domain = 'xml-rollover.example.test'
const tenantID = '1a2b3c4d-5e6f-4a7b-8c9d-0e1f2a3b4c5d'
const rolloverPath = '/admin/keys/xml-federation'

type ListedKey = {
  kid: string
  active: boolean
  prepared?: boolean
  expires_at?: string
  certificate_fingerprint_sha256?: string
}

async function xmlKeys(view: Bun.WebView): Promise<ListedKey[]> {
  return JSON.parse(
    (await view.evaluate(`(async () => {
      const api = await import('/src/api.ts')
      return JSON.stringify(await api.listXmlFederationKeys())
    })()`)) as string,
  )
}

// readBackObservation は、Graph の読み戻しが書き出す観測文書を、ドメインの詳細が示す期待値から作る。
// 実テナントの代わりに、管理者が更新のコマンドを Entra に適用した結果を表す。
async function readBackObservation(view: Bun.WebView): Promise<string> {
  return (await view.evaluate(`(async () => {
    const api = await import('/src/api.ts')
    const detail = await api.getEntraFederationDomain(${JSON.stringify(domain)})
    const p = detail.powershell
    return JSON.stringify({
      format: 'idmagic.entra-federation-observation.v1',
      entra_tenant_id: ${JSON.stringify(tenantID)},
      domain: ${JSON.stringify(domain)},
      authentication_type: 'Federated',
      retrieved_at: new Date().toISOString(),
      federation_configurations: [{
        issuer_uri: p.IssuerUri, passive_sign_in_uri: p.PassiveSignInUri,
        active_sign_in_uri: p.ActiveSignInUri, metadata_exchange_uri: p.MetadataExchangeUri,
        sign_out_uri: p.SignOutUri, signing_certificate: p.SigningCertificate,
        next_signing_certificate: p.NextSigningCertificate ?? '',
        preferred_authentication_protocol: p.PreferredAuthenticationProtocol,
        federated_idp_mfa_behavior: p.FederatedIdpMfaBehavior,
      }],
    })
  })()`)) as string
}

async function activateDisabled(view: Bun.WebView): Promise<boolean> {
  return (await view.evaluate(`(() => {
    const button = [...document.querySelectorAll('[data-testid="xml-key-activate"] button')]
      .find((b) => (b.textContent ?? '').includes('新しい鍵に切り替える'))
    return button ? button.disabled : null
  })()`)) as boolean
}

//spec:covers REQ-SIGNINGKEYS-006: 画面で準備した鍵は再訪問でも同じ kid のまま署名に使われず、Entra の照合と全信頼先の確認の後の切り替えでだけ有効になり、旧鍵は期限付きで残ること。
test('administrator prepares the shared XML key and activates it after every trust is confirmed', async () => {
  const view = await openWebView({ width: 1280, height: 1600 })
  let originalAttributes: string | undefined
  try {
    await navigateAndLogin(view, rolloverPath, 'admin-xml-federation-key')
    await clickButtonByText(view, '日本語')
    originalAttributes = (await view.evaluate(`(async () => {
      const api = await import('/src/api.ts')
      const account = await api.request('/api/auth/account')
      const schema = await api.getTenantUserAttributeSchema()
      const rps = await api.listWsFedRelyingParties()
      if (rps.some(rp => rp.entra_profile)) throw new Error('Entra fixture already exists')
      for (const key of await api.listXmlFederationKeys()) {
        if (key.prepared) await api.cancelPreparedXmlFederationKey(account.csrf_token, key.kid)
      }
      await api.updateTenantUserAttributeSchema(account.csrf_token, [...schema.attributes, {
        key: 'xml_rollover_anchor', type: 'string', multi_valued: false, required: false,
        editable_by_user: false, unique: true, visibility: 'admin_readable', pii: false,
      }])
      await api.configureEntraFederation(account.csrf_token, {
        domain: ${JSON.stringify(domain)}, source_anchor_attribute: 'xml_rollover_anchor',
        entra_tenant_id: ${JSON.stringify(tenantID)},
      })
      return JSON.stringify(schema.attributes)
    })()`)) as string
    const before = (await xmlKeys(view)).find((key) => key.active)
    expect(before).toBeDefined()

    // 準備しても署名に使う鍵は変わらず、再訪問しても同じ準備済みの鍵から続く。
    await view.navigate(`${uiOrigin}${rolloverPath}`)
    await waitForPage(view, 'admin-xml-federation-key')
    await clickButtonByText(view, '次の証明書を準備')
    await waitForText(view, '次の証明書を準備しました。')
    const prepared = (await xmlKeys(view)).find((key) => key.prepared)
    expect(prepared).toBeDefined()
    expect((await xmlKeys(view)).find((key) => key.active)?.kid).toBe(before!.kid)
    await view.navigate(`${uiOrigin}${rolloverPath}`)
    await waitForPage(view, 'admin-xml-federation-key')
    await waitForText(view, prepared!.kid)
    await waitForText(view, `${domain} の照合で、準備済みの証明書が一致していません。`)
    expect(await activateDisabled(view)).toBe(true)

    // Entra へ更新のコマンドを適用した読み戻しを、ドメインの詳細で取り込んで照合する。
    const observation = await readBackObservation(view)
    expect(JSON.parse(observation).federation_configurations[0].next_signing_certificate).not.toBe(
      '',
    )
    await view.navigate(`${uiOrigin}/admin/federation/entra/${domain}`)
    await waitForPage(view, 'admin-entra-federation-domain')
    await setInputValue(view, '#entra_observation', observation)
    await clickButtonByText(view, '取り込んで照合')
    await waitForText(view, '取り込んだ Entra の登録値が、現在の期待値とすべて一致しました。')

    // 照合が一致しても、ほかの信頼先と全体を確認するまで切り替えられない。
    await view.navigate(`${uiOrigin}${rolloverPath}`)
    await waitForPage(view, 'admin-xml-federation-key')
    await waitForText(view, '準備済みの証明書を確認済み')
    expect(await activateDisabled(view)).toBe(true)
    await view.evaluate(`(() => {
      for (const input of document.querySelectorAll('[data-testid="xml-key-register"] input[type="checkbox"], [data-testid="xml-key-activate"] input[type="checkbox"]')) {
        if (!input.checked) input.click()
      }
    })()`)
    expect(await activateDisabled(view)).toBe(false)
    expect((await xmlKeys(view)).find((key) => key.active)?.kid).toBe(before!.kid)
    await clickButtonByText(view, '新しい鍵に切り替える')
    await view.evaluate(`(() => {
      const buttons = [...document.querySelectorAll('[role="dialog"] button')]
        .filter((b) => (b.textContent ?? '').includes('新しい鍵に切り替える'))
      buttons.at(-1)?.click()
    })()`)
    await waitForText(view, `新しい鍵 ${prepared!.kid} に切り替えました。`)

    const after = await xmlKeys(view)
    expect(after.find((key) => key.active)?.kid).toBe(prepared!.kid)
    const previous = after.find((key) => key.kid === before!.kid)
    expect(previous?.active).toBe(false)
    expect(previous?.prepared).toBe(false)
    expect(Date.parse(previous?.expires_at ?? '')).toBeGreaterThan(Date.now() + 6 * 24 * 3600_000)
    await waitForText(view, `旧鍵 ${before!.kid} の公開の期限`)
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
}, 120_000)
