import { tenantURL } from '../../api'
import { Label } from '../../components/ui/label'
import { Input } from '../../components/ui/input'
import type { McpResourceServer } from '../../types'
import type { adminMcpResourceServersDictionary } from './AdminMcpResourceServersPage.i18n'

type Dictionary = (typeof adminMcpResourceServersDictionary)['en']

export type FormState = {
  resource: string
  name: string
  scopes: string
  state: McpResourceServer['state']
}

export const emptyForm: FormState = {
  resource: '',
  name: '',
  scopes: '',
  state: 'Active',
}

export function toForm(resourceServer: McpResourceServer): FormState {
  return {
    resource: resourceServer.resource,
    name: resourceServer.name,
    scopes: resourceServer.scopes.join(' '),
    state: resourceServer.state,
  }
}

export function parseScopes(scopes: string): string[] {
  return scopes.split(/[\s,]+/).filter(Boolean)
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
