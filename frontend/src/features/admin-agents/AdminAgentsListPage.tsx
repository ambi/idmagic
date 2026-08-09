import { IconPlus, IconRefresh, IconRobot } from '@tabler/icons-react'
import { useState } from 'react'
import { AuthenticationAPIError, listAdminAgentsPage, tenantURL } from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { LoadMoreButton } from '../../components/ui/load-more'
import { Toast } from '../../components/ui/toast'
import { useDictionary } from '../../lib/i18n'
import { commonDictionary } from '../../lib/i18n/common.i18n'
import { usePaginatedList } from '../../lib/usePaginatedList'
import type { AdminAgent } from '../../types'
import { AgentDetailCard } from './AdminAgentDetailCard'
import { adminAgentsDictionary } from './AdminAgentsPage.i18n'
import { kindLabel, StatusBadge } from './AdminAgentsShared'

export function AdminAgentsPage({
  csrfToken,
  actorUsername,
  agents: initial,
  nextCursor: initialNextCursor,
}: {
  csrfToken: string
  actorUsername?: string
  agents: AdminAgent[]
  nextCursor: string | null
}) {
  const {
    items: agents,
    hasMore,
    loadingMore,
    loadMore,
    reset: resetAgents,
  } = usePaginatedList<AdminAgent>({ items: initial, nextCursor: initialNextCursor })
  const initialID = new URLSearchParams(window.location.search).get('agent')
  const [selectedID, setSelectedID] = useState<string>(
    () => initial.find((a) => a.id === initialID)?.id ?? initial[0]?.id ?? '',
  )
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const t = useDictionary(adminAgentsDictionary)
  const tCommon = useDictionary(commonDictionary)

  const selected = agents.find((a) => a.id === selectedID) ?? null

  async function refresh(preferredID = selectedID) {
    const next = await listAdminAgentsPage()
    resetAgents({ items: next.agents, nextCursor: next.nextCursor })
    setSelectedID(next.agents.find((a) => a.id === preferredID)?.id ?? next.agents[0]?.id ?? '')
  }

  async function run(action: () => Promise<void>, success: string) {
    setBusy(true)
    setError('')
    setNotice('')
    try {
      await action()
      setNotice(success)
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.genericActionError)
    } finally {
      setBusy(false)
    }
  }

  async function handleLoadMore() {
    setError('')
    try {
      await loadMore(async (cursor) => {
        const page = await listAdminAgentsPage({ cursor })
        return { items: page.agents, nextCursor: page.nextCursor }
      })
    } catch (cause) {
      setError(
        cause instanceof AuthenticationAPIError ? cause.message : tCommon.loadMoreFailedError,
      )
    }
  }

  return (
    <AdminShell
      active="agents"
      actorUsername={actorUsername}
      title={t.pageTitle}
      description={t.pageDescription}
      actions={
        <>
          <Button
            variant="outline"
            className="size-9 px-0"
            aria-label={t.reloadAriaLabel}
            onClick={() => run(() => refresh(), t.listRefreshedNotice)}
            disabled={busy}
          >
            <IconRefresh size={16} aria-hidden="true" />
          </Button>
          <Button
            disabled={busy}
            nativeButton={false}
            render={<a href={tenantURL('/admin/agents/new')} />}
          >
            <IconPlus size={16} aria-hidden="true" />
            {t.addAgent}
          </Button>
        </>
      }
    >
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      <Toast message={notice} onDismiss={() => setNotice('')} />

      <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_440px]">
        <Card className="overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-slate-50 text-left text-xs font-semibold uppercase tracking-wide text-slate-500">
              <tr>
                <th className="px-4 py-3">{t.tableHeaderAgent}</th>
                <th className="px-4 py-3">{t.tableHeaderKind}</th>
                <th className="px-4 py-3">{t.tableHeaderOwner}</th>
                <th className="px-4 py-3">{t.tableHeaderStatus}</th>
                <th className="px-4 py-3 text-right">{t.tableHeaderRolesCredentials}</th>
              </tr>
            </thead>
            <tbody>
              {agents.map((agent) => (
                <tr
                  key={agent.id}
                  onClick={() => setSelectedID(agent.id)}
                  className={`cursor-pointer border-t border-slate-100 hover:bg-slate-50 ${
                    selectedID === agent.id ? 'bg-blue-50/60' : ''
                  }`}
                >
                  <td className="px-4 py-3">
                    <div className="font-semibold text-slate-900">{agent.name}</div>
                    {agent.description ? (
                      <div className="truncate text-xs text-slate-500">{agent.description}</div>
                    ) : null}
                  </td>
                  <td className="px-4 py-3 text-xs text-slate-600">{kindLabel(agent.kind, t)}</td>
                  <td className="px-4 py-3 font-mono text-xs text-slate-600">
                    {agent.owner_user_id || '—'}
                  </td>
                  <td className="px-4 py-3">
                    <StatusBadge status={agent.status} />
                  </td>
                  <td className="px-4 py-3 text-right text-xs text-slate-600">
                    {agent.roles.length} / {agent.client_ids.length}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          {agents.length === 0 ? (
            <div className="flex min-h-40 flex-col items-center justify-center px-6 text-center text-sm text-slate-500">
              <IconRobot size={24} className="text-slate-400" aria-hidden="true" />
              <p className="mt-3">{t.emptyAgentsNotice}</p>
            </div>
          ) : null}
          <LoadMoreButton hasMore={hasMore} loading={loadingMore} onClick={handleLoadMore} />
        </Card>

        <AgentDetailCard
          key={selected?.id}
          agent={selected}
          csrfToken={csrfToken}
          busy={busy}
          detailHref={
            selected ? tenantURL(`/admin/agents/${encodeURIComponent(selected.id)}`) : undefined
          }
          onDeleted={() => run(() => refresh(), t.agentDeletedNotice)}
        />
      </div>
    </AdminShell>
  )
}
