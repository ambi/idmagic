import { tenantURL } from '../../api'
import { Label } from '../../components/ui/label'
import { Input } from '../../components/ui/input'
import type { McpResourceServer, ResourceAccessRule } from '../../types'
import type { adminMcpResourceServersDictionary } from './AdminMcpResourceServersPage.i18n'

type Dictionary = (typeof adminMcpResourceServersDictionary)['en']

export type FormState = {
  resource: string
  name: string
  scopes: string
  clientId: string
  accessRules: string
  state: McpResourceServer['state']
}

export const emptyForm: FormState = {
  resource: '',
  name: '',
  scopes: '',
  clientId: '',
  accessRules: '',
  state: 'Active',
}

export function toForm(resourceServer: McpResourceServer): FormState {
  const rules = resourceServer.access_rules ?? []
  return {
    resource: resourceServer.resource,
    name: resourceServer.name,
    scopes: resourceServer.scopes.join(' '),
    clientId: resourceServer.client_id ?? '',
    accessRules: rules.length > 0 ? JSON.stringify(rules, null, 2) : '',
    state: resourceServer.state,
  }
}

export function parseScopes(scopes: string): string[] {
  return scopes.split(/[\s,]+/).filter(Boolean)
}

// parseAccessRules は規則の入力を配列へ読む。項目の検証はサーバーが行い、ここでは保存を試みる
// 前に JSON の配列であることだけを確かめる。読めなければ null を返す。
export function parseAccessRules(text: string): ResourceAccessRule[] | null {
  if (text.trim() === '') return []
  try {
    const parsed: unknown = JSON.parse(text)
    return Array.isArray(parsed) ? (parsed as ResourceAccessRule[]) : null
  } catch {
    return null
  }
}

export function listURL(): string {
  return tenantURL('/admin/mcp-resource-servers')
}

export function editURL(resourceServerID: string): string {
  return tenantURL(`/admin/mcp-resource-servers/${encodeURIComponent(resourceServerID)}/edit`)
}

export function newURL(): string {
  return tenantURL('/admin/mcp-resource-servers/new')
}

// McpResourceServerFormFields は作成/編集ページで共有するフォーム本体。
// resource (URI) は編集時のみ変更不可にする (locked)。
export function McpResourceServerFormFields({
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
        <Label htmlFor="resource">{t.resourceLabel}</Label>
        <Input
          id="resource"
          type="url"
          value={form.resource}
          disabled={locked}
          placeholder={t.resourcePlaceholder}
          onChange={(event) => onChange({ ...form, resource: event.target.value })}
          required
        />
      </div>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="name">{t.nameLabel}</Label>
        <Input
          id="name"
          value={form.name}
          onChange={(event) => onChange({ ...form, name: event.target.value })}
          required
        />
      </div>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="scopes">{t.scopesLabel}</Label>
        <Input
          id="scopes"
          value={form.scopes}
          placeholder={t.scopesPlaceholder}
          onChange={(event) => onChange({ ...form, scopes: event.target.value })}
          required
        />
        <p className="text-xs leading-5 text-slate-500">{t.scopesHelp}</p>
      </div>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="client-id">{t.clientIdLabel}</Label>
        <Input
          id="client-id"
          value={form.clientId}
          onChange={(event) => onChange({ ...form, clientId: event.target.value })}
        />
        <p className="text-xs leading-5 text-slate-500">{t.clientIdHelp}</p>
      </div>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="access-rules">{t.accessRulesLabel}</Label>
        <textarea
          id="access-rules"
          value={form.accessRules}
          rows={6}
          placeholder={t.accessRulesPlaceholder}
          onChange={(event) => onChange({ ...form, accessRules: event.target.value })}
          className="rounded-md border border-slate-300 bg-white p-2 font-mono text-sm text-slate-900 focus:border-blue-500 focus:outline-none"
        />
        <p className="text-xs leading-5 text-slate-500">{t.accessRulesHelp}</p>
      </div>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="state">{t.stateLabel}</Label>
        <select
          id="state"
          value={form.state}
          onChange={(event) =>
            onChange({ ...form, state: event.target.value as McpResourceServer['state'] })
          }
          className="w-40 rounded-md border border-slate-300 bg-white p-2 text-sm text-slate-900 focus:border-blue-500 focus:outline-none"
        >
          <option value="Active">Active</option>
          <option value="Disabled">Disabled</option>
        </select>
      </div>
    </>
  )
}
