import { tenantURL } from '../../api'
import { Label } from '../../components/ui/label'
import { Input } from '../../components/ui/input'
import type { AuthorizationDetailType } from '../../types'
import type { adminAuthorizationDetailTypesDictionary } from './AdminAuthorizationDetailTypesPage.i18n'

type Dictionary = (typeof adminAuthorizationDetailTypesDictionary)['en']

export type FormState = {
  type: string
  description: string
  displayTemplate: string
  state: AuthorizationDetailType['state']
  schemaJSON: string
}

export const sampleSchema = `{
  "rules": [
    { "name": "actions", "semantics": "set", "required": true, "allowed": ["read", "write"] },
    { "name": "datatypes", "semantics": "set", "required": true }
  ]
}`

export const emptyForm: FormState = {
  type: '',
  description: '',
  displayTemplate: '',
  state: 'Enabled',
  schemaJSON: sampleSchema,
}

export function toForm(t: AuthorizationDetailType): FormState {
  return {
    type: t.type,
    description: t.description ?? '',
    displayTemplate: t.display_template,
    state: t.state,
    schemaJSON: JSON.stringify(t.schema, null, 2),
  }
}

export function listURL(): string {
  return tenantURL('/admin/authorization-detail-types')
}

export function editURL(type: string): string {
  return tenantURL(`/admin/authorization-detail-types/${encodeURIComponent(type)}/edit`)
}

export function newURL(): string {
  return tenantURL('/admin/authorization-detail-types/new')
}

// AuthorizationDetailTypeFormFields は作成/編集ページで共有するフォーム本体。
// type (ID) は編集時のみ変更不可にする (locked)。
export function AuthorizationDetailTypeFormFields({
  form,
  onChange,
  locked,
  t,
}: {
  form: FormState
  onChange: (next: FormState) => void
  locked: boolean
  t: Dictionary
}) {
  return (
    <>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="type">{t.typeIdLabel}</Label>
        <Input
          id="type"
          value={form.type}
          disabled={locked}
          placeholder="payment_initiation"
          onChange={(e) => onChange({ ...form, type: e.target.value })}
          required
        />
      </div>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="description">{t.descriptionLabel}</Label>
        <Input
          id="description"
          value={form.description}
          onChange={(e) => onChange({ ...form, description: e.target.value })}
        />
      </div>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="display_template">{t.displayTemplateLabel}</Label>
        <Input
          id="display_template"
          value={form.displayTemplate}
          placeholder={t.displayTemplatePlaceholder}
          onChange={(e) => onChange({ ...form, displayTemplate: e.target.value })}
          required
        />
      </div>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="schema">{t.schemaLabel}</Label>
        <textarea
          id="schema"
          value={form.schemaJSON}
          onChange={(e) => onChange({ ...form, schemaJSON: e.target.value })}
          rows={10}
          spellCheck={false}
          className="rounded-md border border-slate-300 bg-white p-2.5 font-mono text-xs leading-5 text-slate-900 focus:border-blue-500 focus:outline-none"
        />
        <p className="text-xs leading-5 text-slate-500">{t.schemaHelp}</p>
      </div>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="state">{t.stateLabel}</Label>
        <select
          id="state"
          value={form.state}
          onChange={(e) =>
            onChange({ ...form, state: e.target.value as AuthorizationDetailType['state'] })
          }
          className="w-40 rounded-md border border-slate-300 bg-white p-2 text-sm text-slate-900 focus:border-blue-500 focus:outline-none"
        >
          <option value="Enabled">Enabled</option>
          <option value="Disabled">Disabled</option>
        </select>
      </div>
    </>
  )
}
