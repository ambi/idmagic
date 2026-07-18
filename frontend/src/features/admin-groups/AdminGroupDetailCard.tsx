import {
  IconAlertTriangle,
  IconTrash,
  IconUserMinus,
  IconUserPlus,
  IconUsersGroup,
} from '@tabler/icons-react'
import { useEffect, useState } from 'react'
import {
  addAdminGroupMember,
  AuthenticationAPIError,
  deleteAdminGroup,
  getAdminGroup,
  listAdminUsers,
  removeAdminGroupMember,
  tenantURL,
} from '../../api'
import { AdminPaneActions } from '../../components/AdminPaneActions'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { useDictionary } from '../../lib/i18n'
import type { AdminGroup, AdminGroupMember, AdminUser } from '../../types'
import { adminGroupsDictionary } from './AdminGroupsPage.i18n'

export function GroupDetailCard({
  group,
  csrfToken,
  busy,
  detailHref,
  showActions = true,
  onDeleted,
}: {
  group: AdminGroup | null
  csrfToken: string
  busy: boolean
  detailHref?: string
  showActions?: boolean
  onDeleted: () => void
}) {
  const [members, setMembers] = useState<AdminGroupMember[]>([])
  const [allUsers, setAllUsers] = useState<AdminUser[]>([])
  const [addSub, setAddSub] = useState('')
  const [localBusy, setLocalBusy] = useState(false)
  const [localError, setLocalError] = useState('')
  const [confirmDelete, setConfirmDelete] = useState(false)
  const [rule, setRule] = useState(group?.dynamic_rule)
  const t = useDictionary(adminGroupsDictionary)

  useEffect(() => {
    setConfirmDelete(false)
    setLocalError('')
    if (!group) {
      setMembers([])
      return
    }
    let cancelled = false
    void Promise.all([getAdminGroup(group.id), listAdminUsers()]).then(([detail, users]) => {
      if (cancelled) return
      setMembers(detail.members)
      setAllUsers(users)
      setRule(detail.group.dynamic_rule)
    })
    return () => {
      cancelled = true
    }
  }, [group])

  if (!group) {
    return (
      <Card className="p-5">
        <p className="text-sm text-slate-500">{t.selectGroupPrompt}</p>
      </Card>
    )
  }
  const activeGroup = group

  async function withLocal(action: () => Promise<void>) {
    setLocalBusy(true)
    setLocalError('')
    try {
      await action()
    } catch (cause) {
      setLocalError(cause instanceof AuthenticationAPIError ? cause.message : t.genericOpError)
    } finally {
      setLocalBusy(false)
    }
  }

  async function reloadMembers() {
    const detail = await getAdminGroup(activeGroup.id)
    setMembers(detail.members)
  }

  const memberUserIds = new Set(members.map((m) => m.user_id))
  const addableUsers = allUsers.filter((u) => !memberUserIds.has(u.id))

  return (
    <Card className="overflow-hidden">
      <div className="border-b border-slate-200 bg-white p-5">
        <div className="flex items-start gap-3">
          <span className="flex size-11 shrink-0 items-center justify-center rounded-lg bg-slate-100 text-slate-700">
            <IconUsersGroup size={22} aria-hidden="true" />
          </span>
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-2">
              <h2 className="truncate text-lg font-semibold text-slate-950">{group.name}</h2>
              {group.scim_source && (
                <span className="rounded-md bg-blue-100 px-2 py-0.5 text-xs font-semibold text-blue-800">
                  {t.scimSyncBadge.replace('{source}', group.scim_source)}
                </span>
              )}
            </div>
            <p className="mt-0.5 truncate font-mono text-sm text-slate-500">{group.id}</p>
          </div>
        </div>

        {showActions ? (
          <div className="mt-4">
            <AdminPaneActions
              detailHref={detailHref}
              editHref={
                group ? tenantURL(`/admin/groups/${encodeURIComponent(group.id)}/edit`) : undefined
              }
              busy={busy || localBusy}
              actions={[
                {
                  label: t.deleteGroup,
                  icon: IconTrash,
                  onClick: () => setConfirmDelete(true),
                  tone: 'danger',
                },
              ]}
            />
          </div>
        ) : null}
      </div>

      {confirmDelete ? (
        <Alert
          variant="destructive"
          className="m-5 flex flex-wrap items-center justify-between gap-2"
        >
          <span>{t.confirmDeleteGroupPrompt}</span>
          <div className="flex gap-2">
            <Button variant="outline" onClick={() => setConfirmDelete(false)} disabled={localBusy}>
              {t.dismissConfirm}
            </Button>
            <Button
              variant="destructive"
              disabled={busy || localBusy}
              onClick={() =>
                void withLocal(async () => {
                  await deleteAdminGroup(csrfToken, activeGroup.id)
                  onDeleted()
                })
              }
            >
              <IconTrash size={14} aria-hidden="true" />
              {t.confirmDelete}
            </Button>
          </div>
        </Alert>
      ) : null}

      {localError ? (
        <Alert variant="destructive" className="m-5">
          {localError}
        </Alert>
      ) : null}

      <dl className="grid gap-4 p-5">
        <div>
          <dt className="text-xs font-bold uppercase tracking-normal text-slate-400">
            {t.descriptionLabel}
          </dt>
          <dd className="mt-1 text-sm text-slate-700">{group.description || '—'}</dd>
        </div>
        <div>
          <dt className="text-xs font-bold uppercase tracking-normal text-slate-400">
            {t.rolesLabel}
          </dt>
          <dd className="mt-1 flex flex-wrap gap-1.5">
            {group.roles.length > 0 ? (
              group.roles.map((role) => (
                <span
                  key={role}
                  className="rounded-md bg-slate-100 px-2 py-1 font-mono text-xs text-slate-700"
                >
                  {role}
                </span>
              ))
            ) : (
              <span className="text-sm text-slate-400">{t.noneLabel}</span>
            )}
          </dd>
        </div>
      </dl>

      {group.membership_type === 'dynamic' ? (
        <section className="border-t border-slate-100 p-5">
          <div className="flex items-center justify-between gap-3">
            <h3 className="text-xs font-bold uppercase tracking-normal text-slate-400">
              {t.dynamicRuleHeading}
            </h3>
            <span
              className={`rounded-md px-2 py-1 text-xs font-semibold ${rule?.enabled ? 'bg-emerald-100 text-emerald-800' : 'bg-slate-100 text-slate-600'}`}
            >
              {rule?.enabled ? t.ruleEnabled : t.ruleDisabled}
            </span>
          </div>
          {rule?.expression ? (
            <pre className="mt-3 whitespace-pre-wrap break-words rounded-md border border-slate-200 bg-slate-50 p-3 font-mono text-sm text-slate-700">
              {rule.expression}
            </pre>
          ) : (
            <p className="mt-3 text-sm text-slate-400">{t.dynamicRuleNoneNotice}</p>
          )}
          <p className="mt-2 text-xs text-slate-500">{t.dynamicRuleReadOnlyHint}</p>
        </section>
      ) : null}

      <section className="border-t border-slate-100 p-5">
        <h3 className="text-xs font-bold uppercase tracking-normal text-slate-400">
          {t.membersHeading.replace('{count}', String(members.length))}
        </h3>
        <ul className="mt-3 grid gap-2">
          {members.map((member) => (
            <li
              key={member.user_id}
              className="flex items-center justify-between rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm"
            >
              <a
                className="font-medium text-blue-700 hover:underline"
                href={tenantURL(
                  `/admin/users?role=${encodeURIComponent(member.preferred_username)}`,
                )}
              >
                {member.preferred_username}
              </a>
              {member.source === 'dynamic_rule' ? (
                <span className="rounded bg-blue-50 px-2 py-0.5 text-xs font-semibold text-blue-700">
                  {t.dynamicMembership}
                </span>
              ) : null}
              <Button
                variant="ghost"
                className="text-rose-700 hover:bg-rose-50"
                disabled={localBusy || !!group.scim_source || group.membership_type === 'dynamic'}
                onClick={() =>
                  withLocal(async () => {
                    await removeAdminGroupMember(csrfToken, group.id, member.user_id)
                    await reloadMembers()
                  })
                }
              >
                <IconUserMinus size={14} aria-hidden="true" />
                {t.removeMember}
              </Button>
            </li>
          ))}
          {members.length === 0 ? <li className="text-xs text-slate-400">{t.noMembers}</li> : null}
        </ul>

        <div className="mt-3 flex items-center gap-2">
          <select
            value={addSub}
            onChange={(e) => setAddSub(e.target.value)}
            disabled={!!group.scim_source || group.membership_type === 'dynamic'}
            className="h-9 flex-1 rounded-md border border-slate-300 bg-white px-2 text-sm disabled:opacity-50 disabled:bg-slate-50"
            aria-label={t.selectUserToAddAria}
          >
            <option value="">{t.selectUserPlaceholder}</option>
            {addableUsers.map((user) => (
              <option key={user.id} value={user.id}>
                {user.preferred_username}
              </option>
            ))}
          </select>
          <Button
            disabled={
              localBusy || !addSub || !!group.scim_source || group.membership_type === 'dynamic'
            }
            onClick={() =>
              withLocal(async () => {
                await addAdminGroupMember(csrfToken, group.id, addSub)
                setAddSub('')
                await reloadMembers()
              })
            }
          >
            <IconUserPlus size={14} aria-hidden="true" />
            {t.add}
          </Button>
        </div>

        {group.scim_source && (
          <p className="mt-3 text-xs text-blue-700 flex items-center gap-1.5">
            <IconAlertTriangle size={14} />
            <span>{t.scimGroupManagedNotice}</span>
          </p>
        )}
      </section>
    </Card>
  )
}
