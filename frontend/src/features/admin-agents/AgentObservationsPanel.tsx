import { useState } from 'react'
import {
  confirmAgentObservation,
  excludeAgentObservation,
  listAgentObservations,
  reopenAgentObservation,
  tenantURL,
} from '../../api'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { useDictionary } from '../../lib/i18n'
import type {
  AdminAgent,
  AgentObservation,
  AgentObservationDecision,
  AgentObservationMatchReason,
  AgentObservationState,
} from '../../types'
import { agentObservationsDictionary } from './AgentObservationsPanel.i18n'

type Dictionary = (typeof agentObservationsDictionary)['ja']

const DECISIONS: AgentObservationDecision[] = ['pending', 'confirmed', 'excluded']

function decisionLabel(decision: AgentObservationDecision, t: Dictionary) {
  switch (decision) {
    case 'pending':
      return t.decisionPending
    case 'confirmed':
      return t.decisionConfirmed
    case 'excluded':
      return t.decisionExcluded
  }
}

function stateLabel(state: AgentObservationState, t: Dictionary) {
  switch (state) {
    case 'unregistered':
      return t.stateUnregistered
    case 'candidate':
      return t.stateCandidate
    case 'confirmed':
      return t.stateConfirmed
    case 'excluded':
      return t.stateExcluded
  }
}

function reasonLabel(reason: AgentObservationMatchReason, t: Dictionary) {
  switch (reason) {
    case 'workload_binding':
      return t.reasonWorkloadBinding
    case 'client_binding':
      return t.reasonClientBinding
  }
}

// AgentObservationsPanel は外部の発見器が報告した未確認の観測を、管理済みのエージェントと分けて
// 示す (REQ-IDMANAGEMENT-098)。確認と除外は決定だけを変え、権限を付与しない (REQ-IDMANAGEMENT-101)。
export function AgentObservationsPanel({
  csrfToken,
  agents,
  observations: initial,
}: {
  csrfToken: string
  agents: AdminAgent[]
  observations: AgentObservation[]
}) {
  const t = useDictionary(agentObservationsDictionary)
  const [decision, setDecision] = useState<AgentObservationDecision>('pending')
  const [observations, setObservations] = useState(initial)
  const [selected, setSelected] = useState<Record<string, string>>({})
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  async function load(next: AgentObservationDecision) {
    setObservations(await listAgentObservations({ decision: next }))
    setDecision(next)
  }

  async function run(action: () => Promise<unknown>, nextDecision = decision) {
    setBusy(true)
    setError('')
    try {
      await action()
      await load(nextDecision)
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : t.actionFailedError)
    } finally {
      setBusy(false)
    }
  }

  return (
    <section aria-label={t.sectionTitle} className="grid gap-3">
      <div>
        <h2 className="text-base font-semibold text-slate-900">{t.sectionTitle}</h2>
        <p className="mt-1 text-xs text-slate-500">{t.sectionDescription}</p>
      </div>
      <fieldset aria-label={t.filterAriaLabel} className="m-0 flex flex-wrap gap-2 border-0 p-0">
        {DECISIONS.map((value) => (
          <Button
            key={value}
            variant={decision === value ? 'default' : 'outline'}
            aria-pressed={decision === value}
            disabled={busy}
            onClick={() => run(async () => {}, value)}
          >
            {decisionLabel(value, t)}
          </Button>
        ))}
      </fieldset>
      {error ? (
        <p role="alert" className="text-sm text-red-700">
          {error}
        </p>
      ) : null}
      <Card className="overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-slate-50 text-left text-xs font-semibold uppercase tracking-wide text-slate-500">
            <tr>
              <th className="px-4 py-3">{t.tableHeaderSubject}</th>
              <th className="px-4 py-3">{t.tableHeaderSource}</th>
              <th className="px-4 py-3">{t.tableHeaderState}</th>
              <th className="px-4 py-3">{t.tableHeaderActions}</th>
            </tr>
          </thead>
          <tbody>
            {observations.map((observation) => {
              const agentID = selected[observation.id] ?? observation.candidates[0]?.agent_id ?? ''
              return (
                <tr key={observation.id} className="border-t border-slate-100 align-top">
                  <td className="px-4 py-3">
                    <div className="font-semibold text-slate-900">
                      {observation.display_name ?? observation.subject}
                    </div>
                    <div className="font-mono text-xs text-slate-600">{observation.subject}</div>
                    <div className="font-mono text-xs text-slate-500">{observation.issuer}</div>
                    {observation.responsible_candidate ? (
                      <div className="mt-1 text-xs text-slate-500">
                        {t.responsibleCandidateLabel}: {observation.responsible_candidate}
                      </div>
                    ) : null}
                  </td>
                  <td className="px-4 py-3 text-xs text-slate-600">
                    <div className="font-mono">{observation.source_token_id}</div>
                    <div>{new Date(observation.observed_at).toLocaleString()}</div>
                    {observation.stale ? (
                      <div className="font-semibold text-amber-800">{t.staleLabel}</div>
                    ) : null}
                  </td>
                  <td className="px-4 py-3 text-xs text-slate-600">
                    <div className="font-semibold">{stateLabel(observation.state, t)}</div>
                    {observation.confirmed_agent_id ? (
                      <div className="font-mono">{observation.confirmed_agent_id}</div>
                    ) : null}
                    {observation.candidates.length > 1 ? (
                      <div className="font-semibold text-amber-800">{t.ambiguousLabel}</div>
                    ) : null}
                    <ul className="mt-1 grid gap-0.5">
                      {observation.candidates.map((candidate) => (
                        <li key={candidate.agent_id}>
                          {candidate.agent_name}:{' '}
                          {candidate.reasons.map((reason) => reasonLabel(reason, t)).join(', ')}
                        </li>
                      ))}
                    </ul>
                  </td>
                  <td className="px-4 py-3">
                    {observation.state === 'confirmed' || observation.state === 'excluded' ? (
                      <Button
                        variant="outline"
                        disabled={busy}
                        onClick={() => run(() => reopenAgentObservation(csrfToken, observation.id))}
                      >
                        {t.reopenAction}
                      </Button>
                    ) : (
                      <div className="grid gap-2">
                        <select
                          aria-label={t.agentSelectLabel}
                          className="rounded border border-slate-300 px-2 py-1 text-xs"
                          value={agentID}
                          onChange={(event) =>
                            setSelected({ ...selected, [observation.id]: event.target.value })
                          }
                        >
                          <option value="">{t.agentSelectPlaceholder}</option>
                          {agents.map((agent) => (
                            <option key={agent.id} value={agent.id}>
                              {agent.name}
                            </option>
                          ))}
                        </select>
                        <div className="flex flex-wrap gap-2">
                          <Button
                            disabled={busy || !agentID}
                            onClick={() =>
                              run(() => confirmAgentObservation(csrfToken, observation.id, agentID))
                            }
                          >
                            {t.confirmAction}
                          </Button>
                          <Button
                            variant="outline"
                            disabled={busy}
                            onClick={() =>
                              run(() => excludeAgentObservation(csrfToken, observation.id))
                            }
                          >
                            {t.excludeAction}
                          </Button>
                          <a
                            className="self-center text-xs text-slate-700 underline"
                            href={tenantURL('/admin/agents')}
                          >
                            {t.registerLink}
                          </a>
                        </div>
                      </div>
                    )}
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
        {observations.length === 0 ? (
          <p className="px-6 py-10 text-center text-sm text-slate-500">{t.emptyNotice}</p>
        ) : null}
      </Card>
      <p className="text-xs text-slate-500">{t.pageLimitNotice}</p>
    </section>
  )
}
