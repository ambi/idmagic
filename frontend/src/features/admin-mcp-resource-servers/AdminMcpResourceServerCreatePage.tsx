import { IconArrowLeft } from '@tabler/icons-react'
import { type FormEvent, useMemo, useState } from 'react'
import {
  AuthenticationAPIError,
  createAdminApplication,
  createMcpResourceServer,
  tenantURL,
} from '../../api'
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
} from './AdminMcpResourceServersShared'

type CreatedClient = { clientId: string; secret: string }

// evaluatorName は新しく作る評価用のクライアントの名前である。接続先の名前から導き、入力を求めない。
function evaluatorName(name: string): string {
  return `${name.trim()} Gateway`
}

// gatewayConfig は配置先へ渡す設定の例を組み立てる。シークレットの値はファイルへ置かせ、設定に書かない。
function gatewayConfig(
  server: McpResourceServer,
  clientId: string,
  t: { resultUpstreamComment: string; resultSecretComment: string },
): string {
  const realm = new URL(tenantURL(''), window.location.origin).toString().replace(/\/$/, '')
  return [
    'listen: 127.0.0.1:8080',
    `resource: ${server.resource}`,
    `upstream: http://127.0.0.1:3000/mcp  # ${t.resultUpstreamComment}`,
    'idmagic:',
    `  realm_url: ${realm}`,
    `  client_id: ${JSON.stringify(clientId)}`,
    `  client_secret_file: /etc/idmagic-mcp-gateway/client-secret  # ${t.resultSecretComment}`,
    'mapping:',
    '  catalog: true',
  ].join('\n')
}

// AdminMcpResourceServerCreatePage は接続先の登録の専用画面。評価用のクライアントの作成と束縛を
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
  const schema = useMemo(() => resourceServerSchema(t), [t])
  const validation = useFormValidation(schema, {
    resource: form.resource,
    name: form.name,
    scopes: form.scopes,
    accessRules: form.accessRules,
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
    const accessRules = parseAccessRules(data.accessRules) ?? []
    setSaving(true)
    let client = createdClient
    try {
      if (creatingClient && client === null) {
        const result = await createAdminApplication(csrfToken, {
          name: evaluatorName(form.name),
          type: 'service',
        })
        if (!result.client_id || !result.client_secret) throw new Error(t.saveFailedError)
        client = { clientId: result.client_id, secret: result.client_secret }
        setCreatedClient(client)
      }
      const clientId = creatingClient ? client?.clientId : form.clientId
      const server = await createMcpResourceServer(csrfToken, {
        resource: data.resource,
        name: data.name,
        scopes: data.scopes,
        ...(clientId ? { client_id: clientId } : {}),
        ...(accessRules.length > 0 ? { access_rules: accessRules } : {}),
        state: form.state,
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
      actions={
        <Button variant="outline" nativeButton={false} render={<a href={listURL()} />}>
          <IconArrowLeft size={16} aria-hidden="true" />
          {t.backToList}
        </Button>
      }
    >
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      {registered ? (
        <Card className="w-full p-4" aria-live="polite">
          <h2 className="text-base font-semibold">{t.resultHeading}</h2>
          <p className="mt-1 text-sm text-slate-600">{t.resultIntro}</p>
          <dl className="mt-3 grid gap-2 text-sm">
            <dt className="font-medium">{t.resultClientID}</dt>
            <dd>
              <code id="result-client-id">{resultClientId}</code>
            </dd>
            {createdClient ? (
              <>
                <dt className="font-medium">{t.resultSecret}</dt>
                <dd>
                  <code id="result-client-secret">{createdClient.secret}</code>
                </dd>
              </>
            ) : null}
            <dt className="font-medium">{t.resultConfig}</dt>
            <dd>
              <pre
                id="result-gateway-config"
                className="overflow-x-auto rounded-md bg-slate-50 p-3 text-xs"
              >
                {gatewayConfig(registered, resultClientId, t)}
              </pre>
            </dd>
          </dl>
          <div className="mt-3 flex gap-2.5">
            <Button nativeButton={false} render={<a href={editURL(registered.id)} />}>
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
