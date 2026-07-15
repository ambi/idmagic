import {
  IconArrowLeft,
  IconDotsVertical,
  IconPencil,
  IconPlus,
  IconRefresh,
  IconTrash,
  IconUserMinus,
  IconUserPlus,
  IconUsersGroup,
  IconAlertTriangle,
} from '@tabler/icons-react'
import { type FormEvent, useEffect, useState } from 'react'
import {
  addAdminGroupMember,
  AuthenticationAPIError,
  createAdminGroup,
  deleteAdminGroup,
  getAdminGroup,
  listAdminGroups,
  listAdminUsers,
  previewDynamicGroupRule,
  removeAdminGroupMember,
  setDynamicGroupRuleEnabled,
  tenantURL,
  updateAdminGroup,
  updateDynamicGroupRule,
} from '../../api'
import { AdminPaneActions } from '../../components/AdminPaneActions'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Toast } from '../../components/ui/toast'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '../../components/ui/dropdown-menu'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { useDictionary } from '../../lib/i18n'
import type { AdminGroup, AdminGroupMember, AdminUser, DynamicGroupPreview } from '../../types'
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

function GroupDetailCard({
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

export function AdminGroupCreatePage({
  csrfToken,
  actorUsername,
}: {
  csrfToken: string
  actorUsername?: string
}) {
  const listPath = tenantURL('/admin/groups')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [membershipType, setMembershipType] = useState<'manual' | 'dynamic'>('manual')
  const t = useDictionary(adminGroupsDictionary)

  async function handleSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    const form = e.currentTarget
    const data = new FormData(form)
    const name = String(data.get('name') ?? '').trim()
    if (!name) return

    setBusy(true)
    setError('')
    try {
      const created = await createAdminGroup(csrfToken, {
        name,
        description: optionalValue(data.get('description')),
        roles: parseRoles(String(data.get('roles') ?? '')),
        membership_type: membershipType,
        dynamic_rule:
          membershipType === 'dynamic'
            ? { expression: String(data.get('dynamic-rule') ?? '').trim() }
            : undefined,
      })
      window.location.assign(tenantURL(`/admin/groups/${encodeURIComponent(created.id)}`))
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.groupCreateFailedError)
      setBusy(false)
    }
  }

  return (
    <AdminShell
      active="groups"
      actorUsername={actorUsername}
      title={t.addGroup}
      description={t.addGroupDescription}
    >
      <div className="flex items-center gap-3">
        <a
          href={listPath}
          className="inline-flex size-9 items-center justify-center rounded-lg border border-slate-200 bg-white text-slate-700 transition hover:bg-slate-50 hover:text-slate-900"
          aria-label={t.backToGroupListAria}
        >
          <IconArrowLeft size={18} aria-hidden="true" />
        </a>
        <h1 className="text-2xl font-bold tracking-tight text-slate-900">{t.addGroup}</h1>
      </div>

      <div className="mt-6 max-w-2xl">
        <Card className="shadow-[0_1px_2px_rgb(15_23_42/4%)]">
          <form onSubmit={handleSubmit}>
            <div className="grid gap-6 p-6">
              {error && <Alert variant="destructive">{error}</Alert>}

              <div className="grid gap-1.5">
                <Label htmlFor="group-name">
                  {t.groupNameLabel} <span className="text-red-500">*</span>
                </Label>
                <Input id="group-name" name="name" required placeholder="engineering" />
                <p className="text-xs text-slate-500">{t.groupNameHelp}</p>
              </div>

              <div className="grid gap-1.5">
                <Label htmlFor="group-membership-type">{t.membershipType}</Label>
                <select
                  id="group-membership-type"
                  value={membershipType}
                  onChange={(event) =>
                    setMembershipType(event.target.value as 'manual' | 'dynamic')
                  }
                  className="h-9 rounded-md border border-slate-300 bg-white px-2 text-sm"
                >
                  <option value="manual">{t.manualMembership}</option>
                  <option value="dynamic">{t.dynamicMembership}</option>
                </select>
              </div>
              {membershipType === 'dynamic' ? (
                <div className="grid gap-1.5">
                  <Label htmlFor="group-dynamic-rule">{t.dynamicRuleExpression}</Label>
                  <textarea
                    id="group-dynamic-rule"
                    name="dynamic-rule"
                    required
                    className="min-h-28 rounded-md border border-slate-300 bg-white p-3 font-mono text-sm"
                    placeholder={'user.department == "Engineering"'}
                  />
                  <p className="text-xs text-slate-500">{t.dynamicRuleHelp}</p>
                </div>
              ) : null}

              <div className="grid gap-1.5">
                <Label htmlFor="group-description">{t.descriptionOptionalLabel}</Label>
                <Input
                  id="group-description"
                  name="description"
                  placeholder={t.groupDescriptionPlaceholder}
                />
              </div>

              <div className="grid gap-1.5">
                <Label htmlFor="group-roles">{t.rolesLabel}</Label>
                <Input id="group-roles" name="roles" placeholder="catalog:read, invoice:read" />
                <p className="text-xs text-slate-500">{t.rolesHelp}</p>
              </div>
            </div>

            <div className="flex justify-end gap-2 border-t border-slate-200 bg-slate-50 px-6 py-4">
              <a
                href={listPath}
                className="inline-flex h-9 items-center justify-center rounded-lg border border-slate-200 bg-white px-4 text-sm font-medium text-slate-700 shadow-sm transition hover:bg-slate-50 hover:text-slate-900"
              >
                {t.cancel}
              </a>
              <Button type="submit" disabled={busy}>
                {t.create}
              </Button>
            </div>
          </form>
        </Card>
      </div>
    </AdminShell>
  )
}

export function AdminGroupEditPage({
  csrfToken,
  actorUsername,
  group,
}: {
  csrfToken: string
  actorUsername?: string
  group: AdminGroup
}) {
  const detailPath = tenantURL(`/admin/groups/${encodeURIComponent(group.id)}`)
  const [name, setName] = useState(group.name)
  const [description, setDescription] = useState(group.description ?? '')
  const [roles, setRoles] = useState(group.roles.join(', '))
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [rule, setRule] = useState(group.dynamic_rule)
  const [ruleExpression, setRuleExpression] = useState(group.dynamic_rule?.expression ?? '')
  const [allUsers, setAllUsers] = useState<AdminUser[]>([])
  const [previewUserIDs, setPreviewUserIDs] = useState<string[]>([])
  const [preview, setPreview] = useState<DynamicGroupPreview[]>([])
  const [ruleBusy, setRuleBusy] = useState(false)
  const [ruleError, setRuleError] = useState('')
  const t = useDictionary(adminGroupsDictionary)

  useEffect(() => {
    if (group.membership_type !== 'dynamic') return
    let cancelled = false
    void listAdminUsers().then((users) => {
      if (!cancelled) setAllUsers(users)
    })
    return () => {
      cancelled = true
    }
  }, [group.membership_type])

  async function withRule(action: () => Promise<void>) {
    setRuleBusy(true)
    setRuleError('')
    try {
      await action()
    } catch (cause) {
      setRuleError(cause instanceof AuthenticationAPIError ? cause.message : t.genericActionError)
    } finally {
      setRuleBusy(false)
    }
  }

  const trimmedName = name.trim()
  const nextRoles = parseRoles(roles)
  const nameInvalid = trimmedName === ''
  const changed =
    trimmedName !== group.name ||
    description.trim() !== (group.description ?? '') ||
    nextRoles.join(',') !== group.roles.join(',')

  async function handleSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    if (nameInvalid || !changed) return
    setSaving(true)
    setError('')
    try {
      await updateAdminGroup(csrfToken, group.id, {
        name: trimmedName !== group.name ? trimmedName : undefined,
        description:
          description.trim() !== (group.description ?? '') ? description.trim() : undefined,
        roles: nextRoles.join(',') !== group.roles.join(',') ? nextRoles : undefined,
      })
      window.location.assign(detailPath)
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.groupUpdateFailedError)
      setSaving(false)
    }
  }

  return (
    <AdminShell
      active="groups"
      actorUsername={actorUsername}
      title={t.editGroup}
      description={t.editGroupDescription}
    >
      <div className="flex items-center gap-3">
        <a
          href={detailPath}
          className="inline-flex size-9 items-center justify-center rounded-lg border border-slate-200 bg-white text-slate-700 transition hover:bg-slate-50 hover:text-slate-900"
          aria-label={t.backToDetailAria}
        >
          <IconArrowLeft size={18} aria-hidden="true" />
        </a>
        <h1 className="text-2xl font-bold tracking-tight text-slate-900">{t.editGroup}</h1>
      </div>

      <div className="mt-6 max-w-2xl">
        <Card className="shadow-[0_1px_2px_rgb(15_23_42/4%)]">
          <form onSubmit={handleSubmit}>
            <div className="grid gap-6 p-6">
              {error && <Alert variant="destructive">{error}</Alert>}

              {group.scim_source && (
                <Alert className="mb-2">
                  <div className="flex gap-3">
                    <IconAlertTriangle className="mt-0.5 shrink-0 text-blue-700" size={19} />
                    <div>
                      <p className="text-sm font-semibold text-blue-950">{t.scimSyncGroupTitle}</p>
                      <p className="mt-1 text-xs leading-5 text-blue-800">
                        {t.scimSyncGroupDescription.replace('{source}', group.scim_source)}
                      </p>
                    </div>
                  </div>
                </Alert>
              )}

              <section className="grid gap-4">
                <h3 className="text-xs font-bold uppercase tracking-normal text-slate-400">
                  {t.basicInfoHeading}
                </h3>
                <div className="grid gap-1.5">
                  <Label htmlFor="group-editor-name">{t.groupNameLabel}</Label>
                  <Input
                    id="group-editor-name"
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    required
                    aria-invalid={nameInvalid}
                    readOnly={!!group.scim_source}
                    className={group.scim_source ? 'bg-slate-50' : undefined}
                  />
                </div>
                <div className="grid gap-1.5">
                  <Label htmlFor="group-editor-description">{t.descriptionLabel}</Label>
                  <Input
                    id="group-editor-description"
                    value={description}
                    onChange={(e) => setDescription(e.target.value)}
                    readOnly={!!group.scim_source}
                    className={group.scim_source ? 'bg-slate-50' : undefined}
                  />
                </div>
              </section>
              <section className="grid gap-3 border-t border-slate-200 pt-5">
                <h3 className="text-xs font-bold uppercase tracking-normal text-slate-400">
                  {t.rolesLabel}
                </h3>
                <div className="grid gap-1.5">
                  <Label htmlFor="group-editor-roles">{t.rolesLabel}</Label>
                  <Input
                    id="group-editor-roles"
                    value={roles}
                    onChange={(e) => setRoles(e.target.value)}
                    placeholder="catalog:read, invoice:read"
                  />
                  <p className="text-xs text-slate-500">{t.rolesHelp}</p>
                </div>
              </section>
            </div>

            <div className="flex justify-end gap-2 border-t border-slate-200 bg-slate-50 px-6 py-4">
              <a
                href={detailPath}
                className="inline-flex h-9 items-center justify-center rounded-lg border border-slate-200 bg-white px-4 text-sm font-medium text-slate-700 shadow-sm transition hover:bg-slate-50 hover:text-slate-900"
              >
                {t.cancel}
              </a>
              <Button type="submit" disabled={saving || nameInvalid || !changed}>
                {saving ? t.saving : t.save}
              </Button>
            </div>
          </form>
        </Card>
      </div>

      {group.membership_type === 'dynamic' ? (
        <div className="mt-6 max-w-2xl">
          <Card className="shadow-[0_1px_2px_rgb(15_23_42/4%)]">
            <div className="grid gap-4 p-6">
              {ruleError && <Alert variant="destructive">{ruleError}</Alert>}
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
              <p className="text-xs text-slate-500">{t.dynamicRuleHelp}</p>
              <textarea
                value={ruleExpression}
                onChange={(event) => setRuleExpression(event.target.value)}
                aria-label={t.dynamicRuleExpression}
                className="min-h-28 w-full rounded-md border border-slate-300 bg-white p-3 font-mono text-sm"
                placeholder={'user.department == "Engineering"'}
              />
              <div className="flex flex-wrap gap-2">
                <Button
                  type="button"
                  disabled={ruleBusy || !ruleExpression.trim()}
                  onClick={() =>
                    void withRule(async () => {
                      const saved = await updateDynamicGroupRule(
                        csrfToken,
                        group.id,
                        ruleExpression,
                      )
                      setRule(saved)
                    })
                  }
                >
                  {t.saveRule}
                </Button>
                <Button
                  type="button"
                  variant="outline"
                  disabled={ruleBusy || !ruleExpression.trim() || previewUserIDs.length === 0}
                  onClick={() =>
                    void withRule(async () => {
                      const result = await previewDynamicGroupRule(
                        csrfToken,
                        group.id,
                        ruleExpression,
                        previewUserIDs,
                      )
                      setPreview(result.results)
                    })
                  }
                >
                  {t.previewRule}
                </Button>
                <Button
                  type="button"
                  variant="outline"
                  disabled={ruleBusy || !rule}
                  onClick={() =>
                    void withRule(async () => {
                      const saved = await setDynamicGroupRuleEnabled(
                        csrfToken,
                        group.id,
                        !rule?.enabled,
                      )
                      setRule(saved)
                    })
                  }
                >
                  {rule?.enabled ? t.disableRule : t.enableRule}
                </Button>
              </div>
              <Label htmlFor="group-editor-dynamic-preview">{t.previewUsers}</Label>
              <select
                id="group-editor-dynamic-preview"
                multiple
                value={previewUserIDs}
                onChange={(event) =>
                  setPreviewUserIDs(
                    Array.from(event.currentTarget.selectedOptions, (option) => option.value),
                  )
                }
                className="min-h-24 w-full rounded-md border border-slate-300 bg-white p-2 text-sm"
              >
                {allUsers.map((user) => (
                  <option key={user.id} value={user.id}>
                    {user.preferred_username}
                  </option>
                ))}
              </select>
              {preview.length > 0 ? (
                <ul className="grid gap-1 text-sm">
                  {preview.map((item) => (
                    <li key={item.user_id} className="rounded bg-slate-50 px-2 py-1 font-mono">
                      {item.user_id}: {item.matched ? t.matches : t.doesNotMatch} ({item.change})
                    </li>
                  ))}
                </ul>
              ) : null}
            </div>
          </Card>
        </div>
      ) : null}
    </AdminShell>
  )
}

function parseRoles(value: string) {
  return [
    ...new Set(
      value
        .split(',')
        .map((role) => role.trim())
        .filter(Boolean),
    ),
  ]
}

function optionalValue(value: FormDataEntryValue | null) {
  const normalized = String(value ?? '').trim()
  return normalized || undefined
}
