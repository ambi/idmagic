import { IconArrowLeft } from '@tabler/icons-react'
import { Link } from '@tanstack/react-router'
import { useState } from 'react'
import { findAuthorizationModel, publishAuthorizationModel } from '../../api/admin'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Label } from '../../components/ui/label'
import { useDictionary } from '../../lib/i18n'
import type { AuthorizationModel, ResourceTypeDefinition } from '../../types'
import { adminAuthorizationDictionary } from './AdminAuthorizationPage.i18n'
import { authorizationErrorMessage } from './AdminAuthorizationShared'
import {
  diffModels,
  parseModelJson,
  toModelDraft,
  toResourceTypes,
  validateModelDraft,
  type ModelDiffEntry,
  type ModelDraft,
  type ModelDraftIssue,
  type ModelDraftIssueCode,
} from './authorizationModel'
import { ModelDraftForm } from './ModelDraftForm'

type Dictionary = (typeof adminAuthorizationDictionary)['en']

const ISSUE_MESSAGE_KEYS: Record<ModelDraftIssueCode, keyof Dictionary> = {
  no_resource_types: 'issueNoResourceTypes',
  invalid_name: 'issueInvalidName',
  duplicate_type: 'issueDuplicateType',
  duplicate_relation: 'issueDuplicateRelation',
  no_rewrites: 'issueNoRewrites',
  no_subject_forms: 'issueNoSubjectForms',
  self_reference: 'issueSelfReference',
  unknown_relation: 'issueUnknownRelation',
  unknown_subject: 'issueUnknownSubject',
  tupleset_without_object: 'issueTuplesetWithoutObject',
}

function issueMessage(issue: ModelDraftIssue, t: Dictionary): string {
  return t[ISSUE_MESSAGE_KEYS[issue.code]].replace('{subject}', issue.subject)
}

function diffLabel(entry: ModelDiffEntry, t: Dictionary): string {
  const template = { type: t.diffType, relation: t.diffRelation, rule: t.diffRule }[entry.target]
  return template
    .replace('{type}', entry.type)
    .replace('{relation}', entry.relation ?? '')
    .replace('{rule}', entry.rule ?? '')
}

function ModelDiffReview({
  entries,
  busy,
  onPublish,
  onBack,
}: {
  entries: ModelDiffEntry[]
  busy: boolean
  onPublish: () => void
  onBack: () => void
}) {
  const t = useDictionary(adminAuthorizationDictionary)
  const narrows = entries.some((entry) => entry.change === 'removed')
  return (
    <Card className="flex flex-col gap-3 p-4">
      <h2 className="font-semibold">{t.diffTitle}</h2>
      <p className="text-sm text-slate-600">{t.diffDescription}</p>
      {narrows ? <Alert variant="destructive">{t.diffNarrowingWarning}</Alert> : null}
      {entries.length === 0 ? <p className="text-sm text-slate-600">{t.diffEmpty}</p> : null}
      <ul className="flex flex-col gap-1 text-sm">
        {entries.map((entry) => {
          const label = diffLabel(entry, t)
          return (
            <li key={`${entry.change}-${label}`} className="flex gap-2">
              <span
                className={
                  entry.change === 'added'
                    ? 'font-medium text-emerald-700'
                    : 'font-medium text-red-700'
                }
              >
                {entry.change === 'added' ? t.diffAdded : t.diffRemoved}
              </span>
              <span className="break-all font-mono">{label}</span>
            </li>
          )
        })}
      </ul>
      <div className="flex gap-2.5">
        <Button disabled={busy || entries.length === 0} onClick={onPublish}>
          {busy ? t.publishing : t.publish}
        </Button>
        <Button variant="outline" disabled={busy} onClick={onBack}>
          {t.backToEdit}
        </Button>
      </div>
    </Card>
  )
}

export function AdminAuthorizationModelEditPage({
  csrfToken,
  actorUsername,
  model,
  onPublished,
}: {
  csrfToken: string
  actorUsername?: string
  model: AuthorizationModel | null
  onPublished: () => void
}) {
  const t = useDictionary(adminAuthorizationDictionary)
  const initialTypes = model?.resource_types ?? []
  const initialDraft = toModelDraft(initialTypes)
  const [mode, setMode] = useState<'form' | 'json'>(initialDraft ? 'form' : 'json')
  const [draft, setDraft] = useState<ModelDraft>(initialDraft ?? { resourceTypes: [] })
  const [json, setJson] = useState(JSON.stringify(initialTypes, null, 2))
  const [review, setReview] = useState<{
    resourceTypes: ResourceTypeDefinition[]
    entries: ModelDiffEntry[]
  } | null>(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  const current = mode === 'form' ? draft : parseModelJson(json)
  const issues = current ? validateModelDraft(current) : []
  const canReview = current !== null && issues.length === 0

  function switchMode(next: 'form' | 'json') {
    setReview(null)
    if (next === mode || !current) return
    if (next === 'json') setJson(JSON.stringify(toResourceTypes(current), null, 2))
    else setDraft(current)
    setMode(next)
  }

  async function run(action: () => Promise<void>) {
    setBusy(true)
    setError('')
    try {
      await action()
    } catch (cause) {
      setError(authorizationErrorMessage(cause, t))
    } finally {
      setBusy(false)
    }
  }

  function startReview() {
    if (!current) return
    const resourceTypes = toResourceTypes(current)
    // 画面を開いた後に別の管理者が発行したバージョンとの差分を見逃さないよう、確認の直前に読み直す。
    void run(async () => {
      const latest = await findAuthorizationModel()
      setReview({
        resourceTypes,
        entries: diffModels(latest?.authorization_model.resource_types ?? [], resourceTypes),
      })
    })
  }

  function publish() {
    if (!review) return
    void run(async () => {
      await publishAuthorizationModel(csrfToken, review.resourceTypes)
      onPublished()
    })
  }

  return (
    <AdminShell
      active="authorization"
      actorUsername={actorUsername}
      title={t.editTitle}
      description={t.pageDescription}
      actions={
        <Button
          variant="outline"
          nativeButton={false}
          render={<Link to="/admin/authorization/model" />}
        >
          <IconArrowLeft size={16} aria-hidden="true" />
          {t.backToModel}
        </Button>
      }
    >
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      <fieldset className="flex items-center gap-2" disabled={busy}>
        <legend className="sr-only">{t.modeLabel}</legend>
        <Button
          type="button"
          variant={mode === 'form' ? 'default' : 'outline'}
          aria-pressed={mode === 'form'}
          onClick={() => switchMode('form')}
        >
          {t.modeForm}
        </Button>
        <Button
          type="button"
          variant={mode === 'json' ? 'default' : 'outline'}
          aria-pressed={mode === 'json'}
          onClick={() => switchMode('json')}
        >
          {t.modeJson}
        </Button>
      </fieldset>
      {mode === 'form' ? (
        <ModelDraftForm
          draft={draft}
          onChange={(next) => {
            setDraft(next)
            setReview(null)
          }}
        />
      ) : (
        <Card className="flex flex-col gap-1.5 p-4">
          <Label htmlFor="authorization-model-json">{t.jsonLabel}</Label>
          <textarea
            id="authorization-model-json"
            className="min-h-96 rounded-md border border-slate-300 p-2 font-mono text-xs"
            aria-invalid={!current}
            spellCheck={false}
            value={json}
            onChange={(event) => {
              setJson(event.target.value)
              setReview(null)
            }}
          />
          {!current ? <Alert variant="destructive">{t.jsonInvalid}</Alert> : null}
        </Card>
      )}
      {issues.length > 0 ? (
        <ul className="list-disc pl-5 text-sm text-red-700">
          {issues.map((issue) => {
            const message = issueMessage(issue, t)
            return <li key={message}>{message}</li>
          })}
        </ul>
      ) : null}
      {review ? (
        <ModelDiffReview
          entries={review.entries}
          busy={busy}
          onPublish={publish}
          onBack={() => setReview(null)}
        />
      ) : (
        <div>
          <Button disabled={busy || !canReview} onClick={startReview}>
            {t.review}
          </Button>
        </div>
      )}
    </AdminShell>
  )
}
