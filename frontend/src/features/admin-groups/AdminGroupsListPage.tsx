import {
  IconFileExport,
  IconFileImport,
  IconRefresh,
  IconUsersGroup,
  IconUsersPlus,
} from '@tabler/icons-react'
import { useEffect, useState } from 'react'
import { AuthenticationAPIError, listAdminGroupsPage, tenantURL } from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { PageNavigation, type PageNavigationData } from '../../components/ui/page-navigation'
import { Toast } from '../../components/ui/toast'
import { useDictionary } from '../../lib/i18n'
import { commonDictionary } from '../../lib/i18n/common.i18n'
import type { AdminGroup } from '../../types'
import { GroupDetailCard } from './AdminGroupDetailCard'
import { adminGroupsDictionary } from './AdminGroupsPage.i18n'

export function AdminGroupsPage({
  csrfToken,
  actorUsername,
  groups: initial,
  pagination,
  cursor,
  hasFirst = false,
  previousCursor = null,
  nextCursor: initialNextCursor,
  lastCursor = null,
  totalItems = initial.length,
  totalPages = initial.length > 0 ? 1 : 0,
  currentPage = initial.length > 0 ? 1 : 0,
  onPage,
  cursorReset = false,
}: {
  csrfToken: string
  actorUsername?: string
  groups: AdminGroup[]
  pagination?: PageNavigationData
  cursor?: string
  hasFirst?: boolean
  previousCursor?: string | null
  nextCursor: string | null
  lastCursor?: string | null
  totalItems?: number
  totalPages?: number
  currentPage?: number
  onPage?: (cursor: string | null) => void
  cursorReset?: boolean
}) {
  const [groups, setGroups] = useState(initial)
  const [page, setPage] = useState(pagination)
  const initialID = new URLSearchParams(window.location.search).get('group')
  const [selectedID, setSelectedID] = useState<string>(
    () => initial.find((g) => g.id === initialID)?.id ?? initial[0]?.id ?? '',
  )
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const t = useDictionary(adminGroupsDictionary)
  const tCommon = useDictionary(commonDictionary)

  useEffect(() => {
    if (cursorReset) setNotice(tCommon.cursorResetNotice)
  }, [cursorReset, tCommon.cursorResetNotice])

  const selected = groups.find((g) => g.id === selectedID) ?? null

  async function refresh(preferredID = selectedID) {
    const next = await listAdminGroupsPage({ cursor })
    setGroups(next.groups)
    setPage(next)
    setSelectedID(next.groups.find((g) => g.id === preferredID)?.id ?? next.groups[0]?.id ?? '')
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
          <Button
            variant="outline"
            disabled={busy}
            nativeButton={false}
            render={<a href={tenantURL('/admin/groups/exports')} />}
          >
            <IconFileExport size={16} aria-hidden="true" />
            {t.exportGroups}
          </Button>
          <Button
            variant="outline"
            disabled={busy}
            nativeButton={false}
            render={<a href={tenantURL('/admin/groups/import')} />}
          >
            <IconFileImport size={16} aria-hidden="true" />
            {t.importGroups}
          </Button>
          <Button
            disabled={busy}
            nativeButton={false}
            render={<a href={tenantURL('/admin/groups/new')} />}
          >
            <IconUsersPlus size={16} aria-hidden="true" />
            {t.newGroup}
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
          {onPage ? (
            <PageNavigation
              hasFirst={hasFirst}
              previousCursor={previousCursor}
              nextCursor={initialNextCursor}
              lastCursor={lastCursor}
              totalItems={totalItems}
              totalPages={totalPages}
              currentPage={currentPage}
              {...page}
              onNavigate={onPage}
            />
          ) : null}
        </Card>

        <GroupDetailCard
          group={selected}
          csrfToken={csrfToken}
          busy={busy}
          allowEditing={false}
          detailHref={
            selected ? tenantURL(`/admin/groups/${encodeURIComponent(selected.id)}`) : undefined
          }
          onDeleted={() => run(() => refresh(), t.groupDeletedNotice)}
        />
      </div>
    </AdminShell>
  )
}
