import { type FormEvent, useRef, useState } from 'react'
import {
  AuthenticationAPIError,
  bindAdminAgentCredential,
  createAdminAgentKeyClient,
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
import type { AdminAgent, McpResourceServer } from '../../types'
import { AgentActionConfirmation } from './AgentActionConfirmation'
import { adminAgentsDictionary } from './AdminAgentsPage.i18n'
import { parsePublicKey } from './publicKey'

// CIBA の承認の依頼に要るスコープ。
const APPROVAL_SCOPES = ['openid', 'profile']

export function AdminAgentCredentialsPage({
  csrfToken,
  actorUsername,
  agent,
  resourceServers = [],
}: {
  csrfToken: string
  actorUsername?: string
  agent: AdminAgent
  resourceServers?: McpResourceServer[]
}) {
  const t = useDictionary(adminAgentsDictionary)
  const [clientIDs, setClientIDs] = useState(agent.client_ids)
  const [clientID, setClientID] = useState('')
  const [publicKey, setPublicKey] = useState('')
  const [selectedScopes, setSelectedScopes] = useState<string[]>([])
  const [allowApproval, setAllowApproval] = useState(true)
  const [otherScopes, setOtherScopes] = useState('')
  const [removing, setRemoving] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const running = useRef(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const killed = agent.status === 'killed'

  async function run(action: () => Promise<void | false>, message: string) {
    if (running.current) return
    running.current = true
    setBusy(true)
    setError('')
    setNotice('')
    try {
      if ((await action()) === false) return
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
      running.current = false
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

  const requestedScopes = [
    ...new Set([
      ...(allowApproval ? APPROVAL_SCOPES : []),
      ...selectedScopes,
      ...otherScopes.split(/\s+/).filter(Boolean),
    ]),
  ]

  async function createFromKey(event: FormEvent) {
    event.preventDefault()
    if (busy || killed) return
    void run(async () => {
      const jwks = await parsePublicKey(publicKey)
      if (jwks === 'private' || jwks === 'unsupported' || jwks === null) {
        setError(
          jwks === 'private'
            ? t.publicKeyHasPrivateMembers
            : jwks === 'unsupported'
              ? t.publicKeyUnsupported
              : t.publicKeyInvalid,
        )
        return false
      }
      const created = await createAdminAgentKeyClient(csrfToken, {
        name: agent.name,
        scope: requestedScopes.join(' '),
        jwks,
      })
      // 束縛だけが失敗しても、作成済みのクライアントで再試行できるように残す。
      setClientID(created)
      await bindAdminAgentCredential(csrfToken, agent.id, created)
      setClientID('')
      setPublicKey('')
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
      <Card className="mt-6 max-w-2xl space-y-4 p-6">
        <div>
          <h2 className="text-base font-semibold">{t.createKeyCredentialHeading}</h2>
          <p className="mt-1 text-sm text-slate-600">{t.createKeyCredentialDescription}</p>
        </div>
        <form onSubmit={createFromKey} className="space-y-3">
          <div className="grid gap-1.5">
            <Label htmlFor="agent-public-key" required>
              {t.publicKeyLabel}
            </Label>
            <textarea
              id="agent-public-key"
              required
              rows={6}
              value={publicKey}
              onChange={(event) => setPublicKey(event.target.value)}
              disabled={busy || killed}
              className="rounded-md border border-slate-300 bg-white p-2 font-mono text-xs text-slate-900 focus:border-blue-500 focus:outline-none"
            />
          </div>
          <fieldset className="grid gap-2">
            <legend className="text-sm font-medium">{t.keyScopeLabel}</legend>
            <p className="text-xs leading-5 text-slate-500">{t.keyScopeHelp}</p>
            {resourceServers.flatMap((server) =>
              server.scopes.map((name) => (
                <label key={`${server.id}:${name}`} className="flex items-center gap-2 text-sm">
                  <input
                    type="checkbox"
                    checked={selectedScopes.includes(name)}
                    disabled={busy || killed}
                    onChange={(event) =>
                      setSelectedScopes((current) =>
                        event.target.checked
                          ? [...current, name]
                          : current.filter((item) => item !== name),
                      )
                    }
                  />
                  <code className="text-xs">{name}</code>
                  <span className="text-slate-500">{server.name}</span>
                </label>
              )),
            )}
            {resourceServers.length === 0 ? (
              <p className="text-sm text-slate-600">
                {t.keyScopeNoServers}{' '}
                <a className="underline" href={tenantURL('/admin/mcp-resource-servers/new')}>
                  {t.keyScopeRegisterServer}
                </a>
              </p>
            ) : null}
            <label className="flex items-center gap-2 text-sm">
              <input
                id="agent-allow-approval"
                type="checkbox"
                checked={allowApproval}
                disabled={busy || killed}
                onChange={(event) => setAllowApproval(event.target.checked)}
              />
              {t.keyScopeAllowApproval}
            </label>
            <details className="text-sm">
              <summary className="cursor-pointer text-slate-600">{t.keyScopeOther}</summary>
              <Input
                id="agent-key-scope"
                aria-label={t.keyScopeOther}
                className="mt-2"
                value={otherScopes}
                onChange={(event) => setOtherScopes(event.target.value)}
                disabled={busy || killed}
              />
            </details>
          </fieldset>
          <div className="flex justify-end">
            <Button
              type="submit"
              disabled={busy || killed || !publicKey.trim() || requestedScopes.length === 0}
            >
              {t.createKeyCredential}
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
