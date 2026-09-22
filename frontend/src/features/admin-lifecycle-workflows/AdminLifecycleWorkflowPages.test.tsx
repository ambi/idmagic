import { act, fireEvent, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, mock, spyOn, jest } from 'bun:test'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import { renderWithRouter } from '../../test/renderWithRouter'
import type { AdminGroup, AdminLifecycleWorkflow } from '../../types'
import {
  AdminLifecycleWorkflowCreatePage,
  AdminLifecycleWorkflowEditPage,
} from './AdminLifecycleWorkflowEditorPage'
import { AdminLifecycleWorkflowsPage } from './AdminLifecycleWorkflowsPage'
import { adminLifecycleWorkflowsDictionary } from './AdminLifecycleWorkflowsPage.i18n'
import { workflowFormDictionary } from './WorkflowDefinitionForm.i18n'

const wf = adminLifecycleWorkflowsDictionary.ja
const form = workflowFormDictionary.ja

const workflow: AdminLifecycleWorkflow = {
  id: 'workflow-1',
  name: '入社処理',
  status: 'draft',
  current_revision: 1,
  trigger: { kind: 'user_created' },
  actions: [{ kind: 'send_email', template_key: 'welcome' }],
  created_at: '2026-07-16T00:00:00Z',
  updated_at: '2026-07-16T00:00:00Z',
}

describe('lifecycle workflow page separation', () => {
  afterEach(() => jest.restoreAllMocks())

  it('一覧画面には作成・編集フォームを置かず、専用画面へのリンクを表示する', async () => {
    await renderWithRouter(
      <AdminLifecycleWorkflowsPage csrfToken="csrf" actorUsername="admin" workflows={[workflow]} />,
      { locale: 'ja' },
    )

    expect(screen.queryByLabelText('名前')).not.toBeInTheDocument()
    expect(screen.queryByText('トリガー（いつ実行するか）')).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: wf.addWorkflow })).toHaveAttribute(
      'href',
      '/admin/lifecycle-workflows/new',
    )
    expect(screen.getByRole('button', { name: '編集' })).toHaveAttribute(
      'href',
      '/admin/lifecycle-workflows/workflow-1/edit',
    )
  })

  it('状態に関係なくワークフローを削除し、一覧から取り除く', async () => {
    const fetchMock = spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(null, { status: 204 }),
    )
    await renderWithRouter(
      <AdminLifecycleWorkflowsPage csrfToken="csrf" actorUsername="admin" workflows={[workflow]} />,
      { locale: 'ja' },
    )

    fireEvent.click(screen.getByRole('button', { name: '削除' }))

    expect(fetchMock).not.toHaveBeenCalled()
    expect(screen.getByRole('dialog', { name: 'ワークフローの削除' })).toHaveTextContent('入社処理')

    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: '削除する' }))
      await new Promise((resolve) => setTimeout(resolve, 0))
    })

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/admin/v1/lifecycle-workflows/workflow-1',
      expect.objectContaining({ method: 'DELETE' }),
    )
    expect(screen.queryByText('入社処理')).not.toBeInTheDocument()
  })

  it('実行前確認の対象ユーザーを説明付きアプリ内ダイアログで入力する', async () => {
    const promptSpy = spyOn(window, 'prompt')
    const fetchMock = spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce(Response.json({ runs: [] }))
      .mockResolvedValueOnce(
        Response.json({
          steps: [{ action_kind: 'send_email', would_change: 'would_change' }],
        }),
      )
    await renderWithRouter(
      <AdminLifecycleWorkflowsPage csrfToken="csrf" actorUsername="admin" workflows={[workflow]} />,
      { locale: 'ja' },
    )

    fireEvent.click(screen.getByRole('button', { name: wf.history }))
    await screen.findByRole('heading', {
      name: wf.executionHeading.replace('{name}', workflow.name),
    })
    fireEvent.click(screen.getByRole('button', { name: wf.dryRunButton }))

    const dialog = screen.getByRole('dialog', { name: '実行前確認' })
    expect(dialog).toHaveTextContent(
      'このワークフローを適用した場合の処理内容を確認するユーザーを指定します。',
    )
    fireEvent.change(screen.getByLabelText('対象ユーザー ID'), {
      target: { value: 'user-123' },
    })
    fireEvent.click(screen.getByRole('button', { name: '確認する' }))

    await waitFor(() =>
      expect(fetchMock).toHaveBeenCalledWith(
        '/api/admin/v1/lifecycle-workflows/workflow-1/dry-run',
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({ target_user_id: 'user-123' }),
        }),
      ),
    )
    expect(promptSpy).not.toHaveBeenCalled()
  })

  it('専用作成画面にフォームと一覧へ戻る導線を表示する', async () => {
    await renderWithRouter(
      <AdminLifecycleWorkflowCreatePage
        csrfToken="csrf"
        actorUsername="admin"
        groups={[]}
        applications={[]}
      />,
      { locale: 'ja' },
    )

    expect(screen.getByRole('heading', { name: form.createTitle })).toBeInTheDocument()
    expect(screen.getByLabelText(form.nameLabel)).toBeInTheDocument()
    expect(screen.getByRole('link', { name: form.backToList })).toHaveAttribute(
      'href',
      '/admin/lifecycle-workflows',
    )
  })
})

function group(id: string, name: string, membership: 'manual' | 'dynamic'): AdminGroup {
  return {
    id,
    tenant_id: 'tenant-1',
    name,
    roles: [],
    member_count: 0,
    created_at: '2026-07-16T00:00:00Z',
    membership_type: membership,
  } as AdminGroup
}

// chooseOption は Base UI の Select を開き、表示名で項目を選ぶ。SelectItem はクリックの直前に
// pointerdown が無いと選択を確定しないので両方送る。
async function chooseOption(triggerName: string, optionName: string) {
  fireEvent.click(screen.getByRole('combobox', { name: triggerName }))
  const option = await screen.findByRole('option', { name: optionName })
  fireEvent.pointerDown(option)
  fireEvent.click(option)
}

const at = (template: string, index: number) => template.replace('{index}', String(index))

// 内部識別子は日本語の表示名の代わりに画面へ出てはならない。
function expectNoInternalIdentifiers() {
  for (const internal of [
    'user_created',
    'user_attributes_changed',
    'add_group_member',
    'send_email',
    'draft',
    'revision',
    'current_revision',
  ]) {
    expect(screen.queryByText(internal)).not.toBeInTheDocument()
  }
}

describe('lifecycle workflow editor', () => {
  afterEach(() => {
    jest.restoreAllMocks()
    restoreGlobals()
  })

  //spec:covers EX-IDGOVERNANCE-001-01: 作成画面がトリガーとアクションの種類を日本語の説明付きで示し、選んだトリガー、
  // 必須設定、並べ替えた順序のアクションを作成 API へ送り、内部識別子を画面に出さないことを固定する。
  it('作成画面で選んだトリガーと並べ替えたアクションを作成 API へ送る', async () => {
    const created: AdminLifecycleWorkflow = {
      id: 'workflow-9',
      name: '入社処理',
      status: 'draft',
      current_revision: 1,
      trigger: { kind: 'user_attributes_changed', watched_attributes: ['department'] },
      actions: [
        { kind: 'add_group_member', group_id: 'group-1' },
        { kind: 'send_email', template_key: 'welcome' },
      ],
      created_at: '2026-07-16T00:00:00Z',
      updated_at: '2026-07-16T00:00:00Z',
    }
    const fetchMock = spyOn(globalThis, 'fetch').mockResolvedValue(
      Response.json(created, { status: 201 }),
    )
    const assign = mock()
    stubGlobal('location', { ...window.location, assign })
    await renderWithRouter(
      <AdminLifecycleWorkflowCreatePage
        csrfToken="csrf"
        actorUsername="admin"
        groups={[group('group-1', 'エンジニア', 'manual')]}
        applications={[]}
      />,
      { locale: 'ja' },
    )

    expect(screen.getByText(form.triggerDesc_user_created)).toBeInTheDocument()
    fireEvent.change(screen.getByLabelText(form.nameLabel), { target: { value: '入社処理' } })
    await chooseOption(form.triggerSelectAria, form.trigger_user_attributes_changed)
    expect(screen.getByText(form.triggerDesc_user_attributes_changed)).toBeInTheDocument()
    fireEvent.change(screen.getByLabelText(form.watchedAttributesLabel), {
      target: { value: 'department' },
    })
    await chooseOption(at(form.ariaActionKind, 1), form.action_send_email)
    expect(screen.getByText(form.actionDesc_send_email)).toBeInTheDocument()
    fireEvent.change(screen.getByLabelText(form.templateLabel), { target: { value: 'welcome' } })
    fireEvent.click(screen.getByRole('button', { name: form.addAction }))
    await chooseOption(at(form.ariaActionKind, 2), form.action_add_group_member)
    expect(screen.getByText(form.actionDesc_add_group_member)).toBeInTheDocument()
    await chooseOption(at(form.ariaActionGroup, 2), 'エンジニア')
    fireEvent.click(screen.getByRole('button', { name: at(form.ariaMoveUp, 2) }))
    expectNoInternalIdentifiers()

    fireEvent.click(screen.getByRole('button', { name: form.createDraft }))

    await waitFor(() => expect(assign).toHaveBeenCalled())
    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit]
    expect(url).toBe('/api/admin/v1/lifecycle-workflows')
    expect(init.method).toBe('POST')
    expect(JSON.parse(String(init.body))).toEqual({
      name: '入社処理',
      trigger: { kind: 'user_attributes_changed', watched_attributes: ['department'] },
      actions: [
        { kind: 'add_group_member', group_id: 'group-1' },
        { kind: 'send_email', template_key: 'welcome' },
      ],
    })
    expect(assign).toHaveBeenCalledWith(
      expect.stringContaining('/admin/lifecycle-workflows/workflow-9/edit'),
    )
  })

  //spec:covers EX-IDGOVERNANCE-001-02: グループを使うアクションに選べるグループが無い（動的グループしか無い）と、
  // 先にグループを作るよう日本語で案内し、作成操作を無効にして作成 API を呼ばないことを固定する。
  it('選べる参照先が無いと先に作るよう案内し、作成させない', async () => {
    const fetchMock = spyOn(globalThis, 'fetch')
    await renderWithRouter(
      <AdminLifecycleWorkflowCreatePage
        csrfToken="csrf"
        actorUsername="admin"
        groups={[group('group-dynamic', '全エンジニア', 'dynamic')]}
        applications={[]}
      />,
      { locale: 'ja' },
    )

    fireEvent.change(screen.getByLabelText(form.nameLabel), { target: { value: '入社処理' } })
    await chooseOption(at(form.ariaActionKind, 1), form.action_add_group_member)

    expect(screen.getByText(form.groupEmptyError)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: form.createDraft })).toBeDisabled()
    fireEvent.click(screen.getByRole('button', { name: form.createDraft }))
    expect(fetchMock).not.toHaveBeenCalled()
  })

  //spec:covers EX-IDGOVERNANCE-001-03: トリガーとアクションを選んで実行順を並べ替えても、監視する属性とメールテンプレートキーの
  // 必須設定が足りなければ、不足している設定を日本語で示して作成操作を無効にし、作成 API を呼ばないことを固定する。
  it('必須設定が不足していると不足を示し、作成させない', async () => {
    const fetchMock = spyOn(globalThis, 'fetch')
    await renderWithRouter(
      <AdminLifecycleWorkflowCreatePage
        csrfToken="csrf"
        actorUsername="admin"
        groups={[]}
        applications={[]}
      />,
      { locale: 'ja' },
    )

    fireEvent.change(screen.getByLabelText(form.nameLabel), { target: { value: '入社処理' } })
    await chooseOption(form.triggerSelectAria, form.trigger_user_attributes_changed)
    await chooseOption(at(form.ariaActionKind, 1), form.action_disable_user)
    fireEvent.click(screen.getByRole('button', { name: form.addAction }))
    await chooseOption(at(form.ariaActionKind, 2), form.action_send_email)
    fireEvent.click(screen.getByRole('button', { name: at(form.ariaMoveUp, 2) }))

    const status = screen.getByRole('status')
    expect(status).toHaveTextContent(form.errWatchedAttributesRequired)
    expect(status).toHaveTextContent(at(form.errActionTemplate, 1))
    expect(screen.getByRole('button', { name: form.createDraft })).toBeDisabled()
    fireEvent.click(screen.getByRole('button', { name: form.createDraft }))
    expect(fetchMock).not.toHaveBeenCalled()
  })

  //spec:covers EX-IDGOVERNANCE-002-01: 編集画面が現在のトリガーとアクションを日本語の表示名で復元し、アクションを変えて
  // 保存すると expected_revision を付けて更新 API へ送り、増えたリビジョンを画面に反映することを固定する。
  it('既存定義を復元し、変更を保存すると新しいリビジョンを表示する', async () => {
    const current: AdminLifecycleWorkflow = {
      id: 'workflow-1',
      name: '退職処理',
      status: 'enabled',
      current_revision: 2,
      enabled_revision: 2,
      trigger: { kind: 'user_created' },
      actions: [{ kind: 'disable_user' }],
      created_at: '2026-07-16T00:00:00Z',
      updated_at: '2026-07-16T00:00:00Z',
    }
    const fetchMock = spyOn(globalThis, 'fetch').mockResolvedValue(
      Response.json({
        ...current,
        current_revision: 3,
        actions: [{ kind: 'send_email', template_key: 'bye' }],
      }),
    )
    await renderWithRouter(
      <AdminLifecycleWorkflowEditPage
        csrfToken="csrf"
        actorUsername="admin"
        initialWorkflow={current}
        groups={[]}
        applications={[]}
      />,
      { locale: 'ja' },
    )

    expect(screen.getByDisplayValue('退職処理')).toBeInTheDocument()
    expect(screen.getByText(form.trigger_user_created)).toBeInTheDocument()
    expect(screen.getAllByText(form.action_disable_user).length).toBeGreaterThan(0)
    expect(
      screen.getByText(
        form.editDescription.replace('{name}', '退職処理').replace('{revision}', '2'),
      ),
    ).toBeInTheDocument()

    await chooseOption(at(form.ariaActionKind, 1), form.action_send_email)
    fireEvent.change(screen.getByLabelText(form.templateLabel), { target: { value: 'bye' } })
    fireEvent.click(screen.getByRole('button', { name: form.save }))

    await waitFor(() =>
      expect(
        screen.getByText(
          form.editDescription.replace('{name}', '退職処理').replace('{revision}', '3'),
        ),
      ).toBeInTheDocument(),
    )
    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit]
    expect(url).toBe('/api/admin/v1/lifecycle-workflows/workflow-1')
    expect(init.method).toBe('PUT')
    expect(JSON.parse(String(init.body))).toMatchObject({
      expected_revision: 2,
      actions: [{ kind: 'send_email', template_key: 'bye' }],
    })
    expect(screen.getByDisplayValue('bye')).toBeInTheDocument()
    expectNoInternalIdentifiers()
  })
})
