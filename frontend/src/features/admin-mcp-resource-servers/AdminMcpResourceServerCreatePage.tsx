import { IconArrowLeft } from '@tabler/icons-react'
import { type FormEvent, useMemo, useState } from 'react'
import {
  AuthenticationAPIError,
  createAdminApplication,
  createMcpResourceServer,
  tenantURL,
  updateApplicationOidcConfig,
} from '../../api'
import { CopyableValue } from '../../components/CopyableValue'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { useFormValidation } from '../../lib/form'
import { useDictionary } from '../../lib/i18n'
import type { McpResourceServer } from '../../types'
import { adminMcpResourceServersDictionary } from './AdminMcpResourceServersPage.i18n'
import {
  editURL,
  emptyForm,
  listURL,
  McpResourceServerFormFields,
  NEW_EVALUATOR,
  parseAccessRules,
  resourceServerSchema,
  hostingInput,
  hostedAccessRules,
} from './AdminMcpResourceServersShared'

type CreatedClient = { clientId: string; secret: string; applicationId: string }

// evaluatorName は新しく作る評価用のクライアントの名前である。MCP リソースサーバーの名前から導き、入力を求めない。
function evaluatorName(name: string): string {
  return `${name.trim()} Gateway`
}

// gatewayConfig は配置先へ渡す設定の例を組み立てる。シークレットの値はファイルへ置かせ、設定に書かない。
function gatewayConfig(server: McpResourceServer, clientId: string): string {
  const realm = new URL(tenantURL(''), window.location.origin).toString().replace(/\/$/, '')
  return [
    'listen: 127.0.0.1:8080',
    `resource: ${server.resource}`,
    'upstream: http://127.0.0.1:3000/mcp',
    'idmagic:',
    `  realm_url: ${realm}`,
    `  client_id: ${JSON.stringify(clientId)}`,
    '  client_secret_file: /etc/idmagic-mcp-gateway/client-secret',
    'mapping:',
    '  catalog: true',
  ].join('\n')
}

// AdminMcpResourceServerCreatePage はMCP リソースサーバーの登録の専用画面。評価用のクライアントの作成と紐付けを
// 同じ送信で行い、作成済みのクライアントは再試行で作り直さない (REQ-OAUTH2-088)。
export function AdminMcpResourceServerCreatePage({
  csrfToken,
  actorUsername,
}: {
  csrfToken: string
  actorUsername?: string
}) {
  const [form, setForm] = useState(emptyForm)
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)
  const [createdClient, setCreatedClient] = useState<CreatedClient | null>(null)
  const [registered, setRegistered] = useState<McpResourceServer | null>(null)
  const t = useDictionary(adminMcpResourceServersDictionary)
  const managed = form.hostingMode === 'managed'
  const schema = useMemo(
    () => resourceServerSchema(t, managed, form.upstreamAuthentication === 'token_vault'),
    [t, managed, form.upstreamAuthentication],
  )
  const validation = useFormValidation(schema, {
    resource: form.resource,
    name: form.name,
    scopes: form.scopes,
    accessRules: form.accessRules,
    upstreamURL: form.upstreamURL,
    connectorId: form.connectorId,
  })

  const creatingClient = form.clientId === NEW_EVALUATOR
  const summary = (
    creatingClient ? t.summaryNew : form.clientId ? t.summaryExisting : t.summaryNone
  )
    .replace('{name}', form.name.trim() || '—')
    .replace('{client}', evaluatorName(form.name || '—'))

  async function handleSubmit(event: FormEvent) {
    event.preventDefault()
    setError('')
    const data = validation.data
    if (!data) {
      validation.revealAll()
      return
    }
    const accessRules = hostedAccessRules(form, parseAccessRules(data.accessRules) ?? [])
    setSaving(true)
    let client = createdClient
    try {
      if (creatingClient && client === null) {
        const result = await createAdminApplication(csrfToken, {
          name: evaluatorName(form.name),
          type: 'service',
          ...(managed ? { scope: data.scopes.join(' ') } : {}),
        })
        if (!result.client_id || !result.client_secret) throw new Error(t.saveFailedError)
        client = {
          clientId: result.client_id,
          secret: result.client_secret,
          applicationId: result.application.id,
        }
        setCreatedClient(client)
      }
      if (managed && creatingClient && client && form.upstreamAuthentication === 'token_exchange') {
        await updateApplicationOidcConfig(csrfToken, client.applicationId, {
          grant_types: ['client_credentials', 'urn:ietf:params:oauth:grant-type:token-exchange'],
          scope: data.scopes.join(' '),
        })
      }
      const clientId = creatingClient ? client?.clientId : form.clientId
      const server = await createMcpResourceServer(csrfToken, {
        resource: data.resource,
        name: data.name,
        scopes: data.scopes,
        ...(clientId ? { client_id: clientId } : {}),
        ...(accessRules.length > 0 ? { access_rules: accessRules } : {}),
        state: form.state,
        ...(managed ? { hosting: hostingInput(form) } : {}),
      })
      if (!clientId) {
        window.location.assign(listURL())
        return
      }
      setRegistered(server)
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.saveFailedError)
    } finally {
      setSaving(false)
    }
  }

  const resultClientId = createdClient?.clientId ?? (registered ? form.clientId : '')

  return (
    <AdminShell
      active="mcp-resource-servers"
      actorUsername={actorUsername}
      title={t.registerResourceServer}
      description={t.pageDescription}
    >
      <div className="mb-4">
        <Button
          variant="ghost"
          size="icon"
          nativeButton={false}
          render={<a href={listURL()} />}
          aria-label={t.backToList}
        >
          <IconArrowLeft size={18} aria-hidden="true" />
        </Button>
      </div>
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      {registered ? (
        <Card className="w-full p-6" aria-live="polite">
          <h2 className="text-base font-semibold">{t.resultHeading}</h2>
          <p className="mt-1 text-sm text-slate-600">
            {registered.hosting ? t.managedResultIntro : t.resultIntro}
          </p>
          {registered.hosting ? (
            <dl className="mt-3 grid gap-2 text-sm">
              <dt className="font-medium">{t.publicURL}</dt>
              <dd>
                <CopyableValue
                  id="result-public-url"
                  value={registered.resource}
                  copyLabel={t.copy.replace('{label}', t.publicURL)}
                />
              </dd>
              {registered.hosting.authentication === 'token_exchange' &&
              registered.gateway_dpop_jwk ? (
                <>
                  <dt className="font-medium">{t.publicKey}</dt>
                  <dd>
                    <CopyableValue
                      id="result-gateway-public-key"
                      value={JSON.stringify(registered.gateway_dpop_jwk, null, 2)}
                      copyLabel={t.copy.replace('{label}', t.publicKey)}
                      multiline
                    />
                  </dd>
                </>
              ) : null}
              <dd>
                {registered.hosting.authentication === 'token_exchange'
                  ? t.exchangeBypassHelp
                  : t.vaultBypassHelp}
              </dd>
            </dl>
          ) : (
            <dl className="mt-3 grid gap-2 text-sm">
              <dt className="font-medium">{t.resultClientID}</dt>
              <dd>
                <CopyableValue
                  id="result-client-id"
                  value={resultClientId}
                  copyLabel={t.copy.replace('{label}', t.resultClientID)}
                />
              </dd>
              {createdClient ? (
                <>
                  <dt className="font-medium">{t.resultSecret}</dt>
                  <dd>
                    <p className="mb-2 text-xs text-slate-500">{t.resultSecretHelp}</p>
                    <CopyableValue
                      id="result-client-secret"
                      value={createdClient.secret}
                      copyLabel={t.copy.replace('{label}', t.resultSecret)}
                    />
                  </dd>
                </>
              ) : null}
              <dt className="font-medium">{t.resultConfig}</dt>
              <dd>
                <p className="mb-2 text-xs leading-5 text-slate-500">{t.resultConfigHelp}</p>
                <CopyableValue
                  id="result-gateway-config"
                  value={gatewayConfig(registered, resultClientId)}
                  copyLabel={t.copy.replace('{label}', t.resultConfig)}
                  multiline
                />
              </dd>
            </dl>
          )}
          <p className="mt-3 text-sm text-slate-600">
            {registered.hosting ? t.managedResultNext : t.resultNext}
          </p>
          <div className="mt-3 flex gap-2.5">
            <Button
              nativeButton={false}
              render={<a href={`${editURL(registered.id)}#tool-catalog-heading`} />}
            >
              {t.resultEdit}
            </Button>
            <Button variant="outline" nativeButton={false} render={<a href={listURL()} />}>
              {t.backToList}
            </Button>
          </div>
        </Card>
      ) : (
        <Card className="w-full admin-form p-4">
          {createdClient ? (
            <p className="mb-3 rounded-md bg-amber-50 p-3 text-sm text-amber-900" role="status">
              {t.createdClientPending.replace('{clientId}', createdClient.clientId)}
            </p>
          ) : null}
          <form className="grid gap-4 md:grid-cols-2" onSubmit={handleSubmit}>
            <McpResourceServerFormFields
              form={form}
              validation={validation}
              onChange={setForm}
              locked={false}
              t={t}
            />
            <p className="text-sm md:col-span-2" aria-live="polite">
              {summary}
            </p>
            <div className="flex gap-2.5 md:col-span-2">
              <Button type="submit" disabled={saving || !validation.data}>
                {t.register}
              </Button>
              <Button variant="outline" nativeButton={false} render={<a href={listURL()} />}>
                {t.cancel}
              </Button>
            </div>
          </form>
        </Card>
      )}
    </AdminShell>
  )
}
