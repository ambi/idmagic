import { expect, test } from 'bun:test'
import {
  clickButtonByText,
  clickLinkByText,
  navigateAndLogin,
  openWebView,
  setInputValue,
  uiOrigin,
  waitForPage,
  waitForText,
} from './fixtures'

//spec:covers REQ-WORKLOADIDENTITY-014, REQ-WORKLOADIDENTITY-016: ブラウザーから設定を作り、試行と編集の結果を実 API で読み戻す。
test('workload identity setup previews and edits bindings', async () => {
  const view = await openWebView({ width: 1440, height: 1800 })
  try {
    const suffix = String(Date.now())
    await navigateAndLogin(view, '/realms/default/admin/agents/new', 'admin-agent-create')
    await setInputValue(view, '#agent-name', `Workload agent ${suffix}`)
    await view.evaluate(
      `document.querySelector('input[name="auth-method"][value="later"]').click()`,
    )
    await clickButtonByText(view, 'Register')
    await waitForPage(view, 'admin-agent-detail')
    await view.navigate(`${uiOrigin}/realms/default/admin/workload-identity/new`)
    await waitForPage(view, 'admin-workload-trust-bundle-create')
    await setInputValue(view, '#name', `Workload ${suffix}`)
    await setInputValue(view, '#issuer', 'https://127.0.0.1')
    await clickButtonByText(view, 'Fetch issuer metadata')
    await waitForText(
      view,
      'Could not fetch issuer metadata. Check the URL or enter the settings manually.',
    )
    await setInputValue(view, '#issuer', `https://issuer-${suffix}.example`)
    await setInputValue(view, '#trust_domain', 'example.org')
    await setInputValue(view, '#jwks_uri', 'https://keys.example/jwks')
    await setInputValue(view, '#accepted_audiences', 'idmagic')
    await clickButtonByText(view, 'Register')
    await waitForText(view, `Workload ${suffix}`)
    await clickLinkByText(view, 'Details')
    await waitForPage(view, 'admin-workload-trust-bundle-detail')
    await waitForText(view, 'No OAuth2Client binding')
    await setInputValue(view, '#subject_pattern', `ns/${suffix}/*`)
    await clickButtonByText(view, 'Add binding')
    await waitForText(view, 'The binding has been created.')
    await setInputValue(view, '#sample_subject', `ns/${suffix}/worker`)
    await clickButtonByText(view, 'Try pattern')
    await waitForText(view, 'One binding matches.')
    await view.evaluate(`document.querySelector('tbody a[href$="/edit"]').click()`)
    await waitForPage(view, 'admin-workload-binding-edit')
    expect(await view.evaluate(`document.querySelector('#subject_pattern').value`)).toBe(
      `ns/${suffix}/*`,
    )
    await setInputValue(view, '#subject_pattern', `updated/${suffix}/*`)
    await clickButtonByText(view, 'Update')
    await waitForPage(view, 'admin-workload-trust-bundle-detail')
    await waitForText(view, 'Binding updated.')
    await waitForText(view, `updated/${suffix}/*`)
    await setInputValue(view, '#sample_subject', `ns/${suffix}/worker`)
    await clickButtonByText(view, 'Try pattern')
    await waitForText(view, 'No bindings match.')
    await setInputValue(view, '#sample_subject', `updated/${suffix}/worker`)
    await clickButtonByText(view, 'Try pattern')
    await waitForText(view, 'One binding matches.')
  } finally {
    view.close()
  }
}, 90_000)
