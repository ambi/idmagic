import { useEffect, useMemo, useState } from 'react'
import type { LifecycleWorkflowInput } from '../../api/admin'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { Select, type SelectOption } from '../../components/ui/select'
import {
  REQUIRED_ACTIONS,
  requiredActionLabel,
  type AdminApplication,
  type AdminGroup,
  type AdminLifecycleWorkflow,
  type WorkflowAction,
  type WorkflowActionKind,
  type WorkflowTrigger,
} from '../../types'
import { useDictionary } from '../../lib/i18n'
import { domainLabelsDictionary } from '../../lib/i18n/domainLabels.i18n'

type TriggerKind = WorkflowTrigger['kind']
type ActionDraft = Omit<WorkflowAction, 'kind'> & {
  key: string
  kind: WorkflowActionKind | ''
}

type FormDraft = {
  name: string
  description: string
  triggerKind: TriggerKind
  watchedAttributes: string
  fromStatus: string
  toStatus: string
  actions: ActionDraft[]
}

const triggerOptions: SelectOption[] = [
  { value: 'user_created', label: 'ユーザーが作成されたとき' },
  { value: 'user_attributes_changed', label: 'ユーザー属性が変更されたとき' },
  { value: 'user_status_changed', label: 'ユーザー状態が変更されたとき' },
]

const triggerDescriptions: Record<TriggerKind, string> = {
  user_created: '新しいユーザーが登録された直後に実行します。',
  user_attributes_changed: '指定した属性の値が実際に変わったときだけ実行します。',
  user_status_changed: 'ユーザーが指定した変更前の状態から変更後の状態へ移ったときに実行します。',
}

const actionOptions: SelectOption[] = [
  { value: 'add_group_member', label: 'グループに追加' },
  { value: 'remove_group_member', label: 'グループから削除' },
  { value: 'assign_application', label: 'アプリケーションを割り当て' },
  { value: 'unassign_application', label: 'アプリケーションの割り当てを解除' },
  { value: 'set_required_action', label: '次回ログイン時の必須対応を追加' },
  { value: 'clear_required_action', label: '次回ログイン時の必須対応を解除' },
  { value: 'enable_user', label: 'ユーザーを有効化' },
  { value: 'disable_user', label: 'ユーザーを無効化' },
  { value: 'send_email', label: 'メールを送信' },
]

const actionDescriptions: Record<WorkflowActionKind, string> = {
  add_group_member: '対象ユーザーを選択したグループのメンバーにします。',
  remove_group_member: '対象ユーザーを選択したグループから外します。',
  assign_application: '対象ユーザーにアプリケーションを利用できる権限を付けます。',
  unassign_application: '対象ユーザーからアプリケーションの利用権限を外します。',
  set_required_action: '対象ユーザーの次回ログイン時に指定した対応を求めます。',
  clear_required_action: '対象ユーザーに設定済みの必須対応を解除します。',
  enable_user: '対象ユーザーをログイン可能な状態にします。',
  disable_user: '対象ユーザーをログインできない状態にします。',
  send_email: '対象ユーザーの確認済みメールアドレスへテンプレートメールを送ります。',
}

const userStatusOptions: SelectOption[] = [
  { value: 'active', label: '有効' },
  { value: 'disabled', label: '無効' },
  { value: 'pending_deletion', label: '削除待ち' },
  { value: 'deleted', label: '削除済み' },
  { value: 'locked', label: 'ロック中' },
  { value: 'staged', label: '準備中' },
  { value: 'suspended', label: '一時停止中' },
]

const visibilityOptions: SelectOption[] = [
  { value: 'visible', label: 'ユーザーに表示する' },
  { value: 'hidden', label: 'ユーザーに表示しない' },
]

let nextActionKey = 0

function actionKey(): string {
  nextActionKey += 1
  return `workflow-action-${nextActionKey}`
}

function emptyAction(): ActionDraft {
  return { key: actionKey(), kind: '' }
}

function emptyDraft(): FormDraft {
  return {
    name: '',
    description: '',
    triggerKind: 'user_created',
    watchedAttributes: '',
    fromStatus: '',
    toStatus: '',
    actions: [emptyAction()],
  }
}

function workflowDraft(workflow?: AdminLifecycleWorkflow): FormDraft {
  if (!workflow) return emptyDraft()
  return {
    name: workflow.name,
    description: workflow.description ?? '',
    triggerKind: workflow.trigger.kind,
    watchedAttributes: workflow.trigger.watched_attributes?.join(', ') ?? '',
    fromStatus: workflow.trigger.from_status ?? '',
    toStatus: workflow.trigger.to_status ?? '',
    actions: workflow.actions.map((action) => ({ key: actionKey(), ...action })),
  }
}

function compactAction(action: ActionDraft): WorkflowAction | null {
  if (!action.kind) return null
  switch (action.kind) {
    case 'add_group_member':
    case 'remove_group_member':
      return { kind: action.kind, group_id: action.group_id?.trim() }
    case 'assign_application':
      return {
        kind: action.kind,
        application_id: action.application_id?.trim(),
        visibility: action.visibility || 'visible',
      }
    case 'unassign_application':
      return { kind: action.kind, application_id: action.application_id?.trim() }
    case 'set_required_action':
    case 'clear_required_action':
      return { kind: action.kind, required_action: action.required_action }
    case 'enable_user':
    case 'disable_user':
      return {
        kind: action.kind,
        ...(action.reason?.trim() ? { reason: action.reason.trim() } : {}),
      }
    case 'send_email':
      return { kind: action.kind, template_key: action.template_key?.trim() }
  }
}

export function validateWorkflowDraft(draft: FormDraft): string[] {
  const errors: string[] = []
  if (!draft.name.trim()) errors.push('ワークフロー名を入力してください。')
  if (draft.triggerKind === 'user_attributes_changed' && !draft.watchedAttributes.trim()) {
    errors.push('監視するユーザー属性を1つ以上入力してください。')
  }
  if (draft.triggerKind === 'user_status_changed') {
    if (!draft.fromStatus) errors.push('変更前のユーザー状態を選択してください。')
    if (!draft.toStatus) errors.push('変更後のユーザー状態を選択してください。')
    if (draft.fromStatus && draft.fromStatus === draft.toStatus) {
      errors.push('変更前と変更後には異なるユーザー状態を選択してください。')
    }
  }
  draft.actions.forEach((action, index) => {
    const prefix = `アクション ${index + 1}`
    if (!action.kind) errors.push(`${prefix}の種類を選択してください。`)
    if (
      (action.kind === 'add_group_member' || action.kind === 'remove_group_member') &&
      !action.group_id
    ) {
      errors.push(`${prefix}で対象グループを選択してください。`)
    }
    if (
      (action.kind === 'assign_application' || action.kind === 'unassign_application') &&
      !action.application_id
    ) {
      errors.push(`${prefix}で対象アプリケーションを選択してください。`)
    }
    if (
      (action.kind === 'set_required_action' || action.kind === 'clear_required_action') &&
      !action.required_action
    ) {
      errors.push(`${prefix}で必須対応を選択してください。`)
    }
    if (action.kind === 'send_email' && !action.template_key?.trim()) {
      errors.push(`${prefix}でメールテンプレートキーを入力してください。`)
    }
  })
  return errors
}

export function workflowInput(draft: FormDraft): LifecycleWorkflowInput {
  let trigger: WorkflowTrigger
  if (draft.triggerKind === 'user_attributes_changed') {
    trigger = {
      kind: draft.triggerKind,
      watched_attributes: draft.watchedAttributes
        .split(',')
        .map((value) => value.trim())
        .filter(Boolean),
    }
  } else if (draft.triggerKind === 'user_status_changed') {
    trigger = {
      kind: draft.triggerKind,
      from_status: draft.fromStatus,
      to_status: draft.toStatus,
    }
  } else {
    trigger = { kind: draft.triggerKind }
  }
  return {
    name: draft.name.trim(),
    ...(draft.description.trim() ? { description: draft.description.trim() } : {}),
    trigger,
    actions: draft.actions
      .map(compactAction)
      .filter((action): action is WorkflowAction => !!action),
  }
}

export function workflowStatusLabel(status: AdminLifecycleWorkflow['status']): string {
  return {
    draft: '下書き',
    enabled: '有効',
    disabled: '無効',
    archived: 'アーカイブ済み',
  }[status]
}

export function workflowTriggerLabel(kind: TriggerKind): string {
  return triggerOptions.find((option) => option.value === kind)?.label ?? '不明なトリガー'
}

export function workflowActionLabel(kind: WorkflowActionKind): string {
  return actionOptions.find((option) => option.value === kind)?.label ?? '不明なアクション'
}

export function WorkflowDefinitionForm({
  workflow,
  groups,
  applications,
  busy,
  onSubmit,
  onCancel,
}: {
  workflow?: AdminLifecycleWorkflow
  groups: AdminGroup[]
  applications: AdminApplication[]
  busy: boolean
  onSubmit: (input: LifecycleWorkflowInput) => Promise<void>
  onCancel?: () => void
}) {
  const [draft, setDraft] = useState(() => workflowDraft(workflow))
  const tLabels = useDictionary(domainLabelsDictionary)
  useEffect(() => {
    setDraft(workflowDraft(workflow))
  }, [workflow])
  const errors = validateWorkflowDraft(draft)
  const groupOptions = useMemo(
    () =>
      groups
        .filter((group) => group.membership_type !== 'dynamic')
        .map((group) => ({ value: group.id, label: group.name })),
    [groups],
  )
  const applicationOptions = useMemo(
    () =>
      applications.map((application) => ({
        value: application.application_id,
        label: application.name,
      })),
    [applications],
  )
  const requiredActionOptions = REQUIRED_ACTIONS.map((action) => ({
    value: action,
    label: requiredActionLabel(action, tLabels),
  }))

  function updateAction(index: number, change: Partial<ActionDraft>) {
    setDraft((current) => ({
      ...current,
      actions: current.actions.map((action, actionIndex) =>
        actionIndex === index ? { ...action, ...change } : action,
      ),
    }))
  }

  function moveAction(index: number, offset: -1 | 1) {
    setDraft((current) => {
      const actions = [...current.actions]
      const target = index + offset
      if (target < 0 || target >= actions.length) return current
      ;[actions[index], actions[target]] = [actions[target], actions[index]]
      return { ...current, actions }
    })
  }

  async function submit() {
    if (errors.length) return
    await onSubmit(workflowInput(draft))
  }

  return (
    <Card className="mb-6 space-y-6 p-5">
      <div className="grid gap-4 md:grid-cols-2">
        <div className="space-y-2">
          <Label htmlFor="workflow-name">名前</Label>
          <Input
            id="workflow-name"
            value={draft.name}
            maxLength={100}
            onChange={(event) => setDraft({ ...draft, name: event.target.value })}
            placeholder="例: 入社時の初期設定"
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="workflow-description">説明（任意）</Label>
          <Input
            id="workflow-description"
            value={draft.description}
            maxLength={500}
            onChange={(event) => setDraft({ ...draft, description: event.target.value })}
            placeholder="このワークフローの目的"
          />
        </div>
      </div>

      <section className="rounded-lg border border-slate-200 p-4">
        <Label>トリガー（いつ実行するか）</Label>
        <div className="mt-2 max-w-xl">
          <Select
            aria-label="トリガーの種類"
            value={draft.triggerKind}
            options={triggerOptions}
            onValueChange={(value) => setDraft({ ...draft, triggerKind: value as TriggerKind })}
            className="w-full"
          />
          <p className="mt-2 text-sm text-slate-600">{triggerDescriptions[draft.triggerKind]}</p>
        </div>
        {draft.triggerKind === 'user_attributes_changed' ? (
          <div className="mt-4 max-w-xl space-y-2">
            <Label htmlFor="workflow-attributes">監視する属性</Label>
            <Input
              id="workflow-attributes"
              value={draft.watchedAttributes}
              onChange={(event) => setDraft({ ...draft, watchedAttributes: event.target.value })}
              placeholder="例: department, job_title"
            />
            <p className="text-xs text-slate-500">複数指定する場合はカンマで区切ります。</p>
          </div>
        ) : null}
        {draft.triggerKind === 'user_status_changed' ? (
          <div className="mt-4 grid max-w-2xl gap-4 md:grid-cols-2">
            <div className="space-y-2">
              <Label>変更前の状態</Label>
              <Select
                aria-label="変更前のユーザー状態"
                value={draft.fromStatus}
                options={userStatusOptions}
                placeholder="選択してください"
                onValueChange={(value) => setDraft({ ...draft, fromStatus: value })}
                className="w-full"
              />
            </div>
            <div className="space-y-2">
              <Label>変更後の状態</Label>
              <Select
                aria-label="変更後のユーザー状態"
                value={draft.toStatus}
                options={userStatusOptions}
                placeholder="選択してください"
                onValueChange={(value) => setDraft({ ...draft, toStatus: value })}
                className="w-full"
              />
            </div>
          </div>
        ) : null}
      </section>

      <section className="space-y-3">
        <div className="flex items-center justify-between gap-3">
          <div>
            <Label>アクション（何を行うか）</Label>
            <p className="mt-1 text-sm text-slate-600">
              上から順番に実行します。最大20件まで設定できます。
            </p>
          </div>
          <Button
            type="button"
            variant="outline"
            disabled={draft.actions.length >= 20}
            onClick={() => setDraft({ ...draft, actions: [...draft.actions, emptyAction()] })}
          >
            アクションを追加
          </Button>
        </div>
        {draft.actions.map((action, index) => (
          <Card className="space-y-4 p-4" key={action.key}>
            <div className="flex flex-wrap items-center justify-between gap-2">
              <span className="font-semibold">
                {index + 1}. {action.kind ? workflowActionLabel(action.kind) : '種類を選択'}
              </span>
              <div className="flex gap-2">
                <Button
                  type="button"
                  variant="outline"
                  disabled={index === 0}
                  aria-label={`アクション ${index + 1} を上へ`}
                  onClick={() => moveAction(index, -1)}
                >
                  上へ
                </Button>
                <Button
                  type="button"
                  variant="outline"
                  disabled={index === draft.actions.length - 1}
                  aria-label={`アクション ${index + 1} を下へ`}
                  onClick={() => moveAction(index, 1)}
                >
                  下へ
                </Button>
                <Button
                  type="button"
                  variant="outline"
                  disabled={draft.actions.length === 1}
                  aria-label={`アクション ${index + 1} を削除`}
                  onClick={() =>
                    setDraft({
                      ...draft,
                      actions: draft.actions.filter((_, actionIndex) => actionIndex !== index),
                    })
                  }
                >
                  削除
                </Button>
              </div>
            </div>
            <div className="max-w-xl">
              <Select
                aria-label={`アクション ${index + 1} の種類`}
                value={action.kind}
                options={actionOptions}
                placeholder="アクションの種類を選択"
                onValueChange={(value) =>
                  updateAction(index, { kind: value as WorkflowActionKind })
                }
                className="w-full"
              />
              {action.kind ? (
                <p className="mt-2 text-sm text-slate-600">{actionDescriptions[action.kind]}</p>
              ) : null}
            </div>
            {action.kind === 'add_group_member' || action.kind === 'remove_group_member' ? (
              <div className="max-w-xl space-y-2">
                <Label>対象グループ</Label>
                <Select
                  aria-label={`アクション ${index + 1} の対象グループ`}
                  value={action.group_id ?? ''}
                  options={groupOptions}
                  placeholder={
                    groupOptions.length ? 'グループを選択' : '選択可能なグループがありません'
                  }
                  onValueChange={(value) => updateAction(index, { group_id: value })}
                  className="w-full"
                />
                {!groupOptions.length ? (
                  <p className="text-sm text-red-700">先に手動管理のグループを作成してください。</p>
                ) : null}
              </div>
            ) : null}
            {action.kind === 'assign_application' || action.kind === 'unassign_application' ? (
              <div className="grid max-w-2xl gap-4 md:grid-cols-2">
                <div className="space-y-2">
                  <Label>対象アプリケーション</Label>
                  <Select
                    aria-label={`アクション ${index + 1} の対象アプリケーション`}
                    value={action.application_id ?? ''}
                    options={applicationOptions}
                    placeholder={
                      applicationOptions.length
                        ? 'アプリケーションを選択'
                        : '選択可能なアプリケーションがありません'
                    }
                    onValueChange={(value) => updateAction(index, { application_id: value })}
                    className="w-full"
                  />
                  {!applicationOptions.length ? (
                    <p className="text-sm text-red-700">先にアプリケーションを作成してください。</p>
                  ) : null}
                </div>
                {action.kind === 'assign_application' ? (
                  <div className="space-y-2">
                    <Label>ポータルでの表示</Label>
                    <Select
                      aria-label={`アクション ${index + 1} の表示設定`}
                      value={action.visibility || 'visible'}
                      options={visibilityOptions}
                      onValueChange={(value) => updateAction(index, { visibility: value })}
                      className="w-full"
                    />
                  </div>
                ) : null}
              </div>
            ) : null}
            {action.kind === 'set_required_action' || action.kind === 'clear_required_action' ? (
              <div className="max-w-xl space-y-2">
                <Label>必須対応</Label>
                <Select
                  aria-label={`アクション ${index + 1} の必須対応`}
                  value={action.required_action ?? ''}
                  options={requiredActionOptions}
                  placeholder="必須対応を選択"
                  onValueChange={(value) => updateAction(index, { required_action: value })}
                  className="w-full"
                />
              </div>
            ) : null}
            {action.kind === 'enable_user' || action.kind === 'disable_user' ? (
              <div className="max-w-xl space-y-2">
                <Label htmlFor={`workflow-action-reason-${index}`}>理由（任意）</Label>
                <Input
                  id={`workflow-action-reason-${index}`}
                  value={action.reason ?? ''}
                  onChange={(event) => updateAction(index, { reason: event.target.value })}
                  placeholder="監査時に分かる理由"
                />
              </div>
            ) : null}
            {action.kind === 'send_email' ? (
              <div className="max-w-xl space-y-2">
                <Label htmlFor={`workflow-template-${index}`}>メールテンプレートキー</Label>
                <Input
                  id={`workflow-template-${index}`}
                  value={action.template_key ?? ''}
                  onChange={(event) => updateAction(index, { template_key: event.target.value })}
                  placeholder="例: welcome"
                />
                <p className="text-xs text-slate-500">
                  送信するメール本文を識別するテンプレート名です。
                </p>
              </div>
            ) : null}
          </Card>
        ))}
      </section>

      {errors.length ? (
        <div
          role="status"
          className="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-800"
        >
          <p className="font-semibold">入力内容を確認してください。</p>
          <ul className="mt-2 list-disc pl-5">
            {errors.map((error) => (
              <li key={error}>{error}</li>
            ))}
          </ul>
        </div>
      ) : null}
      <div className="flex justify-end gap-2">
        {onCancel ? (
          <Button type="button" variant="outline" onClick={onCancel}>
            キャンセル
          </Button>
        ) : null}
        <Button type="button" disabled={busy || errors.length > 0} onClick={submit}>
          {workflow ? '変更を保存' : '下書きを作成'}
        </Button>
      </div>
    </Card>
  )
}
