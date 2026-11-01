import { type FormEvent, useState } from 'react'
import {
  AuthenticationAPIError,
  bindAdminAgentCredential,
  getAdminAgent,
  tenantURL,
  unbindAdminAgentCredential,
} from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { Toast } from '../../components/ui/toast'
import { useDictionary } from '../../lib/i18n'
import type { AdminAgent } from '../../types'
import { AgentActionConfirmation } from './AgentActionConfirmation'
import { adminAgentsDictionary } from './AdminAgentsPage.i18n'

export function AdminAgentCredentialsPage({
  csrfToken,
  actorUsername,
  agent,
}: {
  csrfToken: string
  actorUsername?: string
  agent: AdminAgent
}) {
  const t = useDictionary(adminAgentsDictionary)
  const [clientIDs, setClientIDs] = useState(agent.client_ids)
  const [clientID, setClientID] = useState('')
  const [removing, setRemoving] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const killed = agent.status === 'killed'

  async function run(action: () => Promise<void>, message: string) {
    setBusy(true)
    setError('')
    setNotice('')
    try {
      await action()
      setNotice(message)
      try {
        const next = await getAdminAgent(agent.id)
        setClientIDs(next.client_ids)
      } catch {
        setError(t.credentialsReloadFailed)
      }
    } catch (cause) {
      setError(
        cause instanceof AuthenticationAPIError
          ? cause.code === 'agent_client_auth_method_not_allowed'
            ? t.agentClientAuthMethodNotAllowedError
            : cause.message
          : t.genericOpError,
      )
    } finally {
      setBusy(false)
    }
  }

  function bind(event: FormEvent) {
    event.preventDefault()
    if (!clientID.trim() || busy || killed) return
    void run(async () => {
      await bindAdminAgentCredential(csrfToken, agent.id, clientID.trim())
      setClientID('')
    }, t.credentialBound)
  }

  return (
    <AdminShell
      active="agents"
      actorUsername={actorUsername}
      title={t.manageCredentials}
      description={agent.name}
      actions={
        <Button
          variant="outline"
          nativeButton={false}
          render={<a href={tenantURL(`/admin/agents/${encodeURIComponent(agent.id)}`)} />}
        >
          {t.agentDetail}
        </Button>
      }
    >
      <Card className="max-w-2xl space-y-5 p-6">
        <p className="text-sm text-slate-600">{t.credentialsImmediate}</p>
        {error ? <Alert variant="destructive">{error}</Alert> : null}
        <Toast message={notice} onDismiss={() => setNotice('')} />
        <ul className="space-y-2">
          {clientIDs.map((id) => (
            <li key={id} className="flex items-center justify-between gap-4 rounded-lg border p-3">
              <span className="break-all font-mono text-sm">{id}</span>
              <Button variant="outline" disabled={busy} onClick={() => setRemoving(id)}>
                {t.unbind}
              </Button>
            </li>
          ))}
          {clientIDs.length === 0 ? (
            <li className="text-sm text-slate-500">{t.noCredentialsNotice}</li>
          ) : null}
        </ul>
        <form onSubmit={bind} className="space-y-3">
          <Label htmlFor="agent-credential-client">{t.bindClientIdAria}</Label>
          <Input
            id="agent-credential-client"
            value={clientID}
            onChange={(event) => setClientID(event.target.value)}
            disabled={busy || killed}
          />
          <div className="flex justify-end">
            <Button type="submit" disabled={busy || killed || !clientID.trim()}>
              {t.bind}
            </Button>
          </div>
        </form>
      </Card>
      {removing !== null ? (
        <AgentActionConfirmation
          title={t.unbind}
          description={t.confirmUnbind.replace('{client}', removing)}
          busy={busy}
          error={error}
          onCancel={() => setRemoving(null)}
          onConfirm={() =>
            void run(async () => {
              await unbindAdminAgentCredential(csrfToken, agent.id, removing)
              setRemoving(null)
            }, t.credentialUnbound)
          }
        />
      ) : null}
    </AdminShell>
  )
}
