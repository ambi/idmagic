import { IconArrowLeft, IconDotsVertical, IconPencil, IconTrash } from '@tabler/icons-react'
import { useState } from 'react'
import { AuthenticationAPIError, deleteAdminGroup, tenantURL } from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '../../components/ui/dropdown-menu'
import { useDictionary } from '../../lib/i18n'
import type { AdminGroup } from '../../types'
import { GroupDetailCard } from './AdminGroupDetailCard'
import { adminGroupsDictionary } from './AdminGroupsPage.i18n'

// AdminGroupDetailPage はグループの編集・メンバー管理を扱う専用詳細画面 (wi-39)。
export function AdminGroupDetailPage({
  csrfToken,
  actorUsername,
  group,
}: {
  csrfToken: string
  actorUsername?: string
  group: AdminGroup
}) {
  const [confirmDelete, setConfirmDelete] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const t = useDictionary(adminGroupsDictionary)

  async function handleDelete() {
    setBusy(true)
    setError('')
    try {
      await deleteAdminGroup(csrfToken, group.id)
      window.location.assign(tenantURL('/admin/groups'))
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.groupDeleteFailedError)
      setBusy(false)
    }
  }

  return (
    <AdminShell
      active="groups"
      actorUsername={actorUsername}
      title={group.name}
      description={group.description || group.id}
      actions={
        <div className="flex items-center gap-2">
          <a
            href={tenantURL('/admin/groups')}
            className="inline-flex items-center gap-1.5 rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm font-semibold text-slate-700 transition-colors hover:bg-slate-50"
          >
            <IconArrowLeft size={16} aria-hidden="true" />
            {t.backToGroupList}
          </a>
          <Button type="button" disabled={busy} asChild>
            <a href={tenantURL(`/admin/groups/${encodeURIComponent(group.id)}/edit`)}>
              <IconPencil size={16} aria-hidden="true" />
              {t.edit}
            </a>
          </Button>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button
                type="button"
                variant="outline"
                className="size-9 px-0"
                aria-label={t.groupActionsAriaLabel}
                disabled={busy}
              >
                <IconDotsVertical size={18} aria-hidden="true" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem className="text-red-700" onSelect={() => setConfirmDelete(true)}>
                <IconTrash size={17} aria-hidden="true" />
                {t.deleteGroup}
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      }
    >
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      {confirmDelete ? (
        <Alert variant="destructive" className="flex flex-wrap items-center justify-between gap-2">
          <span>{t.confirmDeleteGroupPrompt}</span>
          <div className="flex gap-2">
            <Button variant="outline" onClick={() => setConfirmDelete(false)} disabled={busy}>
              {t.dismissConfirm}
            </Button>
            <Button variant="destructive" disabled={busy} onClick={() => void handleDelete()}>
              <IconTrash size={14} aria-hidden="true" />
              {t.confirmDelete}
            </Button>
          </div>
        </Alert>
      ) : null}
      <div className="max-w-3xl">
        <GroupDetailCard
          group={group}
          csrfToken={csrfToken}
          busy={busy}
          showActions={false}
          onDeleted={() => window.location.assign(tenantURL('/admin/groups'))}
        />
      </div>
    </AdminShell>
  )
}
