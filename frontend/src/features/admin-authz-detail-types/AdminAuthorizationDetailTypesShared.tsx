import { useMemo } from 'react'
import { tenantURL } from '../../api'
import { FieldError } from '../../components/FieldError'
import { Label } from '../../components/ui/label'
import { Input } from '../../components/ui/input'
import {
  type FormValidation,
  formSchema,
  requiredText,
  rule,
  useFormValidation,
} from '../../lib/form'
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
// 規則は oauth2 の authorizationDetailTypeSchema に従う。type と表示のテンプレートは必須で、
// スキーマは JSON のオブジェクトである。スキーマの中の規則はサーバーが検査する。
function detailTypeSchema(t: Dictionary) {
  return formSchema({
    type: requiredText(),
    displayTemplate: requiredText(),
    schemaJSON: rule(requiredText(), (value) =>
      parseSchemaObject(value) === undefined
        ? { kind: 'message', text: t.schemaInvalidError }
        : undefined,
    ),
  })
}

/** スキーマの入力を JSON のオブジェクトとして読む。読めないか、オブジェクトでなければ `undefined`。 */
export function parseSchemaObject(value: string): AuthorizationDetailType['schema'] | undefined {
  try {
    const parsed: unknown = JSON.parse(value)
    return typeof parsed === 'object' && parsed !== null && !Array.isArray(parsed)
      ? (parsed as AuthorizationDetailType['schema'])
      : undefined
  } catch {
    return undefined
  }
}

export type DetailTypeValidation = FormValidation<ReturnType<typeof detailTypeSchema>>

export function useDetailTypeValidation(form: FormState, t: Dictionary): DetailTypeValidation {
  const schema = useMemo(() => detailTypeSchema(t), [t])
  return useFormValidation(schema, {
    type: form.type,
    displayTemplate: form.displayTemplate,
    schemaJSON: form.schemaJSON,
  })
}

export function AuthorizationDetailTypeFormFields({
  form,
  validation,
  onChange,
  locked,
  t,
}: {
  form: FormState
  validation: DetailTypeValidation
  onChange: (next: FormState) => void
  locked: boolean
  t: Dictionary
}) {
  return (
    <>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="type" required>
          {t.typeIdLabel}
        </Label>
        <Input
          id="type"
          value={form.type}
          disabled={locked}
          placeholder="payment_initiation"
          onChange={(e) => onChange({ ...form, type: e.target.value })}
          required
          {...validation.fieldProps('type')}
        />
        <FieldError form={validation} name="type" />
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
        <Label htmlFor="display_template" required>
          {t.displayTemplateLabel}
        </Label>
        <Input
          id="display_template"
          value={form.displayTemplate}
          placeholder={t.displayTemplatePlaceholder}
          onChange={(e) => onChange({ ...form, displayTemplate: e.target.value })}
          required
          {...validation.fieldProps('displayTemplate')}
        />
        <FieldError form={validation} name="displayTemplate" />
      </div>
      <div className="flex flex-col gap-1.5 md:col-span-2">
        <Label htmlFor="schema" required>
          {t.schemaLabel}
        </Label>
        <textarea
          id="schema"
          value={form.schemaJSON}
          onChange={(e) => onChange({ ...form, schemaJSON: e.target.value })}
          rows={10}
          spellCheck={false}
          className="rounded-md border border-slate-300 bg-white p-2.5 font-mono text-xs leading-5 text-slate-900 focus:border-blue-500 focus:outline-none"
          {...validation.fieldProps('schemaJSON')}
        />
        <FieldError form={validation} name="schemaJSON" />
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
