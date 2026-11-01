import { tenantURL } from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { useDictionary } from '../../lib/i18n'
import type { AdminAgent } from '../../types'
import { AgentGuardrailPanel } from './AgentGuardrailPanel'
import { agentGuardrailDictionary } from './AgentGuardrailPanel.i18n'
import { returnToAgent } from './agentNavigation'

export function AdminAgentGuardrailEditPage({
  csrfToken,
  actorUsername,
  agent,
}: {
  csrfToken: string
  actorUsername?: string
  agent: AdminAgent
}) {
  const t = useDictionary(agentGuardrailDictionary)
  return (
    <AdminShell
      active="agents"
      actorUsername={actorUsername}
      title={t.editTitle}
      description={agent.name}
    >
      <AgentGuardrailPanel
        agentID={agent.id}
        agentName={agent.name}
        csrfToken={csrfToken}
        cancelHref={tenantURL(`/admin/agents/${encodeURIComponent(agent.id)}`)}
        onSaved={(result) => returnToAgent(agent.id, result === 'saved' ? t.saved : t.removed)}
      />
    </AdminShell>
  )
}
