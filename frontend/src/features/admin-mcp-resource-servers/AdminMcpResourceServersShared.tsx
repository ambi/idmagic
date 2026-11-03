import { tenantURL } from '../../api'
import { Button } from '../../components/ui/button'
import { Label } from '../../components/ui/label'
import { Input } from '../../components/ui/input'
import type { McpResourceServer, ResourceAccessRule } from '../../types'
import { MCP_OPERATION_TYPE } from '../../lib/mcpAccess'
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

// AccessRuleRows は規則の基本の項目を行で編集する。JSON を正とし、行にない項目（条件や上限など）は保つ。
function AccessRuleRows({
  form,
  onChange,
  t,
}: {
  form: FormState
  onChange: (next: FormState) => void
  t: Dictionary
}) {
  const rules = parseAccessRules(form.accessRules)
  if (
    rules === null ||
    !rules.every(
      (rule) =>
        typeof rule === 'object' &&
        rule !== null &&
        (rule.resource_type === undefined || typeof rule.resource_type === 'string') &&
        (rule.relation === undefined || typeof rule.relation === 'string') &&
        (rule.scopes === undefined ||
          (Array.isArray(rule.scopes) && rule.scopes.every((scope) => typeof scope === 'string'))),
    )
  )
    return <p className="text-sm text-slate-600">{t.accessRulesUnreadable}</p>
  const scopes = parseScopes(form.scopes)
  const save = (next: ResourceAccessRule[]) =>
    onChange({ ...form, accessRules: next.length > 0 ? JSON.stringify(next, null, 2) : '' })
  const update = (index: number, change: Partial<ResourceAccessRule>) =>
    save(rules.map((rule, position) => (position === index ? { ...rule, ...change } : rule)))
  return (
    <div className="flex flex-col gap-2">
      {rules.map((rule, index) => (
        // 行は内容で識別できず、並べ替えもない。
        // biome-ignore lint/suspicious/noArrayIndexKey: 位置だけが行を識別する。
        <div key={index} className="flex flex-wrap items-end gap-3 rounded-md border p-3">
          {rule.tool ? (
            <>
              <div className="flex flex-col gap-1">
                <Label htmlFor={`tool-name-${index}`}>{t.toolName}</Label>
                <Input
                  id={`tool-name-${index}`}
                  value={rule.tool.name}
                  onChange={(event) =>
                    update(index, { tool: { ...rule.tool!, name: event.target.value } })
                  }
                />
              </div>
              <div className="flex flex-col gap-1">
                <Label htmlFor={`tool-label-${index}`}>{t.toolDisplayName}</Label>
                <Input
                  id={`tool-label-${index}`}
                  value={rule.tool.display_name}
                  onChange={(event) =>
                    update(index, { tool: { ...rule.tool!, display_name: event.target.value } })
                  }
                />
              </div>
              <label className="flex items-center gap-2 text-sm">
                <input
                  type="checkbox"
                  checked={rule.resource_type !== MCP_OPERATION_TYPE}
                  onChange={(event) =>
                    update(
                      index,
                      event.target.checked
                        ? {
                            resource_type: 'document',
                            relation: 'viewer',
                            tool: { ...rule.tool!, resource_id_argument: 'document_id' },
                          }
                        : {
                            resource_type: MCP_OPERATION_TYPE,
                            relation: 'invoke',
                            tool: { name: rule.tool!.name, display_name: rule.tool!.display_name },
                          },
                    )
                  }
                />
                {t.toolTarget}
              </label>
              {rule.resource_type !== MCP_OPERATION_TYPE ? (
                <div className="flex flex-col gap-1">
                  <Label htmlFor={`tool-argument-${index}`}>{t.toolArgument}</Label>
                  <Input
                    id={`tool-argument-${index}`}
                    value={rule.tool.resource_id_argument ?? ''}
                    onChange={(event) =>
                      update(index, {
                        tool: { ...rule.tool!, resource_id_argument: event.target.value },
                      })
                    }
                  />
                </div>
              ) : null}
            </>
          ) : null}
          {!rule.tool || rule.resource_type !== MCP_OPERATION_TYPE ? (
            <>
              <div className="flex flex-col gap-1">
                <Label htmlFor={`rule-type-${index}`}>{t.ruleResourceType}</Label>
                <Input
                  id={`rule-type-${index}`}
                  value={rule.resource_type}
                  placeholder="document"
                  onChange={(event) => update(index, { resource_type: event.target.value })}
                />
              </div>
              <div className="flex flex-col gap-1">
                <Label htmlFor={`rule-relation-${index}`}>{t.ruleRelation}</Label>
                <Input
                  id={`rule-relation-${index}`}
                  value={rule.relation}
                  placeholder="viewer"
                  onChange={(event) => update(index, { relation: event.target.value })}
                />
              </div>
            </>
          ) : null}
          <fieldset className="flex flex-col gap-1">
            <legend className="text-sm font-medium">{t.ruleScopes}</legend>
            {scopes.length === 0 ? (
              <span className="text-xs text-slate-500">{t.ruleScopesEmpty}</span>
            ) : null}
            {scopes.map((name) => (
              <label key={name} className="flex items-center gap-2 text-sm">
                <input
                  type="checkbox"
                  checked={rule.scopes?.includes(name) ?? false}
                  onChange={(event) =>
                    update(index, {
                      scopes: event.target.checked
                        ? [...(rule.scopes ?? []), name]
                        : (rule.scopes ?? []).filter((item) => item !== name),
                    })
                  }
                />
                <code className="text-xs">{name}</code>
              </label>
            ))}
          </fieldset>
          <Button
            type="button"
            variant="outline"
            onClick={() => save(rules.filter((_, position) => position !== index))}
          >
            {t.ruleRemove}
          </Button>
        </div>
      ))}
      <div>
        <Button
          type="button"
          variant="outline"
          onClick={() =>
            save([
              ...rules,
              {
                resource_type: MCP_OPERATION_TYPE,
                relation: 'invoke',
                scopes,
                tool: { name: '', display_name: '' },
              },
            ])
          }
        >
          {t.ruleAdd}
        </Button>
        <details className="mt-3">
          <summary className="cursor-pointer text-sm">{t.legacyRuleAdd}</summary>
          <Button
            type="button"
            variant="outline"
            onClick={() => save([...rules, { resource_type: '', relation: '', scopes }])}
          >
            {t.legacyRuleAdd}
          </Button>
        </details>
      </div>
    </div>
  )
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
        <Label htmlFor="resource" required>
          {t.resourceLabel}
        </Label>
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
        <Label htmlFor="name" required>
          {t.nameLabel}
        </Label>
        <Input
          id="name"
          value={form.name}
          onChange={(event) => onChange({ ...form, name: event.target.value })}
          required
        />
      </div>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="scopes" required>
          {t.scopesLabel}
        </Label>
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
      <div className="flex flex-col gap-2 md:col-span-2">
        <p className="text-sm font-medium">{t.operationsHeading}</p>
        <p className="text-xs leading-5 text-slate-500">{t.accessRulesHelp}</p>
        <AccessRuleRows form={form} onChange={onChange} t={t} />
        <details className="text-sm">
          <summary className="cursor-pointer text-slate-600">{t.accessRulesAdvanced}</summary>
          <div className="mt-2 flex flex-col gap-1.5">
            <Label htmlFor="access-rules">{t.accessRulesLabel}</Label>
            <textarea
              id="access-rules"
              value={form.accessRules}
              rows={6}
              placeholder={t.accessRulesPlaceholder}
              onChange={(event) => onChange({ ...form, accessRules: event.target.value })}
              className="rounded-md border border-slate-300 bg-white p-2 font-mono text-sm text-slate-900 focus:border-blue-500 focus:outline-none"
            />
          </div>
        </details>
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
