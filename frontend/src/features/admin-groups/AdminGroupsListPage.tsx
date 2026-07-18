import { IconPlus, IconRefresh, IconUsersGroup } from '@tabler/icons-react'
import { useState } from 'react'
import { AuthenticationAPIError, listAdminGroups, tenantURL } from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Toast } from '../../components/ui/toast'
import { useDictionary } from '../../lib/i18n'
import type { AdminGroup } from '../../types'
import { GroupDetailCard } from './AdminGroupDetailCard'
import { adminGroupsDictionary } from './AdminGroupsPage.i18n'

export function AdminGroupsPage({
  csrfToken,
  actorUsername,
  groups: initial,
}: {
  csrfToken: string
  actorUsername?: string
  groups: AdminGroup[]
}) {
  const [groups, setGroups] = useState(initial)
  const initialID = new URLSearchParams(window.location.search).get('group')
  const [selectedID, setSelectedID] = useState<string>(
    () => initial.find((g) => g.id === initialID)?.id ?? initial[0]?.id ?? '',
  )
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const t = useDictionary(adminGroupsDictionary)

  const selected = groups.find((g) => g.id === selectedID) ?? null

  async function refresh(preferredID = selectedID) {
    const next = await listAdminGroups()
    setGroups(next)
    setSelectedID(next.find((g) => g.id === preferredID)?.id ?? next[0]?.id ?? '')
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

  return (
    <AdminShell
      active="groups"
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
          <Button asChild disabled={busy}>
            <a href={tenantURL('/admin/groups/new')}>
              <IconPlus size={16} aria-hidden="true" />
              {t.newGroup}
            </a>
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
                <th className="px-4 py-3">{t.tableHeaderGroup}</th>
                <th className="px-4 py-3">{t.tableHeaderRoles}</th>
                <th className="px-4 py-3 text-right">{t.tableHeaderMembers}</th>
              </tr>
            </thead>
            <tbody>
              {groups.map((group) => (
                <tr
                  key={group.id}
                  onClick={() => setSelectedID(group.id)}
                  className={`cursor-pointer border-t border-slate-100 hover:bg-slate-50 ${
                    selectedID === group.id ? 'bg-blue-50/60' : ''
                  }`}
                >
                  <td className="px-4 py-3">
                    <div className="font-semibold text-slate-900">{group.name}</div>
                    {group.description ? (
                      <div className="truncate text-xs text-slate-500">{group.description}</div>
                    ) : null}
                  </td>
                  <td className="px-4 py-3 text-xs text-slate-600">
                    {t.rolesCount.replace('{count}', String(group.roles.length))}
                  </td>
                  <td className="px-4 py-3 text-right text-xs text-slate-600">
                    {group.member_count}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          {groups.length === 0 ? (
            <div className="flex min-h-40 flex-col items-center justify-center px-6 text-center text-sm text-slate-500">
              <IconUsersGroup size={24} className="text-slate-400" aria-hidden="true" />
              <p className="mt-3">{t.emptyGroupsNotice}</p>
            </div>
          ) : null}
        </Card>

        <GroupDetailCard
          group={selected}
          csrfToken={csrfToken}
          busy={busy}
          detailHref={
            selected ? tenantURL(`/admin/groups/${encodeURIComponent(selected.id)}`) : undefined
          }
          onDeleted={() => run(() => refresh(), t.groupDeletedNotice)}
        />
      </div>
    </AdminShell>
  )
}
