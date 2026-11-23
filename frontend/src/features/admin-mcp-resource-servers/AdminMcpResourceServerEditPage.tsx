import { IconArrowLeft } from '@tabler/icons-react'
import { type FormEvent, useMemo, useState } from 'react'
import { AuthenticationAPIError, updateMcpResourceServer } from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { useFormValidation } from '../../lib/form'
import { useDictionary } from '../../lib/i18n'
import type { McpResourceServer } from '../../types'
import { adminMcpResourceServersDictionary } from './AdminMcpResourceServersPage.i18n'
import {
  listURL,
  McpResourceServerFormFields,
  parseAccessRules,
  ToolCatalogPanel,
  ToolDiscoveryPanel,
  resourceServerSchema,
  toForm,
  hostingInput,
  hostedAccessRules,
} from './AdminMcpResourceServersShared'

// AdminMcpResourceServerEditPage は MCP リソースサーバー編集の専用画面 (wi-314 T014)。
export function AdminMcpResourceServerEditPage({
  csrfToken,
  actorUsername,
  resourceServer,
}: {
  csrfToken: string
  actorUsername?: string
  resourceServer: McpResourceServer
}) {
  const [form, setForm] = useState(() => toForm(resourceServer))
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)
  const [catalogVersion, setCatalogVersion] = useState(0)
  const t = useDictionary(adminMcpResourceServersDictionary)
  const schema = useMemo(
    () =>
      resourceServerSchema(
        t,
        form.hostingMode === 'managed',
        form.upstreamAuthentication === 'token_vault',
      ),
    [t, form.hostingMode, form.upstreamAuthentication],
  )
  const validation = useFormValidation(schema, {
    resource: form.resource,
    name: form.name,
    scopes: form.scopes,
    accessRules: form.accessRules,
    upstreamURL: form.upstreamURL,
    connectorId: form.connectorId,
  })

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
    try {
      await updateMcpResourceServer(csrfToken, resourceServer.id, {
        name: data.name,
        scopes: data.scopes,
        scope_hints: Object.fromEntries(
          data.scopes.flatMap((scope) =>
            form.scopeHints[scope] ? [[scope, form.scopeHints[scope]]] : [],
          ),
        ),
        client_id: form.clientId,
        access_rules: accessRules,
        state: form.state,
        ...(resourceServer.hosting ? { hosting: hostingInput(form) } : {}),
      })
      window.location.assign(listURL())
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.saveFailedError)
      setSaving(false)
    }
  }

  return (
    <AdminShell
      active="mcp-resource-servers"
      actorUsername={actorUsername}
      title={t.editResource.replace('{resource}', resourceServer.name)}
      description={t.pageDescription}
      actions={
        <Button variant="outline" nativeButton={false} render={<a href={listURL()} />}>
          <IconArrowLeft size={16} aria-hidden="true" />
          {t.backToList}
        </Button>
      }
    >
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      <Card className="w-full admin-form p-4">
        <form className="grid gap-4 md:grid-cols-2" onSubmit={handleSubmit}>
          <McpResourceServerFormFields
            form={form}
            validation={validation}
            onChange={setForm}
            locked={true}
            t={t}
          />
          {resourceServer.hosting ? null : (
            <ToolDiscoveryPanel
              csrfToken={csrfToken}
              resourceServerID={resourceServer.id}
              form={form}
              onChange={setForm}
              onCatalogUpdated={() => setCatalogVersion((version) => version + 1)}
              t={t}
            />
          )}
          {/* MCP サーバーから取得した後は key を変えて作り直し、カタログを読み直す。 */}
          <ToolCatalogPanel
            key={catalogVersion}
            resourceServerID={resourceServer.id}
            form={form}
            onChange={setForm}
            t={t}
          />
          <div className="flex gap-2.5 md:col-span-2">
            <Button type="submit" disabled={saving || !validation.data}>
              {t.update}
            </Button>
            <Button variant="outline" nativeButton={false} render={<a href={listURL()} />}>
              {t.cancel}
            </Button>
          </div>
        </form>
      </Card>
    </AdminShell>
  )
}
