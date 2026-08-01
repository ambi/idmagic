import { IconArrowLeft } from '@tabler/icons-react'
import { type FormEvent, useState } from 'react'
import { AuthenticationAPIError, createMcpResourceServer } from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { useDictionary } from '../../lib/i18n'
import { adminMcpResourceServersDictionary } from './AdminMcpResourceServersPage.i18n'
import {
  emptyForm,
  listURL,
  McpResourceServerFormFields,
  parseScopes,
} from './AdminMcpResourceServersShared'

// AdminMcpResourceServerCreatePage は MCP リソースサーバー登録の専用画面 (wi-314 T014)。
// 従来一覧画面にインライン表示していたフォームを専用ルートへ移す。
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
  const t = useDictionary(adminMcpResourceServersDictionary)

  async function handleSubmit(event: FormEvent) {
    event.preventDefault()
    setError('')
    setSaving(true)
    try {
      await createMcpResourceServer(csrfToken, {
        resource: form.resource,
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
      <Card className="w-full max-w-2xl p-4">
        <form className="flex flex-col gap-4" onSubmit={handleSubmit}>
          <McpResourceServerFormFields form={form} onChange={setForm} locked={false} t={t} />
          <div className="flex gap-2.5">
            <Button type="submit" disabled={saving}>
              {t.register}
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
