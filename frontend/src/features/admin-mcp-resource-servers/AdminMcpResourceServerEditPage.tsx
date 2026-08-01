import { IconArrowLeft } from '@tabler/icons-react'
import { type FormEvent, useState } from 'react'
import { AuthenticationAPIError, updateMcpResourceServer } from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { useDictionary } from '../../lib/i18n'
import type { McpResourceServer } from '../../types'
import { adminMcpResourceServersDictionary } from './AdminMcpResourceServersPage.i18n'
import {
  listURL,
  McpResourceServerFormFields,
  parseScopes,
  toForm,
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
  const t = useDictionary(adminMcpResourceServersDictionary)

  async function handleSubmit(event: FormEvent) {
    event.preventDefault()
    setError('')
    setSaving(true)
    try {
      await updateMcpResourceServer(csrfToken, resourceServer.resource_server_id, {
        name: form.name,
        scopes: parseScopes(form.scopes),
        state: form.state,
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
      title={t.editResource.replace('{resource}', resourceServer.resource)}
      description={t.pageDescription}
      actions={
        <Button variant="outline" nativeButton={false} render={<a href={listURL()} />}>
          <IconArrowLeft size={16} aria-hidden="true" />
          {t.backToList}
        </Button>
      }
    >
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      <Card className="w-full max-w-2xl p-4">
        <form className="flex flex-col gap-4" onSubmit={handleSubmit}>
          <McpResourceServerFormFields form={form} onChange={setForm} locked={true} t={t} />
          <div className="flex gap-2.5">
            <Button type="submit" disabled={saving}>
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
