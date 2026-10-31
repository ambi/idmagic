import { expect, test } from 'bun:test'
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
} from './fixtures'

// 型は TypeSpec から生成しておらず、単体テストのダブルは同じ誤りを共有するので、画面から実 API
// までをこの 1 本で通す。判定の境界は Go のテストが持つ。
const model = [
  { name: 'user', relations: [] },
  { name: 'agent', relations: [] },
  {
    name: 'document',
    relations: [
      { name: 'viewer', rewrites: [{ kind: 'direct', direct_subject_types: ['user', 'agent'] }] },
    ],
  },
]

async function fillCheck(view: Bun.WebView, resourceId: string): Promise<void> {
  await setInputValue(view, '#check-subject-type', 'user')
  await setInputValue(view, '#check-subject-id', 'u-e2e')
  await setInputValue(view, '#check-resource-type', 'document')
  await setInputValue(view, '#check-resource-id', resourceId)
  await setInputValue(view, '#check-relation', 'viewer')
}

//spec:covers REQ-AUTHORIZATION-011: 正式な編集画面で差分を確認してから発行し、最新の版が詳細に表示される。
//spec:covers REQ-AUTHORIZATION-013: 判定の許可と経路、代行者の拒否の説明、波及削除の後の拒否を画面で観測する。
test('administrator publishes a model and explains an access decision', async () => {
  const view = await openWebView({ width: 1440, height: 2400 })
  try {
    const resourceId = `d-${Date.now()}`
    await navigateAndLogin(
      view,
      '/admin/authorization/model/edit',
      'admin-authorization-model-edit',
    )
    await clickButtonByText(view, 'JSON')
    await setInputValue(view, '#authorization-model-json', JSON.stringify(model))
    await clickEnabledButtonByText(view, 'Review publication')
    await waitForText(view, 'Differences from the latest version')
    await clickEnabledButtonByText(view, 'Publish')
    await waitForPage(view, 'admin-authorization-model')

    await clickLinkByText(view, 'Relation tuples')
    await waitForPage(view, 'admin-authorization-tuples')
    await clickLinkByText(view, 'Add relation tuple')
    await waitForPage(view, 'admin-authorization-tuple-create')
    await setSelectValue(view, '#tuple-resource-type', 'document')
    await setInputValue(view, '#tuple-resource-id', resourceId)
    await setSelectValue(view, '#tuple-relation', 'viewer')
    await setSelectValue(view, '#tuple-subject-form', 'user')
    await setInputValue(view, '#tuple-subject-id', 'u-e2e')
    await clickEnabledButtonByText(view, 'Add relation tuple')
    await waitForPage(view, 'admin-authorization-tuples')
    await waitForText(view, resourceId)

    await clickLinkByText(view, 'Access check')
    await waitForPage(view, 'admin-authorization-check')
    await fillCheck(view, resourceId)
    await clickEnabledButtonByText(view, 'Check access')
    await waitForText(view, 'Permitted')
    await waitForText(view, 'document#viewer')

    await clickButtonByText(view, 'Add actor')
    await setInputValue(view, '#check-actor-0-type', 'agent')
    await setInputValue(view, '#check-actor-0-id', 'a-e2e')
    await clickEnabledButtonByText(view, 'Check access')
    await waitForText(view, 'Denied')
    await waitForText(view, 'An actor in the chain does not have the relation on this resource.')

    await view.navigate(`${uiOrigin}/admin/authorization/relation-tuples`)
    await waitForPage(view, 'admin-authorization-tuples')
    await setInputValue(view, '#object-type', 'document')
    await setInputValue(view, '#object-id', resourceId)
    await clickEnabledButtonByText(view, 'Delete relation tuples of the object')
    await clickEnabledButtonByText(view, 'Delete now')
    await waitForText(view, 'Deleted 1 relation tuple(s).')

    await clickLinkByText(view, 'Access check')
    await waitForPage(view, 'admin-authorization-check')
    await fillCheck(view, resourceId)
    await clickEnabledButtonByText(view, 'Check access')
    await waitForText(view, 'Denied')
    expect(await view.evaluate('document.body.textContent?.includes("Permitted")')).toBe(false)
  } finally {
    view.close()
  }
}, 120_000)
