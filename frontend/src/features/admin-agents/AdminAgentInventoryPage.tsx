import { IconArrowLeft, IconRobot } from '@tabler/icons-react'
import { type FormEvent, useState } from 'react'
import { tenantURL } from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { useDictionary } from '../../lib/i18n'
import type {
  AgentInventory,
  AgentInventoryEntry,
  AgentInventoryFinding,
  AgentObservation,
} from '../../types'
import { adminAgentInventoryDictionary } from './AdminAgentInventoryPage.i18n'
import { StatusBadge } from './AdminAgentsShared'
import { AgentObservationsPanel } from './AgentObservationsPanel'

type Dictionary = (typeof adminAgentInventoryDictionary)['ja']

const FINDINGS: AgentInventoryFinding[] = ['owner_missing', 'dormant', 'unbound']

const FINDING_STYLES: Record<AgentInventoryFinding, string> = {
  owner_missing: 'bg-red-50 text-red-700',
  dormant: 'bg-amber-50 text-amber-800',
  unbound: 'bg-slate-100 text-slate-700',
}

function findingLabel(finding: AgentInventoryFinding, t: Dictionary) {
  switch (finding) {
    case 'owner_missing':
      return t.findingOwnerMissing
    case 'dormant':
      return t.findingDormant
    case 'unbound':
      return t.findingUnbound
  }
}

function ownerLabel(status: AgentInventoryEntry['owner_status'], t: Dictionary) {
  switch (status) {
    case 'active':
      return t.ownerActive
    case 'disabled':
      return t.ownerDisabled
    case 'pending_deletion':
      return t.ownerPendingDeletion
    case 'missing':
      return t.ownerMissing
  }
}

// AdminAgentInventoryPage は所有者不在、休眠、未束縛の所見を付けた Agent の一覧である
// (REQ-IDMANAGEMENT-094)。絞り込みと日数は URL の検索条件に置き、読み込みはルートが行う。
export function AdminAgentInventoryPage({
  actorUsername,
  inventory,
  csrfToken,
  observations,
  finding,
  onFilter,
}: {
  actorUsername?: string
  inventory: AgentInventory
  csrfToken: string
  observations: AgentObservation[]
  finding?: AgentInventoryFinding
  onFilter: (filter: { finding?: AgentInventoryFinding; dormantDays?: number }) => void
}) {
  const t = useDictionary(adminAgentInventoryDictionary)
  const [dormantDays, setDormantDays] = useState(String(inventory.dormant_days))

  function applyDays(event: FormEvent) {
    event.preventDefault()
    onFilter({ finding, dormantDays: Number(dormantDays) })
  }

  return (
    <AdminShell
      active="agents"
      actorUsername={actorUsername}
      title={t.pageTitle}
      description={t.pageDescription}
      actions={
        <Button
          variant="outline"
          nativeButton={false}
          render={<a href={tenantURL('/admin/agents')} />}
        >
          <IconArrowLeft size={16} aria-hidden="true" />
          {t.backToAgentList}
        </Button>
      }
    >
      <div className="flex flex-wrap items-end justify-between gap-4">
        <fieldset aria-label={t.filterAriaLabel} className="m-0 flex flex-wrap gap-2 border-0 p-0">
          {[undefined, ...FINDINGS].map((value) => (
            <Button
              key={value ?? 'all'}
              variant={finding === value ? 'default' : 'outline'}
              aria-pressed={finding === value}
              onClick={() => onFilter({ finding: value, dormantDays: inventory.dormant_days })}
            >
              {value ? findingLabel(value, t) : t.filterAll}
            </Button>
          ))}
        </fieldset>
        <form onSubmit={applyDays} className="flex items-end gap-2">
          <div className="grid gap-1.5">
            <Label htmlFor="dormant-days">{t.dormantDaysLabel}</Label>
            <Input
              id="dormant-days"
              type="number"
              min={1}
              max={365}
              required
              value={dormantDays}
              onChange={(event) => setDormantDays(event.target.value)}
              aria-describedby="dormant-days-help"
              className="w-28"
            />
          </div>
          <Button type="submit" variant="outline">
            {t.apply}
          </Button>
        </form>
      </div>
      <p id="dormant-days-help" className="text-xs text-slate-500">
        {t.dormantDaysHelp}
      </p>

      <Card className="overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-slate-50 text-left text-xs font-semibold uppercase tracking-wide text-slate-500">
            <tr>
              <th className="px-4 py-3">{t.tableHeaderAgent}</th>
              <th className="px-4 py-3">{t.tableHeaderOwner}</th>
              <th className="px-4 py-3">{t.tableHeaderLastIssued}</th>
              <th className="px-4 py-3">{t.tableHeaderFindings}</th>
            </tr>
          </thead>
          <tbody>
            {inventory.agents.map((entry) => (
              <tr key={entry.agent.id} className="border-t border-slate-100 align-top">
                <td className="px-4 py-3">
                  <a
                    className="font-semibold text-slate-900 hover:underline"
                    href={tenantURL(`/admin/agents/${encodeURIComponent(entry.agent.id)}`)}
                  >
                    {entry.agent.name}
                  </a>
                  <div className="mt-1">
                    <StatusBadge status={entry.agent.status} />
                  </div>
                </td>
                <td className="px-4 py-3 text-xs text-slate-600">
                  <div className="font-mono">{entry.agent.owner_user_id || '—'}</div>
                  <div>{ownerLabel(entry.owner_status, t)}</div>
                </td>
                <td className="px-4 py-3 text-xs text-slate-600">
                  {entry.last_token_issued_at
                    ? new Date(entry.last_token_issued_at).toLocaleString()
                    : t.neverIssued}
                </td>
                <td className="px-4 py-3">
                  {entry.findings.length === 0 ? (
                    <span className="text-xs text-slate-500">{t.noFindings}</span>
                  ) : (
                    <ul className="flex flex-wrap gap-1.5">
                      {entry.findings.map((value) => (
                        <li
                          key={value}
                          className={`rounded-full px-2 py-0.5 text-xs font-semibold ${FINDING_STYLES[value]}`}
                        >
                          {findingLabel(value, t)}
                        </li>
                      ))}
                    </ul>
                  )}
                  {entry.missing_client_ids.length > 0 ? (
                    <div className="mt-1 text-xs text-slate-500">
                      {t.missingClientsLabel}:{' '}
                      <span className="font-mono">{entry.missing_client_ids.join(', ')}</span>
                    </div>
                  ) : null}
                  {entry.confirmed_observations.length > 0 ? (
                    <div className="mt-1 text-xs text-slate-500">
                      {t.confirmedObservationsLabel}:{' '}
                      <span className="font-mono">
                        {entry.confirmed_observations.map((item) => item.subject).join(', ')}
                      </span>
                    </div>
                  ) : null}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        {inventory.agents.length === 0 ? (
          <div className="flex min-h-40 flex-col items-center justify-center px-6 text-center text-sm text-slate-500">
            <IconRobot size={24} className="text-slate-400" aria-hidden="true" />
            <p className="mt-3">{t.emptyNotice}</p>
          </div>
        ) : null}
      </Card>
      <AgentObservationsPanel
        csrfToken={csrfToken}
        agents={inventory.agents.map((entry) => entry.agent)}
        observations={observations}
      />
    </AdminShell>
  )
}
