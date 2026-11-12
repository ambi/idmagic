import { useEffect, useState } from 'react'
import { getMcpToolCatalog, listAdminApplications, tenantURL } from '../../api'
import { FieldError } from '../../components/FieldError'
import { Button } from '../../components/ui/button'
import { Label } from '../../components/ui/label'
import { Input } from '../../components/ui/input'
import {
  type FormValidation,
  formSchema,
  requiredList,
  requiredText,
  requiredUrl,
  rule,
  text,
} from '../../lib/form'
import { LENGTH } from '../../lib/lengthLimits'
import type {
  AdminApplication,
  McpResourceServer,
  CatalogTool,
  ToolCatalogChanges,
  ToolCatalog,
  ResourceAccessRule,
} from '../../types'
import { MCP_OPERATION_TYPE } from '../../lib/mcpAccess'
import { useDictionary } from '../../lib/i18n'
import { adminMcpResourceServersDictionary } from './AdminMcpResourceServersPage.i18n'
import { AccessRuleConstraints } from './AccessRuleConstraints'
import { accessRuleProblems, constraintsRepresentable, hasProblems } from './accessRuleProblems'

type Dictionary = (typeof adminMcpResourceServersDictionary)['en']

export type FormState = {
  resource: string
  name: string
  scopes: string
  clientId: string
  accessRules: string
  state: McpResourceServer['state']
}

// NEW_EVALUATOR は、保存と同時に評価用のクライアントを作成することを表す選択肢の値である。
export const NEW_EVALUATOR = '__new__'

// DEFAULT_SCOPE は、Gateway が執行する構成で IdMagic の中だけに使うスコープである。
// サーバーも省略時に同じ値を補う (REQ-OAUTH2-064)。
export const DEFAULT_SCOPE = 'mcp'

export const emptyForm: FormState = {
  resource: '',
  name: '',
  scopes: DEFAULT_SCOPE,
  clientId: NEW_EVALUATOR,
  accessRules: '',
  state: 'Active',
}

// publishCatalogTool はカタログのツールを、名前と表示名を入れた公開の行として足す。
// 必要なスコープは登録のスコープをすべて選ぶ。既に公開中なら何もしない。
export function publishCatalogTool(form: FormState, tool: CatalogTool): FormState {
  const rules = parseAccessRules(form.accessRules) ?? []
  if (rules.some((rule) => rule.tool?.name === tool.name)) return form
  const next: ResourceAccessRule[] = [
    ...rules,
    {
      resource_type: MCP_OPERATION_TYPE,
      relation: 'invoke',
      scopes: parseScopes(form.scopes),
      tool: { name: tool.name, display_name: tool.title?.trim() || tool.name },
    },
  ]
  return { ...form, accessRules: JSON.stringify(next, null, 2) }
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

// 規則は oauth2/domain の mcpResourceServerSchema に従う。resource は RFC 8707 のリソース指示子
// （フラグメントのない絶対 URI）、名前は表示名の上限、スコープは 1 件以上である。
export function resourceServerSchema(t: Dictionary) {
  return formSchema({
    resource: requiredUrl({ noFragment: true }),
    name: requiredText(LENGTH.displayName),
    scopes: requiredList(parseScopes),
    accessRules: rule(text(), (value) => {
      const rules = parseAccessRules(value)
      if (rules === null) return { kind: 'message', text: t.accessRulesInvalidError }
      // 行の欄の問題は行に示し、ここでは保存を止めるだけにする (REQ-OAUTH2-090)。
      const rowProblem = rules.some(
        (rule) =>
          typeof rule === 'object' &&
          rule !== null &&
          constraintsRepresentable(rule) &&
          hasProblems(accessRuleProblems(rule)),
      )
      return rowProblem ? { kind: 'message', text: t.accessRulesRowsInvalid } : undefined
    }),
  })
}

export type ResourceServerValidation = FormValidation<ReturnType<typeof resourceServerSchema>>

export function listURL(): string {
  return tenantURL('/admin/mcp-resource-servers')
}

export function editURL(resourceServerID: string): string {
  return tenantURL(`/admin/mcp-resource-servers/${encodeURIComponent(resourceServerID)}/edit`)
}

export function newURL(): string {
  return tenantURL('/admin/mcp-resource-servers/new')
}

// targetScoped は、対象 ID を読む引数を持つツールの行かを返す。従来の型と関係の行も対象別である。
function targetScoped(rule: ResourceAccessRule): boolean {
  return (
    rule.resource_type !== MCP_OPERATION_TYPE ||
    (rule.tool !== undefined && 'resource_id_argument' in rule.tool)
  )
}

export function McpResourceServerStatus({ state }: { state: McpResourceServer['state'] }) {
  const t = useDictionary(adminMcpResourceServersDictionary)
  return (
    <span
      className={
        state === 'Active'
          ? 'rounded-full bg-emerald-50 px-2 py-0.5 text-[0.68rem] font-bold text-emerald-700'
          : 'rounded-full bg-slate-100 px-2 py-0.5 text-[0.68rem] font-bold text-slate-500'
      }
    >
      {state === 'Active' ? t.statusActive : t.statusDisabled}
    </span>
  )
}

// AccessRuleRows は規則を行で編集する。JSON を正とし、行にない項目（操作の識別子など）は保つ。
function AccessRuleRows({
  form,
  onChange,
  showProblems,
  t,
}: {
  form: FormState
  onChange: (next: FormState) => void
  showProblems: boolean
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
                  aria-describedby={`tool-target-help-${index}`}
                  checked={targetScoped(rule)}
                  onChange={(event) =>
                    update(
                      index,
                      event.target.checked
                        ? // 組み込みの型のまま対象引数を持たせ、範囲は割当で選ぶ (REQ-OAUTH2-083)。
                          { tool: { ...rule.tool!, resource_id_argument: 'document_id' } }
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
              <p
                id={`tool-target-help-${index}`}
                className="w-full text-xs leading-5 text-slate-500"
              >
                {t.toolTargetHelp}
              </p>
              {targetScoped(rule) ? (
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
          {/* スコープが一つなら選ぶものがないので、行ごとの選択を出さない。 */}
          <fieldset className="flex flex-col gap-1" hidden={scopes.length === 1}>
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
          <AccessRuleConstraints
            rule={rule}
            index={index}
            showProblems={showProblems}
            onChange={(change) => update(index, change)}
            t={t}
          />
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
  validation,
  onChange,
  locked,
  t,
}: {
  form: FormState
  validation: ResourceServerValidation
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
          {...validation.fieldProps('resource')}
        />
        <FieldError form={validation} name="resource" />
        <p className="text-xs leading-5 text-slate-500">{t.resourceHelp}</p>
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
          {...validation.fieldProps('name')}
        />
        <FieldError form={validation} name="name" />
      </div>
      <EvaluatorClientField form={form} onChange={onChange} allowCreate={!locked} t={t} />
      <details className="text-sm md:col-span-2" open={form.scopes.trim() !== DEFAULT_SCOPE}>
        <summary className="cursor-pointer text-slate-600">{t.scopesAdvanced}</summary>
        <div className="mt-2 flex flex-col gap-1.5">
          <Label htmlFor="scopes" required>
            {t.scopesLabel}
          </Label>
          <Input
            id="scopes"
            value={form.scopes}
            placeholder={t.scopesPlaceholder}
            onChange={(event) => onChange({ ...form, scopes: event.target.value })}
            required
            {...validation.fieldProps('scopes')}
          />
          <FieldError form={validation} name="scopes" />
          <p className="text-xs leading-5 text-slate-500">{t.scopesHelp}</p>
        </div>
      </details>
      <div className="flex flex-col gap-2 md:col-span-2">
        <p className="text-sm font-medium">{t.operationsHeading}</p>
        <p className="text-xs leading-5 text-slate-500">{t.accessRulesHelp}</p>
        <AccessRuleRows
          form={form}
          onChange={onChange}
          showProblems={validation.problem('accessRules') !== undefined}
          t={t}
        />
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
              {...validation.fieldProps('accessRules')}
            />
            <FieldError form={validation} name="accessRules" />
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
          <option value="Active">{t.statusActive}</option>
          <option value="Disabled">{t.statusDisabled}</option>
        </select>
      </div>
    </>
  )
}

// EvaluatorClientField は評価用のクライアントを、新規作成、既存のサービスの名前、なしから選ばせる。
// client_id を手で写させない (REQ-OAUTH2-088)。
function EvaluatorClientField({
  form,
  onChange,
  allowCreate,
  t,
}: {
  form: FormState
  onChange: (next: FormState) => void
  allowCreate: boolean
  t: Dictionary
}) {
  const [services, setServices] = useState<AdminApplication[]>([])
  useEffect(() => {
    let active = true
    listAdminApplications()
      .then((applications) => {
        if (active) setServices(applications.filter((app) => app.kind === 'service'))
      })
      // 一覧を読めなくても、新規作成と現在の値は選べる。
      .catch(() => undefined)
    return () => {
      active = false
    }
  }, [])
  const listed = services.filter((app) => app.protocol?.client_id)
  const unlisted =
    form.clientId !== '' &&
    form.clientId !== NEW_EVALUATOR &&
    !listed.some((app) => app.protocol?.client_id === form.clientId)
  return (
    <div className="flex flex-col gap-1.5 md:col-span-2">
      <Label htmlFor="evaluator-client">{t.evaluatorLabel}</Label>
      <select
        id="evaluator-client"
        value={form.clientId}
        onChange={(event) => onChange({ ...form, clientId: event.target.value })}
        className="max-w-md rounded-md border border-slate-300 bg-white p-2 text-sm text-slate-900 focus:border-blue-500 focus:outline-none"
      >
        {allowCreate ? <option value={NEW_EVALUATOR}>{t.evaluatorNew}</option> : null}
        {listed.map((app) => (
          <option key={app.id} value={app.protocol!.client_id}>
            {app.name}
          </option>
        ))}
        {unlisted ? (
          <option value={form.clientId}>
            {t.evaluatorUnlisted.replace('{clientId}', form.clientId)}
          </option>
        ) : null}
        <option value="">{t.evaluatorNone}</option>
      </select>
      <p className="text-xs leading-5 text-slate-500">{t.evaluatorHelp}</p>
    </div>
  )
}

// ToolCatalogPanel は Gateway がカタログのツールを、公開中のツールと区別して示す (REQ-OAUTH2-087)。
// 「公開する」は編集中のフォームへ行を足すだけで、保存するまで何も公開しない。
export function ToolCatalogPanel({
  resourceServerID,
  form,
  onChange,
  t,
}: {
  resourceServerID: string
  form: FormState
  onChange: (next: FormState) => void
  t: Dictionary
}) {
  const [catalog, setCatalog] = useState<ToolCatalog | null>(null)
  const [failed, setFailed] = useState(false)
  useEffect(() => {
    let active = true
    getMcpToolCatalog(resourceServerID)
      .then((value) => {
        if (active) setCatalog(value)
      })
      .catch(() => {
        if (active) setFailed(true)
      })
    return () => {
      active = false
    }
  }, [resourceServerID])
  const published = new Set(
    (parseAccessRules(form.accessRules) ?? []).flatMap((rule) =>
      rule.tool ? [rule.tool.name] : [],
    ),
  )
  const catalogNames = new Set(catalog?.tools.map((tool) => tool.name) ?? [])
  const missing = catalog?.complete ? [...published].filter((name) => !catalogNames.has(name)) : []
  return (
    <section className="flex flex-col gap-2 md:col-span-2" aria-labelledby="tool-catalog-heading">
      <p id="tool-catalog-heading" className="text-sm font-medium">
        {t.catalogHeading}
      </p>
      {failed ? <p className="text-sm text-red-700">{t.catalogLoadFailed}</p> : null}
      {catalog && !catalog.received_at ? (
        <p className="text-xs leading-5 text-slate-500">{t.catalogNone}</p>
      ) : null}
      {catalog?.received_at ? (
        <p className="text-xs leading-5 text-slate-500">
          {t.catalogAt.replace('{time}', new Date(catalog.received_at).toLocaleString())}
          {catalog.complete ? '' : ` ${t.catalogIncomplete}`}
        </p>
      ) : null}
      {catalog && catalog.tools.length > 0 ? (
        <ul className="flex flex-col gap-2">
          {catalog.tools.map((tool) => (
            <li
              key={tool.name}
              className="flex flex-wrap items-center justify-between gap-2 rounded-md border p-2"
            >
              <span className="flex flex-col">
                <span className="text-sm font-medium">{tool.title || tool.name}</span>
                <code className="text-xs text-slate-600">{tool.name}</code>
                {tool.description ? (
                  <span className="text-xs text-slate-600">{tool.description}</span>
                ) : null}
              </span>
              {published.has(tool.name) ? (
                <span className="text-xs text-slate-600">{t.catalogPublished}</span>
              ) : (
                <Button
                  type="button"
                  variant="outline"
                  aria-label={`${t.catalogPublish} ${tool.name}`}
                  onClick={() => onChange(publishCatalogTool(form, tool))}
                >
                  {t.catalogPublish}
                </Button>
              )}
            </li>
          ))}
        </ul>
      ) : null}
      {missing.length > 0 ? (
        <p className="text-sm text-amber-800">
          {t.catalogMissing.replace('{names}', missing.join(', '))}
        </p>
      ) : null}
      {catalog?.changes ? (
        <ToolCatalogChangesList changes={catalog.changes} published={published} t={t} />
      ) : null}
    </section>
  )
}

// ToolCatalogChangesList は直前のカタログからの差分を示す (REQ-OAUTH2-087)。操作を持たず、公開設定を変えない。
// 公開中のツールの説明や引数の変化に気付けるよう、公開中かどうかと変更前の定義を並べる。
function ToolCatalogChangesList({
  changes,
  published,
  t,
}: {
  changes: ToolCatalogChanges
  published: Set<string>
  t: Dictionary
}) {
  const fieldLabel = {
    title: t.changeFieldTitle,
    description: t.changeFieldDescription,
    arguments: t.changeFieldArguments,
  }
  const mark = (name: string) =>
    published.has(name) ? (
      <span className="text-xs text-slate-600"> ({t.catalogPublished})</span>
    ) : null
  const groups = [
    { key: 'added', label: t.changesAdded, names: changes.added },
    { key: 'removed', label: t.changesRemoved, names: changes.removed },
  ]
  return (
    <section
      className="flex flex-col gap-2 rounded-md border border-amber-300 p-2"
      aria-labelledby="catalog-changes-heading"
    >
      <p id="catalog-changes-heading" className="text-sm font-medium">
        {t.changesHeading}
      </p>
      <p className="text-xs leading-5 text-slate-500">
        {t.changesAt.replace('{time}', new Date(changes.changed_at).toLocaleString())}
      </p>
      {groups.map((group) =>
        group.names.length > 0 ? (
          <div key={group.key} data-change={group.key} className="text-sm">
            <span className="font-medium">
              {group.label}
              {t.changesLabelSeparator}
            </span>
            {group.names.map((name, index) => (
              <span key={name}>
                {index > 0 ? ', ' : ''}
                <code>{name}</code>
                {mark(name)}
              </span>
            ))}
          </div>
        ) : null,
      )}
      {changes.modified.length > 0 ? (
        <div data-change="modified" className="flex flex-col gap-1 text-sm">
          <span className="font-medium">{t.changesModified}</span>
          <ul className="flex flex-col gap-1">
            {changes.modified.map((tool) => (
              <li key={tool.name} className="flex flex-col">
                <span>
                  <code>{tool.name}</code>
                  {mark(tool.name)}
                  {t.changesLabelSeparator}
                  {tool.fields.map((field) => fieldLabel[field]).join(', ')}
                </span>
                {tool.fields.includes('description') ? (
                  <span className="text-xs text-slate-600">
                    {t.changesPreviousDescription.replace(
                      '{description}',
                      tool.previous.description ?? '',
                    )}
                  </span>
                ) : null}
                {tool.fields.includes('arguments') ? (
                  <span className="text-xs text-slate-600">
                    {t.changesPreviousArguments.replace(
                      '{arguments}',
                      (tool.previous.arguments ?? []).join(', '),
                    )}
                  </span>
                ) : null}
              </li>
            ))}
          </ul>
        </div>
      ) : null}
    </section>
  )
}
