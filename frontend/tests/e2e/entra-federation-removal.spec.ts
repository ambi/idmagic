import { expect, test } from 'bun:test'
import {
  answerEntraRealm,
  clickButtonByText,
  clickEnabledButtonByText,
  navigateAndLogin,
  openWebView,
  removeEntraFederationInPage,
  uiOrigin,
  waitForPage,
  waitForText,
} from './fixtures'

const removed = 'entra-removal-a.example.test'
const kept = 'entra-removal-b.example.test'

// 確認の画面の確定のボタンは、解除の項目のボタンと同じ文言なので、ダイアログの中から押す。
async function confirmInDialog(view: Bun.WebView, text: string): Promise<void> {
  const deadline = Date.now() + 10_000
  while (Date.now() < deadline) {
    const clicked = await view.evaluate(`(() => {
      const dialog = document.querySelector('[role="dialog"]')
      const button = dialog && [...dialog.querySelectorAll('button')]
        .find((b) => (b.textContent ?? '').includes(${JSON.stringify(text)}) && !b.disabled)
      if (!button) return false
      button.click()
      return true
    })()`)
    if (clicked === true) return
    await Bun.sleep(100)
  }
  throw new Error(`dialog button not found: ${text}`)
}

async function removeFromDetail(view: Bun.WebView): Promise<void> {
  await view.navigate(`${uiOrigin}/admin/federation/entra/${removed}`)
  await waitForPage(view, 'admin-entra-federation-domain')
  await clickEnabledButtonByText(view, 'ドメインを解除')
  await confirmInDialog(view, 'ドメインを解除')
}

//spec:covers REQ-WSFEDERATION-001: 画面の解除が、レルム探索がまだ IdMagic を指すと答える間は拒否されて設定を残し、探索が Managed と答えるようになった後の再実行で対象のドメインだけを外して一覧へ移ること。
test('administrator removes one domain only after Entra stops sending it to IdMagic', async () => {
  const view = await openWebView({ width: 1280, height: 1600 })
  let originalAttributes: string | undefined
  try {
    await navigateAndLogin(view, '/admin/federation/entra', 'admin-entra-federation')
    await clickButtonByText(view, '日本語')
    const setup = JSON.parse(
      (await view.evaluate(`(async () => {
        const api = await import('/src/api.ts')
        const account = await api.request('/api/auth/account')
        const schema = await api.getTenantUserAttributeSchema()
        const rps = await api.listWsFedRelyingParties()
        if (rps.some(rp => rp.entra_profile)) throw new Error('Entra fixture already exists')
        await api.updateTenantUserAttributeSchema(account.csrf_token, [...schema.attributes, {
          key: 'entra_removal_anchor', type: 'string', multi_valued: false, required: false,
          editable_by_user: false, unique: true, visibility: 'admin_readable', pii: false,
        }])
        for (const domain of ${JSON.stringify([removed, kept])}) {
          await api.configureEntraFederation(account.csrf_token, {
            domain, issuer_uri: '', source_anchor_attribute: 'entra_removal_anchor',
            reply_url: '', entra_tenant_id: '',
          })
        }
        const detail = await api.getEntraFederationDomain(${JSON.stringify(removed)})
        return JSON.stringify({ attributes: schema.attributes, passive: detail.profile.passive_logon_uri })
      })()`)) as string,
    ) as { attributes: unknown; passive: string }
    originalAttributes = JSON.stringify(setup.attributes)

    // Entra がまだこのドメインを IdMagic の受動のエンドポイントへ送っている間は、解除は拒否され、詳細に留まる。
    answerEntraRealm(removed, {
      NameSpaceType: 'Federated',
      AuthURL: `${setup.passive}?username=idmagic-realm-check%40${removed}&wa=wsignin1.0&wtrealm=urn%3afederation%3aMicrosoftOnline&wctx=`,
    })
    await removeFromDetail(view)
    await waitForText(view, 'Entra はまだこのドメインの利用者を IdMagic へ送っています。')
    expect(new URL(view.url).pathname.endsWith(`/admin/federation/entra/${removed}`)).toBe(true)

    // Entra を Managed へ戻した後、再訪問して解除をやり直すと、対象のドメインだけが外れる。
    answerEntraRealm(removed, { NameSpaceType: 'Managed' })
    await removeFromDetail(view)
    await waitForPage(view, 'admin-entra-federation')
    await waitForText(view, `${removed} のフェデレーションを解除しました。`)
    const listed = (await view.evaluate(
      `Array.from(document.querySelectorAll('[data-domain]'), node => node.getAttribute('data-domain'))`,
    )) as string[]
    expect(listed).toEqual([kept])
  } finally {
    answerEntraRealm(removed, null)
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
}, 90_000)
