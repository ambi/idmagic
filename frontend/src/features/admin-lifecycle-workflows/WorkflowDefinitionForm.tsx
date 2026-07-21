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
import { workflowFormDictionary, type WorkflowFormDictionary } from './WorkflowDefinitionForm.i18n'

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

const TRIGGER_KINDS: TriggerKind[] = [
  'user_created',
  'user_attributes_changed',
  'user_status_changed',
]

const ACTION_KINDS: WorkflowActionKind[] = [
  'add_group_member',
  'remove_group_member',
  'assign_application',
  'unassign_application',
  'set_required_action',
  'clear_required_action',
  'enable_user',
  'disable_user',
  'send_email',
]

const USER_STATUS_VALUES = [
  'active',
  'disabled',
  'pending_deletion',
  'deleted',
  'locked',
  'staged',
  'suspended',
] as const

const VISIBILITY_VALUES = ['visible', 'hidden'] as const

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

export function validateWorkflowDraft(draft: FormDraft, t: WorkflowFormDictionary): string[] {
  const errors: string[] = []
  if (!draft.name.trim()) errors.push(t.errNameRequired)
  if (draft.triggerKind === 'user_attributes_changed' && !draft.watchedAttributes.trim()) {
    errors.push(t.errWatchedAttributesRequired)
  }
  if (draft.triggerKind === 'user_status_changed') {
    if (!draft.fromStatus) errors.push(t.errFromStatusRequired)
    if (!draft.toStatus) errors.push(t.errToStatusRequired)
    if (draft.fromStatus && draft.fromStatus === draft.toStatus) {
      errors.push(t.errStatusSame)
    }
  }
  draft.actions.forEach((action, index) => {
    const at = (template: string) => template.replace('{index}', String(index + 1))
    if (!action.kind) errors.push(at(t.errActionKind))
    if (
      (action.kind === 'add_group_member' || action.kind === 'remove_group_member') &&
      !action.group_id
    ) {
      errors.push(at(t.errActionGroup))
    }
    if (
      (action.kind === 'assign_application' || action.kind === 'unassign_application') &&
      !action.application_id
    ) {
      errors.push(at(t.errActionApplication))
    }
    if (
      (action.kind === 'set_required_action' || action.kind === 'clear_required_action') &&
      !action.required_action
    ) {
      errors.push(at(t.errActionRequiredAction))
    }
    if (action.kind === 'send_email' && !action.template_key?.trim()) {
      errors.push(at(t.errActionTemplate))
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

export function workflowStatusLabel(
  status: AdminLifecycleWorkflow['status'],
  t: WorkflowFormDictionary,
): string {
  return t[`status_${status}` as const] ?? status
}

export function workflowTriggerLabel(kind: TriggerKind, t: WorkflowFormDictionary): string {
  return t[`trigger_${kind}` as const] ?? t.trigger_unknown
}

export function workflowActionLabel(kind: WorkflowActionKind, t: WorkflowFormDictionary): string {
  return t[`action_${kind}` as const] ?? t.action_unknown
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
  const t = useDictionary(workflowFormDictionary)
  const [draft, setDraft] = useState(() => workflowDraft(workflow))
  const tLabels = useDictionary(domainLabelsDictionary)
  useEffect(() => {
    setDraft(workflowDraft(workflow))
  }, [workflow])
  const errors = validateWorkflowDraft(draft, t)
  const triggerOptions: SelectOption[] = TRIGGER_KINDS.map((kind) => ({
    value: kind,
    label: t[`trigger_${kind}` as const],
  }))
  const actionOptions: SelectOption[] = ACTION_KINDS.map((kind) => ({
    value: kind,
    label: t[`action_${kind}` as const],
  }))
  const userStatusOptions: SelectOption[] = USER_STATUS_VALUES.map((value) => ({
    value,
    label: t[`userStatus_${value}` as const],
  }))
  const visibilityOptions: SelectOption[] = VISIBILITY_VALUES.map((value) => ({
    value,
    label: t[`visibility_${value}` as const],
  }))
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

  const aria = (template: string, index: number) => template.replace('{index}', String(index + 1))

  return (
    <Card className="mb-6 space-y-6 p-5">
      <div className="grid gap-4 md:grid-cols-2">
        <div className="space-y-2">
          <Label htmlFor="workflow-name">{t.nameLabel}</Label>
          <Input
            id="workflow-name"
            value={draft.name}
            maxLength={100}
            onChange={(event) => setDraft({ ...draft, name: event.target.value })}
            placeholder={t.namePlaceholder}
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="workflow-description">{t.descriptionLabel}</Label>
          <Input
            id="workflow-description"
            value={draft.description}
            maxLength={500}
            onChange={(event) => setDraft({ ...draft, description: event.target.value })}
            placeholder={t.descriptionPlaceholder}
          />
        </div>
      </div>

      <section className="rounded-lg border border-slate-200 p-4">
        <Label>{t.triggerSectionLabel}</Label>
        <div className="mt-2 max-w-xl">
          <Select
            aria-label={t.triggerSelectAria}
            value={draft.triggerKind}
            options={triggerOptions}
            onValueChange={(value) => setDraft({ ...draft, triggerKind: value as TriggerKind })}
            className="w-full"
          />
          <p className="mt-2 text-sm text-slate-600">
            {t[`triggerDesc_${draft.triggerKind}` as const]}
          </p>
        </div>
        {draft.triggerKind === 'user_attributes_changed' ? (
          <div className="mt-4 max-w-xl space-y-2">
            <Label htmlFor="workflow-attributes">{t.watchedAttributesLabel}</Label>
            <Input
              id="workflow-attributes"
              value={draft.watchedAttributes}
              onChange={(event) => setDraft({ ...draft, watchedAttributes: event.target.value })}
              placeholder={t.watchedAttributesPlaceholder}
            />
            <p className="text-xs text-slate-500">{t.watchedAttributesHelp}</p>
          </div>
        ) : null}
        {draft.triggerKind === 'user_status_changed' ? (
          <div className="mt-4 grid max-w-2xl gap-4 md:grid-cols-2">
            <div className="space-y-2">
              <Label>{t.fromStatusLabel}</Label>
              <Select
                aria-label={t.fromStatusAria}
                value={draft.fromStatus}
                options={userStatusOptions}
                placeholder={t.statusPlaceholder}
                onValueChange={(value) => setDraft({ ...draft, fromStatus: value })}
                className="w-full"
              />
            </div>
            <div className="space-y-2">
              <Label>{t.toStatusLabel}</Label>
              <Select
                aria-label={t.toStatusAria}
                value={draft.toStatus}
                options={userStatusOptions}
                placeholder={t.statusPlaceholder}
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
            <Label>{t.actionsSectionLabel}</Label>
            <p className="mt-1 text-sm text-slate-600">{t.actionsHelp}</p>
          </div>
          <Button
            type="button"
            variant="outline"
            disabled={draft.actions.length >= 20}
            onClick={() => setDraft({ ...draft, actions: [...draft.actions, emptyAction()] })}
          >
            {t.addAction}
          </Button>
        </div>
        {draft.actions.map((action, index) => (
          <Card className="space-y-4 p-4" key={action.key}>
            <div className="flex flex-wrap items-center justify-between gap-2">
              <span className="font-semibold">
                {index + 1}. {action.kind ? workflowActionLabel(action.kind, t) : t.actionKindUnset}
              </span>
              <div className="flex gap-2">
                <Button
                  type="button"
                  variant="outline"
                  disabled={index === 0}
                  aria-label={aria(t.ariaMoveUp, index)}
                  onClick={() => moveAction(index, -1)}
                >
                  {t.moveUp}
                </Button>
                <Button
                  type="button"
                  variant="outline"
                  disabled={index === draft.actions.length - 1}
                  aria-label={aria(t.ariaMoveDown, index)}
                  onClick={() => moveAction(index, 1)}
                >
                  {t.moveDown}
                </Button>
                <Button
                  type="button"
                  variant="outline"
                  disabled={draft.actions.length === 1}
                  aria-label={aria(t.ariaRemove, index)}
                  onClick={() =>
                    setDraft({
                      ...draft,
                      actions: draft.actions.filter((_, actionIndex) => actionIndex !== index),
                    })
                  }
                >
                  {t.remove}
                </Button>
              </div>
            </div>
            <div className="max-w-xl">
              <Select
                aria-label={aria(t.ariaActionKind, index)}
                value={action.kind}
                options={actionOptions}
                placeholder={t.actionKindPlaceholder}
                onValueChange={(value) =>
                  updateAction(index, { kind: value as WorkflowActionKind })
                }
                className="w-full"
              />
              {action.kind ? (
                <p className="mt-2 text-sm text-slate-600">
                  {t[`actionDesc_${action.kind}` as const]}
                </p>
              ) : null}
            </div>
            {action.kind === 'add_group_member' || action.kind === 'remove_group_member' ? (
              <div className="max-w-xl space-y-2">
                <Label>{t.groupLabel}</Label>
                <Select
                  aria-label={aria(t.ariaActionGroup, index)}
                  value={action.group_id ?? ''}
                  options={groupOptions}
                  placeholder={groupOptions.length ? t.groupPlaceholder : t.groupEmptyPlaceholder}
                  onValueChange={(value) => updateAction(index, { group_id: value })}
                  className="w-full"
                />
                {!groupOptions.length ? (
                  <p className="text-sm text-red-700">{t.groupEmptyError}</p>
                ) : null}
              </div>
            ) : null}
            {action.kind === 'assign_application' || action.kind === 'unassign_application' ? (
              <div className="grid max-w-2xl gap-4 md:grid-cols-2">
                <div className="space-y-2">
                  <Label>{t.applicationLabel}</Label>
                  <Select
                    aria-label={aria(t.ariaActionApplication, index)}
                    value={action.application_id ?? ''}
                    options={applicationOptions}
                    placeholder={
                      applicationOptions.length
                        ? t.applicationPlaceholder
                        : t.applicationEmptyPlaceholder
                    }
                    onValueChange={(value) => updateAction(index, { application_id: value })}
                    className="w-full"
                  />
                  {!applicationOptions.length ? (
                    <p className="text-sm text-red-700">{t.applicationEmptyError}</p>
                  ) : null}
                </div>
                {action.kind === 'assign_application' ? (
                  <div className="space-y-2">
                    <Label>{t.visibilityLabel}</Label>
                    <Select
                      aria-label={aria(t.ariaActionVisibility, index)}
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
                <Label>{t.requiredActionLabel}</Label>
                <Select
                  aria-label={aria(t.ariaActionRequiredAction, index)}
                  value={action.required_action ?? ''}
                  options={requiredActionOptions}
                  placeholder={t.requiredActionPlaceholder}
                  onValueChange={(value) => updateAction(index, { required_action: value })}
                  className="w-full"
                />
              </div>
            ) : null}
            {action.kind === 'enable_user' || action.kind === 'disable_user' ? (
              <div className="max-w-xl space-y-2">
                <Label htmlFor={`workflow-action-reason-${index}`}>{t.reasonLabel}</Label>
                <Input
                  id={`workflow-action-reason-${index}`}
                  value={action.reason ?? ''}
                  onChange={(event) => updateAction(index, { reason: event.target.value })}
                  placeholder={t.reasonPlaceholder}
                />
              </div>
            ) : null}
            {action.kind === 'send_email' ? (
              <div className="max-w-xl space-y-2">
                <Label htmlFor={`workflow-template-${index}`}>{t.templateLabel}</Label>
                <Input
                  id={`workflow-template-${index}`}
                  value={action.template_key ?? ''}
                  onChange={(event) => updateAction(index, { template_key: event.target.value })}
                  placeholder={t.templatePlaceholder}
                />
                <p className="text-xs text-slate-500">{t.templateHelp}</p>
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
          <p className="font-semibold">{t.errorsTitle}</p>
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
            {t.cancel}
          </Button>
        ) : null}
        <Button type="button" disabled={busy || errors.length > 0} onClick={submit}>
          {workflow ? t.save : t.createDraft}
        </Button>
      </div>
    </Card>
  )
}
