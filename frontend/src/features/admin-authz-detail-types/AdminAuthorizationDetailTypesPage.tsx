import { IconPlus, IconShieldCheck, IconTrash, IconX } from '@tabler/icons-react'
import { type FormEvent, useState } from 'react'
import {
  AuthenticationAPIError,
  createAuthorizationDetailType,
  deleteAuthorizationDetailType,
  tenantURL,
  updateAuthorizationDetailType,
} from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Toast } from '../../components/ui/toast'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { useDictionary } from '../../lib/i18n'
import type { AuthorizationDetailType } from '../../types'
import { adminAuthorizationDetailTypesDictionary } from './AdminAuthorizationDetailTypesPage.i18n'

type FormState = {
  type: string
  description: string
  displayTemplate: string
  state: AuthorizationDetailType['state']
  schemaJSON: string
}

const sampleSchema = `{
  "rules": [
    { "name": "actions", "semantics": "set", "required": true, "allowed": ["read", "write"] },
    { "name": "datatypes", "semantics": "set", "required": true }
  ]
}`

const emptyForm: FormState = {
  type: '',
  description: '',
  displayTemplate: '',
  state: 'Enabled',
  schemaJSON: sampleSchema,
}

function toForm(t: AuthorizationDetailType): FormState {
  return {
    type: t.type,
    description: t.description ?? '',
    displayTemplate: t.display_template,
    state: t.state,
    schemaJSON: JSON.stringify(t.schema, null, 2),
  }
}

export function AdminAuthorizationDetailTypesPage({
  csrfToken,
  actorUsername,
  types,
}: {
  csrfToken: string
  actorUsername?: string
  types: AuthorizationDetailType[]
}) {
  const [items, setItems] = useState(types)
  const [editing, setEditing] = useState<string | null>(null)
  const [form, setForm] = useState<FormState>(emptyForm)
  const [creating, setCreating] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const dict = useDictionary(adminAuthorizationDetailTypesDictionary)

  function reset() {
    setEditing(null)
    setCreating(false)
    setForm(emptyForm)
    setError('')
  }

  async function handleSubmit(event: FormEvent) {
    event.preventDefault()
    setError('')
    let schema: AuthorizationDetailType['schema']
    try {
      schema = JSON.parse(form.schemaJSON)
    } catch {
      setError(dict.schemaInvalidError)
      return
    }
    const input = {
      type: form.type,
      description: form.description,
      display_template: form.displayTemplate,
      state: form.state,
      schema,
    }
    try {
      if (editing) {
        const updated = await updateAuthorizationDetailType(csrfToken, editing, input)
        setItems((prev) => prev.map((item) => (item.type === editing ? updated : item)))
        setNotice(dict.updatedNotice.replace('{type}', editing))
      } else {
        const created = await createAuthorizationDetailType(csrfToken, input)
        setItems((prev) => [...prev, created].sort((a, b) => a.type.localeCompare(b.type)))
        setNotice(dict.registeredNotice.replace('{type}', created.type))
      }
      reset()
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : dict.saveFailedError)
    }
  }

  async function handleDelete(detailType: string) {
    setError('')
    try {
      await deleteAuthorizationDetailType(csrfToken, detailType)
      setItems((prev) => prev.filter((item) => item.type !== detailType))
      setNotice(dict.deletedNotice.replace('{type}', detailType))
      if (editing === detailType) reset()
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : dict.deleteFailedError)
    }
  }

  const showForm = creating || editing !== null

  return (
    <AdminShell
      active="authz-detail-types"
      actorUsername={actorUsername}
      title={dict.pageTitle}
      description={dict.pageDescription}
      actions={
        showForm ? undefined : (
          <Button
            type="button"
            onClick={() => {
              setCreating(true)
              setForm(emptyForm)
            }}
          >
            <IconPlus size={17} aria-hidden="true" />
            {dict.registerType}
          </Button>
        )
      }
    >
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      <Toast message={notice} onDismiss={() => setNotice('')} />

      {showForm ? (
        <Card className="p-4">
          <form className="flex flex-col gap-4" onSubmit={handleSubmit}>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="type">{dict.typeIdLabel}</Label>
              <Input
                id="type"
                value={form.type}
                disabled={editing !== null}
                placeholder="payment_initiation"
                onChange={(e) => setForm({ ...form, type: e.target.value })}
                required
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="description">{dict.descriptionLabel}</Label>
              <Input
                id="description"
                value={form.description}
                onChange={(e) => setForm({ ...form, description: e.target.value })}
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="display_template">{dict.displayTemplateLabel}</Label>
              <Input
                id="display_template"
                value={form.displayTemplate}
                placeholder={dict.displayTemplatePlaceholder}
                onChange={(e) => setForm({ ...form, displayTemplate: e.target.value })}
                required
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="schema">{dict.schemaLabel}</Label>
              <textarea
                id="schema"
                value={form.schemaJSON}
                onChange={(e) => setForm({ ...form, schemaJSON: e.target.value })}
                rows={10}
                spellCheck={false}
                className="rounded-md border border-slate-300 bg-white p-2.5 font-mono text-xs leading-5 text-slate-900 focus:border-blue-500 focus:outline-none"
              />
              <p className="text-xs leading-5 text-slate-500">{dict.schemaHelp}</p>
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="state">{dict.stateLabel}</Label>
              <select
                id="state"
                value={form.state}
                onChange={(e) =>
                  setForm({ ...form, state: e.target.value as AuthorizationDetailType['state'] })
                }
                className="w-40 rounded-md border border-slate-300 bg-white p-2 text-sm text-slate-900 focus:border-blue-500 focus:outline-none"
              >
                <option value="Enabled">Enabled</option>
                <option value="Disabled">Disabled</option>
              </select>
            </div>
            <div className="flex gap-2.5">
              <Button type="submit">{editing ? dict.update : dict.register}</Button>
              <Button type="button" variant="ghost" onClick={reset}>
                <IconX size={17} aria-hidden="true" />
                {dict.cancel}
              </Button>
            </div>
          </form>
        </Card>
      ) : null}

      {items.length === 0 ? (
        <Card className="p-8 text-center text-sm text-slate-500">{dict.emptyNotice}</Card>
      ) : (
        <div className="flex flex-col gap-3">
          {items.map((item) => (
            <Card key={item.type} className="flex flex-col gap-3 p-4">
              <div className="flex items-start justify-between gap-3">
                <div className="min-w-0">
                  <div className="flex items-center gap-2">
                    <p className="font-mono text-sm font-semibold text-slate-900">{item.type}</p>
                    <span
                      className={
                        item.state === 'Enabled'
                          ? 'rounded-full bg-emerald-50 px-2 py-0.5 text-[0.68rem] font-bold text-emerald-700'
                          : 'rounded-full bg-slate-100 px-2 py-0.5 text-[0.68rem] font-bold text-slate-500'
                      }
                    >
                      {item.state}
                    </span>
                  </div>
                  {item.description ? (
                    <p className="mt-0.5 text-xs leading-5 text-slate-500">{item.description}</p>
                  ) : null}
                </div>
                <div className="flex shrink-0 gap-2">
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => {
                      setEditing(item.type)
                      setCreating(false)
                      setForm(toForm(item))
                    }}
                  >
                    {dict.edit}
                  </Button>
                  <Button type="button" variant="ghost" onClick={() => handleDelete(item.type)}>
                    <IconTrash size={16} aria-hidden="true" />
                  </Button>
                </div>
              </div>
              <div className="flex items-start gap-2 rounded-lg bg-slate-50 p-2.5 text-xs leading-5 text-slate-600">
                <IconShieldCheck
                  size={15}
                  className="mt-0.5 shrink-0 text-blue-600"
                  aria-hidden="true"
                />
                <div className="flex flex-wrap gap-1.5">
                  {item.schema.rules.map((rule) => (
                    <span key={rule.name} className="font-mono">
                      {rule.name}:{rule.semantics}
                      {rule.required ? '*' : ''}
                    </span>
                  ))}
                </div>
              </div>
            </Card>
          ))}
        </div>
      )}

      <p className="text-xs text-slate-400">
        <a className="underline" href={tenantURL('/admin/applications')}>
          {dict.footerLinkLabel}
        </a>{' '}
        {dict.footerText}
      </p>
    </AdminShell>
  )
}
