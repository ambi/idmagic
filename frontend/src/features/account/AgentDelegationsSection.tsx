import { IconRobot, IconTrash } from '@tabler/icons-react'
import { type FormEvent, useState } from 'react'
import {
  AuthenticationAPIError,
  grantAccountAgentDelegation,
  revokeAccountAgentDelegation,
} from '../../api'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { Toast } from '../../components/ui/toast'
import { useDictionary, useFormatters } from '../../lib/i18n'
import type { AccountAgentDelegation, DelegableAgent } from '../../types'
import { agentDelegationsDictionary } from './AgentDelegationsSection.i18n'

export function AgentDelegationsSection({
  csrfToken,
  delegations: initial,
  delegableAgents,
}: {
  csrfToken: string
  delegations: AccountAgentDelegation[]
  delegableAgents: DelegableAgent[]
}) {
  const t = useDictionary(agentDelegationsDictionary)
  const { formatDate } = useFormatters()
  const [delegations, setDelegations] = useState(initial)
  const [agentId, setAgentId] = useState('')
  const [scopes, setScopes] = useState('')
  const [pending, setPending] = useState('')
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')

  async function run(key: string, failure: string, action: () => Promise<string>) {
    setPending(key)
    setError('')
    setNotice('')
    try {
      setNotice(await action())
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : failure)
    } finally {
      setPending('')
    }
  }

  function handleGrant(event: FormEvent) {
    event.preventDefault()
    void run('grant', t.grantFailed, async () => {
      const granted = await grantAccountAgentDelegation(
        csrfToken,
        agentId,
        scopes.split(/\s+/).filter(Boolean),
      )
      setDelegations((current) => [
        ...current.filter((item) => item.agent_id !== granted.agent_id),
        granted,
      ])
      setAgentId('')
      setScopes('')
      return t.grantedNotice.replace('{name}', granted.agent_name)
    })
  }

  function handleRevoke(delegation: AccountAgentDelegation) {
    void run(delegation.agent_id, t.revokeFailed, async () => {
      await revokeAccountAgentDelegation(csrfToken, delegation.agent_id)
      setDelegations((current) => current.filter((item) => item.agent_id !== delegation.agent_id))
      return t.revokedNotice.replace('{name}', delegation.agent_name)
    })
  }

  return (
    <section className="mt-8 grid gap-3" aria-labelledby="agent-delegations-heading">
      <div>
        <h2 id="agent-delegations-heading" className="text-base font-semibold text-slate-900">
          {t.heading}
        </h2>
        <p className="mt-1 text-sm text-slate-500">{t.description}</p>
      </div>
      <Toast message={notice} onDismiss={() => setNotice('')} />
      {error ? <Alert variant="destructive">{error}</Alert> : null}

      {delegations.length === 0 ? (
        <Card className="flex flex-col items-center gap-2 p-10 text-center">
          <IconRobot size={28} className="text-slate-300" aria-hidden="true" />
          <p className="text-sm text-slate-500">{t.empty}</p>
        </Card>
      ) : (
        delegations.map((delegation) => (
          <Card key={delegation.agent_id} className="flex flex-wrap items-start gap-4 p-5">
            <div className="min-w-0 flex-1">
              <p
                className="truncate text-sm font-semibold text-slate-900"
                title={delegation.agent_id}
              >
                {delegation.agent_name}
              </p>
              <p className="mt-0.5 text-xs text-slate-500">
                {delegation.owner_name
                  ? `${t.owner.replace('{name}', delegation.owner_name)} · `
                  : ''}
                {t.expires.replace('{date}', formatDate(delegation.expires_at))}
                {' · '}
                {delegation.last_used_at
                  ? t.lastUsed.replace('{date}', formatDate(delegation.last_used_at))
                  : t.neverUsed}
              </p>
              <div className="mt-2 flex flex-wrap gap-1.5">
                {delegation.scopes.map((scope) => (
                  <span
                    key={scope}
                    className="rounded-md bg-slate-100 px-2 py-0.5 font-mono text-xs text-slate-600"
                  >
                    {scope}
                  </span>
                ))}
              </div>
            </div>
            <Button
              type="button"
              variant="outline"
              className="text-red-700 hover:bg-red-50"
              disabled={pending === delegation.agent_id}
              onClick={() => handleRevoke(delegation)}
            >
              <IconTrash size={16} aria-hidden="true" />
              {pending === delegation.agent_id ? t.revoking : t.revoke}
            </Button>
          </Card>
        ))
      )}

      <Card className="p-5">
        <form className="grid gap-3" onSubmit={handleGrant}>
          <h3 className="text-sm font-semibold text-slate-900">{t.grantHeading}</h3>
          <div className="grid gap-1.5">
            <Label htmlFor="delegation-agent">{t.agentLabel}</Label>
            <select
              id="delegation-agent"
              className="h-9 rounded-md border border-slate-200 bg-white px-3 text-sm"
              value={agentId}
              required
              onChange={(event) => setAgentId(event.target.value)}
            >
              <option value="">{t.agentPlaceholder}</option>
              {delegableAgents.map((agent) => (
                <option key={agent.agent_id} value={agent.agent_id}>
                  {agent.owner_name ? `${agent.name} (${agent.owner_name})` : agent.name}
                </option>
              ))}
            </select>
          </div>
          <div className="grid gap-1.5">
            <Label htmlFor="delegation-scopes">{t.scopesLabel}</Label>
            <Input
              id="delegation-scopes"
              value={scopes}
              required
              onChange={(event) => setScopes(event.target.value)}
            />
          </div>
          <div>
            <Button type="submit" disabled={pending === 'grant'}>
              {pending === 'grant' ? t.granting : t.grant}
            </Button>
          </div>
        </form>
      </Card>
    </section>
  )
}
