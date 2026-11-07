import { IconArrowLeft } from '@tabler/icons-react'
import { type FormEvent, useMemo, useRef, useState } from 'react'
import {
  AuthenticationAPIError,
  bindAdminAgentCredential,
  createAdminAgentKeyClient,
  registerAdminAgent,
  tenantURL,
  writeRelationTuples,
} from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { CharacterCount } from '../../components/CharacterCount'
import { CopyableValue } from '../../components/CopyableValue'
import { FieldError } from '../../components/FieldError'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import {
  formSchema,
  optionalText,
  requiredText,
  requiredUrl,
  useFormValidation,
  when,
} from '../../lib/form'
import { useDictionary } from '../../lib/i18n'
import { LENGTH } from '../../lib/lengthLimits'
import { operationTuple } from '../../lib/mcpAccess'
import type { AdminAgent, AdminUser, McpResourceServer } from '../../types'
import { agentCreateFlowDictionary } from './agentCreateFlow.i18n'
import { adminAgentsDictionary } from './AdminAgentsPage.i18n'
import { OwnerSelect } from './AdminAgentsShared'
import { parsePublicKey } from './publicKey'
import { isTargetScoped } from './targetRange'

// CIBA の承認の依頼に要るスコープ。資格情報の画面のデフォルトと揃える。
const APPROVAL_SCOPES = ['openid', 'profile']

type AuthMethod = 'key' | 'jwks_uri' | 'later'

// Progress は完了した段階の結果である。再試行は未完了の段階だけを呼ぶ (REQ-IDMANAGEMENT-105)。
type Progress = { agentID?: string; clientID?: string; bound?: boolean; assigned?: boolean }

type Operation = {
  server: McpResourceServer
  toolName: string
  label: string
  key: string
  // targetScoped な操作は、範囲を選ぶまでどの対象も許可しない (REQ-AUTHORIZATION-017)。
  targetScoped: boolean
}

function publishedOperations(servers: McpResourceServer[]): Operation[] {
  return servers
    .filter((server) => server.state === 'Active')
    .flatMap((server) =>
      (server.access_rules ?? []).flatMap((rule) =>
        rule.tool?.assignment_resource_id
          ? [
              {
                server,
                toolName: rule.tool.name,
                label: rule.tool.display_name,
                key: `${server.id}/${rule.tool.name}`,
                targetScoped: isTargetScoped(rule),
              },
            ]
          : [],
      ),
    )
}

function OriginTag({ label }: { label: string }) {
  return (
    <span className="mr-1 rounded bg-slate-100 px-1.5 py-0.5 text-[11px] font-medium text-slate-700">
      {label}
    </span>
  )
}

// AdminAgentCreatePage は Agent の登録、認証の準備、利用する接続先の割当を一つの送信で行う。
export function AdminAgentCreatePage({
  csrfToken,
  actorUsername,
  users = [],
  resourceServers = [],
}: {
  csrfToken: string
  actorUsername?: string
  users?: AdminUser[]
  resourceServers?: McpResourceServer[]
}) {
  const listPath = tenantURL('/admin/agents')
  const t = useDictionary(adminAgentsDictionary)
  const flow = useDictionary(agentCreateFlowDictionary)
  const operations = publishedOperations(resourceServers)
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [kind, setKind] = useState<AdminAgent['kind']>('autonomous')
  const [owner, setOwner] = useState('')
  const [method, setMethod] = useState<AuthMethod>('key')
  const [publicKey, setPublicKey] = useState('')
  const [jwksURI, setJwksURI] = useState('')
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [progress, setProgress] = useState<Progress>({})
  const [done, setDone] = useState(false)
  const [busy, setBusy] = useState(false)
  const running = useRef(false)
  const [error, setError] = useState('')

  const schema = useMemo(
    () =>
      formSchema({
        name: requiredText(LENGTH.name),
        description: optionalText(LENGTH.description),
        publicKey: when(method === 'key', requiredText()),
        // 配置先の JWKS URI は、サーバーが tokens_jose.ValidateJWKSURI で検証する。
        jwksURI: when(method === 'jwks_uri', requiredUrl({ https: true })),
      }),
    [method],
  )
  const form = useFormValidation(schema, { name, description, publicKey, jwksURI })

  const chosen = operations.filter((operation) => selected.has(operation.key))
  const summary = flow.summary
    .replace('{name}', name.trim() || '—')
    .replace(
      '{auth}',
      method === 'key'
        ? flow.summaryAuthKey
        : method === 'jwks_uri'
          ? flow.summaryAuthJwksUri
          : flow.summaryAuthLater,
    )
    .replace(
      '{operations}',
      chosen.length > 0
        ? flow.summaryOperations.replace(
            '{operations}',
            chosen.map((operation) => operation.label).join('、'),
          )
        : flow.summaryNoOperations,
    )

  // keySource は入力から認証用クライアントの鍵を作る。入力が受け付けられなければ理由を返す。
  // 鍵の解析は Web Crypto を使う非同期の処理なので、欄の規則ではなく送信時に行う。
  async function keySource(): Promise<
    { jwks: { keys: Record<string, unknown>[] } } | { jwks_uri: string } | string
  > {
    if (form.data?.jwksURI !== undefined) return { jwks_uri: form.data.jwksURI }
    const jwks = await parsePublicKey(publicKey)
    if (jwks === 'private') return t.publicKeyHasPrivateMembers
    if (jwks === 'unsupported') return t.publicKeyUnsupported
    if (jwks === null) return t.publicKeyInvalid
    return { jwks }
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const data = form.data
    if (!data) {
      form.revealAll()
      return
    }
    if (running.current) return
    running.current = true
    setBusy(true)
    setError('')
    const next: Progress = { ...progress }
    try {
      const key = method === 'later' ? null : await keySource()
      if (typeof key === 'string') {
        setError(key)
        return
      }
      if (!next.agentID) {
        const created = await registerAdminAgent(csrfToken, {
          name: data.name,
          description: data.description,
          kind,
          owner_user_id: owner || undefined,
        })
        next.agentID = created.id
        setProgress({ ...next })
      }
      if (key && !next.clientID) {
        const scopes = new Set([
          ...APPROVAL_SCOPES,
          ...resourceServers
            .filter((server) => server.state === 'Active')
            .flatMap((server) => server.scopes),
        ])
        next.clientID = await createAdminAgentKeyClient(csrfToken, {
          name: data.name,
          scope: [...scopes].join(' '),
          ...key,
        })
        setProgress({ ...next })
      }
      if (next.clientID && !next.bound) {
        await bindAdminAgentCredential(csrfToken, next.agentID, next.clientID)
        next.bound = true
        setProgress({ ...next })
      }
      if (chosen.length > 0 && !next.assigned) {
        await writeRelationTuples(csrfToken, {
          writes: chosen.map((operation) =>
            operationTuple(operation.server, operation.toolName, next.agentID!),
          ),
          deletes: [],
        })
        next.assigned = true
        setProgress({ ...next })
      }
      // 配置先へ渡す値がなければ、従来どおり Agent の詳細へ進む。
      if (!next.clientID) {
        window.location.assign(tenantURL(`/admin/agents/${encodeURIComponent(next.agentID)}`))
        return
      }
      setDone(true)
    } catch (cause) {
      setError(
        cause instanceof AuthenticationAPIError
          ? cause.code === 'agent_client_auth_method_not_allowed'
            ? t.agentClientAuthMethodNotAllowedError
            : cause.message
          : t.genericActionError,
      )
    } finally {
      running.current = false
      setBusy(false)
    }
  }

  const tokenEndpoint = new URL(tenantURL('/token'), window.location.origin).toString()
  const resources = [...new Set(chosen.map((operation) => operation.server.resource))]
  const config = progress.clientID
    ? [
        `token_endpoint: ${tokenEndpoint}`,
        `client_id: ${JSON.stringify(progress.clientID)}`,
        'token_endpoint_auth_method: private_key_jwt',
        ...resources.map((resource) => `resource: ${resource}`),
      ].join('\n')
    : ''

  return (
    <AdminShell
      active="agents"
      actorUsername={actorUsername}
      title={t.registerAgentHeading}
      description={t.registerAgentDescription}
    >
      <div className="flex items-center gap-3">
        <a
          href={listPath}
          className="inline-flex size-9 items-center justify-center rounded-lg border border-slate-200 bg-white text-slate-700 transition hover:bg-slate-50 hover:text-slate-900"
          aria-label={t.backToAgentListAria}
        >
          <IconArrowLeft size={18} aria-hidden="true" />
        </a>
      </div>

      {done && progress.agentID ? (
        <Card className="mt-6 p-6" aria-live="polite">
          <h2 className="text-base font-semibold">{flow.resultHeading}</h2>
          {progress.clientID ? (
            <>
              <p className="mt-1 text-sm text-slate-600">{flow.resultIntro}</p>
              <dl className="mt-3 grid gap-2 text-sm">
                <dt className="font-medium">{flow.resultClientId}</dt>
                <dd id="result-agent-client-id">
                  <CopyableValue value={progress.clientID} copyLabel={flow.copy} />
                </dd>
                <dt className="font-medium">{flow.resultTokenEndpoint}</dt>
                <dd>
                  <code>{tokenEndpoint}</code>
                </dd>
                {resources.length > 0 ? (
                  <>
                    <dt className="font-medium">{flow.resultResources}</dt>
                    <dd>
                      {resources.map((resource) => (
                        <code key={resource} className="block">
                          {resource}
                        </code>
                      ))}
                    </dd>
                  </>
                ) : null}
                <dt className="font-medium">{flow.resultConfig}</dt>
                <dd>
                  <pre
                    id="result-agent-config"
                    className="overflow-x-auto rounded-md bg-slate-50 p-3 text-xs"
                  >
                    {config}
                  </pre>
                </dd>
              </dl>
            </>
          ) : (
            <p className="mt-1 text-sm text-slate-600">{flow.resultNoClient}</p>
          )}
          <div className="mt-4">
            <Button
              nativeButton={false}
              render={
                <a href={tenantURL(`/admin/agents/${encodeURIComponent(progress.agentID)}`)} />
              }
            >
              {flow.resultOpenAgent}
            </Button>
          </div>
        </Card>
      ) : (
        <div className="mt-6 admin-form">
          <Card className="shadow-[0_1px_2px_rgb(15_23_42/4%)]">
            <form onSubmit={handleSubmit}>
              <div className="grid gap-6 p-6 md:grid-cols-2">
                {error && <Alert variant="destructive">{error}</Alert>}
                {progress.agentID ? (
                  <p
                    className="rounded-md bg-amber-50 p-3 text-sm text-amber-900 md:col-span-2"
                    role="status"
                  >
                    {flow.stageAgent}{' '}
                    {progress.clientID
                      ? `${flow.stageClient.replace('{clientId}', progress.clientID)} `
                      : ''}
                    {progress.bound ? `${flow.stageBound} ` : ''}
                    {flow.stageRetry}
                  </p>
                ) : null}

                <div className="grid gap-1.5">
                  <Label htmlFor="agent-name" required>
                    {t.agentNameLabel}
                  </Label>
                  <Input
                    id="agent-name"
                    name="name"
                    required
                    placeholder="invoice-bot"
                    value={name}
                    disabled={Boolean(progress.agentID)}
                    onChange={(event) => setName(event.target.value)}
                    {...form.fieldProps('name')}
                  />
                  <FieldError form={form} name="name" />
                  <p className="text-xs text-slate-500">
                    <OriginTag label={flow.originYou} /> {t.agentNameHelp}
                  </p>
                </div>
                <div className="grid gap-1.5">
                  <Label htmlFor="agent-description" optional>
                    {t.descriptionOptionalLabel}
                  </Label>
                  <Input
                    id="agent-description"
                    name="description"
                    placeholder={t.agentDescriptionPlaceholder}
                    value={description}
                    disabled={Boolean(progress.agentID)}
                    onChange={(event) => setDescription(event.target.value)}
                    {...form.fieldProps('description')}
                  />
                  <CharacterCount value={description} max={LENGTH.description} />
                  <FieldError form={form} name="description" />
                </div>
                <div className="grid gap-1.5">
                  <Label htmlFor="agent-kind">{t.kindLabel}</Label>
                  <select
                    id="agent-kind"
                    value={kind}
                    disabled={Boolean(progress.agentID)}
                    onChange={(e) => setKind(e.target.value as AdminAgent['kind'])}
                    className="h-9 rounded-md border border-slate-300 bg-white px-2 text-sm"
                  >
                    <option value="autonomous">{t.kindAutonomousOption}</option>
                    <option value="supervised">{t.kindSupervisedOption}</option>
                  </select>
                  <p className="text-xs text-slate-500">
                    <OriginTag label={flow.originDefault} />
                  </p>
                </div>
                <div className="grid gap-1.5">
                  <Label htmlFor="agent-owner" optional>
                    {t.ownerOptionalLabel}
                  </Label>
                  <OwnerSelect
                    id="agent-owner"
                    name="owner_user_id"
                    users={users}
                    emptyLabel={t.ownerDefaultOption}
                    value={owner}
                    onChange={setOwner}
                  />
                  <p className="text-xs text-slate-500">
                    <OriginTag label={flow.originDefault} /> {t.ownerHelp}
                  </p>
                </div>

                <fieldset className="grid gap-2 md:col-span-2">
                  <legend className="mb-1 font-semibold">{flow.authHeading}</legend>
                  <p className="text-xs text-slate-500">{flow.authHelp}</p>
                  <div className="flex flex-wrap gap-4 text-sm">
                    {(
                      [
                        ['key', flow.authKey],
                        ['jwks_uri', flow.authJwksUri],
                        ['later', flow.authLater],
                      ] as const
                    ).map(([value, label]) => (
                      <label key={value} className="flex items-center gap-2">
                        <input
                          type="radio"
                          name="auth-method"
                          value={value}
                          checked={method === value}
                          disabled={Boolean(progress.clientID)}
                          onChange={() => setMethod(value)}
                        />
                        {label}
                      </label>
                    ))}
                  </div>
                  {method === 'key' ? (
                    <div className="grid gap-1.5">
                      <Label htmlFor="agent-create-public-key" required>
                        {flow.publicKeyLabel}
                      </Label>
                      <textarea
                        id="agent-create-public-key"
                        rows={5}
                        value={publicKey}
                        disabled={Boolean(progress.clientID)}
                        onChange={(event) => setPublicKey(event.target.value)}
                        className="rounded-md border border-slate-300 bg-white p-2 font-mono text-xs"
                        {...form.fieldProps('publicKey')}
                      />
                      <FieldError form={form} name="publicKey" />
                      <p className="text-xs text-slate-500">
                        <OriginTag label={flow.originYou} />
                      </p>
                    </div>
                  ) : null}
                  {method === 'jwks_uri' ? (
                    <div className="grid gap-1.5">
                      <Label htmlFor="agent-create-jwks-uri" required>
                        {flow.jwksUriLabel}
                      </Label>
                      <Input
                        id="agent-create-jwks-uri"
                        type="url"
                        value={jwksURI}
                        disabled={Boolean(progress.clientID)}
                        onChange={(event) => setJwksURI(event.target.value)}
                        {...form.fieldProps('jwksURI')}
                      />
                      <FieldError form={form} name="jwksURI" />
                      <p className="text-xs text-slate-500">
                        <OriginTag label={flow.originYou} /> {flow.jwksUriHelp}
                      </p>
                    </div>
                  ) : null}
                </fieldset>

                <fieldset className="grid gap-2 md:col-span-2">
                  <legend className="mb-1 font-semibold">{flow.connectionsHeading}</legend>
                  <p className="text-xs text-slate-500">
                    <OriginTag label={flow.originYou} /> {flow.connectionsHelp}
                  </p>
                  {operations.length === 0 ? (
                    <p className="text-sm text-slate-600">{flow.connectionsEmpty}</p>
                  ) : (
                    resourceServers
                      .filter((server) => operations.some((item) => item.server.id === server.id))
                      .map((server) => (
                        <div key={server.id} className="grid gap-1 rounded-md border p-3">
                          <span className="text-sm font-medium">{server.name}</span>
                          {operations
                            .filter((operation) => operation.server.id === server.id)
                            .map((operation) => (
                              <label
                                key={operation.key}
                                className="flex items-center gap-2 text-sm"
                              >
                                <input
                                  type="checkbox"
                                  checked={selected.has(operation.key)}
                                  disabled={Boolean(progress.assigned)}
                                  onChange={(event) =>
                                    setSelected((current) => {
                                      const next = new Set(current)
                                      if (event.target.checked) next.add(operation.key)
                                      else next.delete(operation.key)
                                      return next
                                    })
                                  }
                                />
                                <span>
                                  {operation.label}
                                  {operation.targetScoped ? (
                                    <span className="block text-xs text-amber-800">
                                      {flow.targetScopedNote}
                                    </span>
                                  ) : null}
                                </span>
                              </label>
                            ))}
                        </div>
                      ))
                  )}
                </fieldset>

                <p className="text-sm md:col-span-2" aria-live="polite" id="agent-create-summary">
                  {summary}
                </p>
              </div>

              <div className="flex justify-end gap-2 border-t border-slate-200 bg-slate-50 px-6 py-4">
                <a
                  href={listPath}
                  className="inline-flex h-9 items-center justify-center rounded-lg border border-slate-200 bg-white px-4 text-sm font-medium text-slate-700 shadow-sm transition hover:bg-slate-50 hover:text-slate-900"
                >
                  {t.cancel}
                </a>
                <Button type="submit" disabled={busy || !form.data}>
                  {t.register}
                </Button>
              </div>
            </form>
          </Card>
        </div>
      )}
    </AdminShell>
  )
}
