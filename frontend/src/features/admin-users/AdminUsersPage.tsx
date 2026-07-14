import {
  IconAdjustments,
  IconAlertTriangle,
  IconArrowLeft,
  IconBan,
  IconCheck,
  IconCircleCheck,
  IconClock,
  IconDotsVertical,
  IconDownload,
  IconKey,
  IconMail,
  IconPencil,
  IconRefresh,
  IconSearch,
  IconShield,
  IconShieldCheck,
  IconTrash,
  IconUpload,
  IconUser,
  IconUserPlus,
  IconUsers,
  IconUsersGroup,
  IconX,
} from '@tabler/icons-react'
import { type ChangeEvent, type FormEvent, useCallback, useEffect, useMemo, useState } from 'react'
import {
  addAdminGroupMember,
  AuthenticationAPIError,
  clearAdminUserRequiredAction,
  createAdminUser,
  deleteAdminUser,
  getAdminUser,
  getAdminUserGroups,
  getAdminUserImport,
  importAdminUsers,
  issueMfaEnrollmentBypass,
  listAdminGroups,
  listAdminUsers,
  restoreAdminUser,
  revokeMfaEnrollmentBypass,
  setAdminUserDisabled,
  setAdminUserRequiredAction,
  tenantURL,
  type UpdateAdminUserInput,
  updateAdminUser,
} from '../../api'
import { AdminPaneActions } from '../../components/AdminPaneActions'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Toast } from '../../components/ui/toast'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '../../components/ui/dropdown-menu'
import { attributeGroupKey, attributeGroupTitle, attributeLabel, cn } from '../../lib/utils'
import { useDictionary, useLocale } from '../../lib/i18n'
import {
  domainLabelsDictionary,
  type DomainLabelsDictionary,
} from '../../lib/i18n/domainLabels.i18n'
import { adminUsersDictionary } from './AdminUsersPage.i18n'
import {
  type AdminGroup,
  type AdminUser,
  type AdminUserGroups,
  type AttributeValue,
  REQUIRED_ACTIONS,
  requiredActionLabel,
  type TenantUserAttributeSchema,
  type UserAttributeDef,
  type UserImportResult,
  type UserImportRowError,
} from '../../types'
import {
  daysUntil,
  DetailRow,
  Field,
  formatDateTime,
  Metric,
  optionalValue,
  parseRoles,
  RoleList,
  StatusBadge,
  UserAvatar,
  userLifecycleStatus,
} from './AdminUsersPrimitives'

type StatusFilter = 'all' | 'active' | 'disabled' | 'pending_deletion'

export function AdminUsersPage({
  csrfToken,
  actorUsername,
  users: initialUsers,
}: {
  csrfToken: string
  actorUsername?: string
  users: AdminUser[]
}) {
  const [users, setUsers] = useState(initialUsers)
  const [selectedUserId, setSelectedUserId] = useState(initialUsers[0]?.id ?? '')
  const [query, setQuery] = useState(
    () => new URLSearchParams(window.location.search).get('role') ?? '',
  )
  const [status, setStatus] = useState<StatusFilter>('all')
  const [showDelete, setShowDelete] = useState(false)
  const [showPurge, setShowPurge] = useState(false)
  const [showDisable, setShowDisable] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const t = useDictionary(adminUsersDictionary)
  const { locale } = useLocale()

  const selected = users.find((user) => user.id === selectedUserId)
  const activeCount = users.filter((user) => userLifecycleStatus(user) === 'active').length
  const adminCount = users.filter((user) => user.roles.includes('admin')).length
  const mfaCount = users.filter((user) => user.mfa_enrolled).length
  const filteredUsers = useMemo(() => {
    const needle = query.trim().toLowerCase()
    return users.filter((user) => {
      const matchesStatus = status === 'all' || userLifecycleStatus(user) === status
      const matchesQuery =
        !needle ||
        [user.preferred_username, user.name, user.email, user.id, ...user.roles]
          .filter(Boolean)
          .some((value) => value?.toLowerCase().includes(needle))
      return matchesStatus && matchesQuery
    })
  }, [query, status, users])

  async function refresh(preferredUserId = selectedUserId) {
    const next = await listAdminUsers()
    setUsers(next)
    const nextSelected = next.find((user) => user.id === preferredUserId) ?? next[0]
    setSelectedUserId(nextSelected?.id ?? '')
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

  async function handleDisabled(user: AdminUser) {
    const disabled = !user.disabled_at
    await run(
      async () => {
        await setAdminUserDisabled(csrfToken, user.id, disabled)
        setShowDisable(false)
        await refresh(user.id)
      },
      disabled ? t.userDisabledNotice : t.userEnabledNotice,
    )
  }

  // 無効化は破壊的なので確認ダイアログを挟む。再有効化はアクセス回復のみで
  // 誤操作リスクが低いため即時実行する (片側非対称)。
  function requestDisable(user: AdminUser) {
    if (user.disabled_at) {
      void handleDisabled(user)
    } else {
      setShowDisable(true)
    }
  }

  async function handleRequiredAction(user: AdminUser, action: string, present: boolean) {
    await run(
      async () => {
        if (present) {
          await clearAdminUserRequiredAction(csrfToken, user.id, action)
        } else {
          await setAdminUserRequiredAction(csrfToken, user.id, action)
        }
        await refresh(user.id)
      },
      present ? t.requiredActionClearedNotice : t.requiredActionSetNotice,
    )
  }

  async function handleDelete(user: AdminUser) {
    await run(async () => {
      await deleteAdminUser(csrfToken, user.id)
      setShowDelete(false)
      await refresh(user.id)
    }, t.userDeleteScheduledNotice)
  }

  async function handlePurge(user: AdminUser) {
    await run(async () => {
      await deleteAdminUser(csrfToken, user.id, { purge: true })
      setShowPurge(false)
      await refresh()
    }, t.userPurgedNotice)
  }

  async function handleRestore(user: AdminUser) {
    await run(async () => {
      await restoreAdminUser(csrfToken, user.id)
      await refresh(user.id)
    }, t.userRestoredNotice)
  }

  function selectUser(user: AdminUser) {
    setSelectedUserId(user.id)
  }

  return (
    <>
      <AdminShell
        active="users"
        actorUsername={actorUsername}
        title={t.pageTitle}
        description={t.pageDescription}
        actions={
          <div className="flex items-center gap-2">
            <Button asChild variant="outline">
              <a href={tenantURL('/admin/users/import')}>
                <IconUpload size={17} aria-hidden="true" />
                {t.importUsers}
              </a>
            </Button>
            <Button asChild>
              <a href={tenantURL('/admin/users/new')}>
                <IconUserPlus size={17} aria-hidden="true" />
                {t.addUser}
              </a>
            </Button>
          </div>
        }
      >
        <section
          className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4"
          aria-label={t.overviewSectionLabel}
        >
          <Metric label={t.totalUsers} value={users.length} icon={IconUsers} tone="blue" />
          <Metric
            label={t.activeAccounts}
            value={activeCount}
            icon={IconCircleCheck}
            tone="green"
          />
          <Metric label={t.admins} value={adminCount} icon={IconShield} tone="violet" />
          <Metric label={t.mfaEnrolled} value={mfaCount} icon={IconKey} tone="amber" />
        </section>

        {error && <Alert>{error}</Alert>}
        <Toast message={notice} onDismiss={() => setNotice('')} />

        <Card className="overflow-hidden shadow-[0_1px_2px_rgb(15_23_42/4%)]">
          <div className="flex flex-col gap-3 border-b border-slate-200 p-4 lg:flex-row lg:items-center lg:justify-between">
            <div className="relative w-full max-w-xl">
              <IconSearch
                size={18}
                className="pointer-events-none absolute left-3.5 top-1/2 -translate-y-1/2 text-slate-400"
                aria-hidden="true"
              />
              <Input
                value={query}
                onChange={(event) => setQuery(event.target.value)}
                className="h-10 pl-10"
                placeholder={t.searchPlaceholder}
                aria-label={t.searchAriaLabel}
              />
            </div>
            <div className="flex items-center gap-2">
              <IconAdjustments size={17} className="text-slate-400" aria-hidden="true" />
              <div className="flex rounded-lg border border-slate-200 bg-slate-50 p-0.5">
                {(['all', 'active', 'disabled', 'pending_deletion'] as const).map((value) => (
                  <button
                    key={value}
                    type="button"
                    onClick={() => setStatus(value)}
                    className={cn(
                      'rounded-md px-3 py-1.5 text-xs font-semibold transition-colors',
                      status === value
                        ? 'bg-white text-slate-900 shadow-sm'
                        : 'text-slate-500 hover:text-slate-800',
                    )}
                  >
                    {
                      {
                        all: t.filterAll,
                        active: t.statusActive,
                        disabled: t.statusDisabled,
                        pending_deletion: t.statusPendingDeletion,
                      }[value]
                    }
                  </button>
                ))}
              </div>
              <Button
                variant="outline"
                className="size-9 px-0"
                disabled={busy}
                aria-label={t.reloadAriaLabel}
                onClick={() => void run(() => refresh(), t.listRefreshedNotice)}
              >
                <IconRefresh size={16} aria-hidden="true" />
              </Button>
            </div>
          </div>

          <div className="grid min-h-[520px] xl:grid-cols-[minmax(0,1.55fr)_400px]">
            <div className="min-w-0 overflow-x-auto">
              <table className="w-full min-w-[760px] text-left text-sm">
                <thead className="border-b border-slate-200 bg-slate-50/80 text-[0.68rem] font-bold uppercase tracking-[0.08em] text-slate-500">
                  <tr>
                    <th className="px-5 py-3.5">{t.tableHeaderUser}</th>
                    <th className="px-5 py-3.5">{t.tableHeaderAccess}</th>
                    <th className="px-5 py-3.5">{t.tableHeaderSecurity}</th>
                    <th className="px-5 py-3.5">{t.tableHeaderStatus}</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-100">
                  {filteredUsers.map((user) => (
                    <tr
                      key={user.id}
                      onClick={() => selectUser(user)}
                      className={cn(
                        'cursor-pointer bg-white transition-colors hover:bg-slate-50',
                        selectedUserId === user.id && 'bg-blue-50/60 hover:bg-blue-50/80',
                      )}
                    >
                      <td className="px-5 py-4">
                        <div className="flex items-center gap-3">
                          <UserAvatar user={user} />
                          <div className="min-w-0">
                            <p className="truncate font-semibold text-slate-900">
                              {user.name || user.preferred_username}
                            </p>
                            <p className="truncate text-xs text-slate-500">
                              {user.email || `@${user.preferred_username}`}
                            </p>
                          </div>
                        </div>
                      </td>
                      <td className="px-5 py-4">
                        <RoleList roles={user.roles} />
                      </td>
                      <td className="px-5 py-4">
                        <div className="flex items-center gap-2 text-xs text-slate-600">
                          <span
                            className={cn(
                              'flex size-6 items-center justify-center rounded-full',
                              user.mfa_enrolled
                                ? 'bg-emerald-50 text-emerald-700'
                                : 'bg-slate-100 text-slate-400',
                            )}
                          >
                            <IconKey size={13} aria-hidden="true" />
                          </span>
                          {user.mfa_enrolled ? 'MFA' : t.passwordBadge}
                        </div>
                      </td>
                      <td className="px-5 py-4">
                        <StatusBadge status={userLifecycleStatus(user)} />
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
              {filteredUsers.length === 0 && (
                <div className="flex min-h-64 flex-col items-center justify-center px-6 text-center">
                  <span className="flex size-12 items-center justify-center rounded-full bg-slate-100 text-slate-400">
                    <IconSearch size={22} aria-hidden="true" />
                  </span>
                  <p className="mt-4 font-semibold text-slate-800">{t.emptyStateTitle}</p>
                  <p className="mt-1 text-sm text-slate-500">{t.emptyStateDescription}</p>
                </div>
              )}
            </div>

            <aside className="border-t border-slate-200 bg-slate-50/40 xl:border-l xl:border-t-0">
              {selected ? (
                <UserDetails
                  user={selected}
                  csrfToken={csrfToken}
                  busy={busy}
                  editHref={tenantURL(`/admin/users/${encodeURIComponent(selected.id)}/edit`)}
                  onDisabled={() => requestDisable(selected)}
                  onDelete={() => setShowDelete(true)}
                  onRestore={() => void handleRestore(selected)}
                  onPurge={() => setShowPurge(true)}
                  onRequiredAction={(action, present) =>
                    void handleRequiredAction(selected, action, present)
                  }
                />
              ) : (
                <div className="flex h-full min-h-80 items-center justify-center p-8 text-center text-sm text-slate-500">
                  {t.selectUserPrompt}
                </div>
              )}
            </aside>
          </div>
          <div className="flex items-center justify-between border-t border-slate-200 bg-slate-50/70 px-5 py-3 text-xs text-slate-500">
            <span>{t.countDisplayed.replace('{count}', String(filteredUsers.length))}</span>
            <span>
              {t.lastUpdated.replace('{date}', formatDateTime(new Date().toISOString(), locale))}
            </span>
          </div>
        </Card>
      </AdminShell>

      {showDelete && selected && (
        <DeleteUserDialog
          user={selected}
          busy={busy}
          mode="soft"
          onClose={() => setShowDelete(false)}
          onConfirm={() => void handleDelete(selected)}
        />
      )}
      {showPurge && selected && (
        <DeleteUserDialog
          user={selected}
          busy={busy}
          mode="purge"
          onClose={() => setShowPurge(false)}
          onConfirm={() => void handlePurge(selected)}
        />
      )}
      {showDisable && selected && (
        <DisableUserDialog
          user={selected}
          busy={busy}
          onClose={() => setShowDisable(false)}
          onConfirm={() => void handleDisabled(selected)}
        />
      )}
    </>
  )
}

// AdminUserDetailPage はユーザーの全情報を扱う専用詳細画面 (wi-39)。右ペインの
// 簡易ビューに収まらない網羅情報 (全プロフィール / 全属性 / ライフサイクル /
// 強制アクション / ロールとグループ) をここに集約し、縦スクロール前提で見せる。
export function AdminUserDetailPage({
  csrfToken,
  actorUsername,
  user: initialUser,
  schema,
}: {
  csrfToken: string
  actorUsername?: string
  user: AdminUser
  schema: TenantUserAttributeSchema
}) {
  const [user, setUser] = useState(initialUser)
  const [showDelete, setShowDelete] = useState(false)
  const [showPurge, setShowPurge] = useState(false)
  const [showDisable, setShowDisable] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')

  const attributeDefs = [...schema.builtin, ...schema.attributes]
  const tLabels = useDictionary(domainLabelsDictionary)
  const t = useDictionary(adminUsersDictionary)
  const { locale } = useLocale()

  async function reload() {
    setUser(await getAdminUser(user.id))
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

  async function handleDisabled() {
    const disabled = !user.disabled_at
    await run(
      async () => {
        await setAdminUserDisabled(csrfToken, user.id, disabled)
        setShowDisable(false)
        await reload()
      },
      disabled ? t.userDisabledNotice : t.userEnabledNotice,
    )
  }

  // 無効化は確認ダイアログを挟み、再有効化は即時実行する (片側非対称)。
  function requestDisable() {
    if (user.disabled_at) {
      void handleDisabled()
    } else {
      setShowDisable(true)
    }
  }

  async function handleDelete() {
    await run(async () => {
      await deleteAdminUser(csrfToken, user.id)
      setShowDelete(false)
      await reload()
    }, t.userDeleteScheduledNotice)
  }

  async function handlePurge() {
    await run(async () => {
      await deleteAdminUser(csrfToken, user.id, { purge: true })
      window.location.assign(tenantURL('/admin/users'))
    }, t.userPurgedNotice)
  }

  async function handleRestore() {
    await run(async () => {
      await restoreAdminUser(csrfToken, user.id)
      await reload()
    }, t.userRestoredNotice)
  }

  async function handleIssueMfaEnrollmentBypass() {
    await run(async () => {
      await issueMfaEnrollmentBypass(csrfToken, user.id)
    }, t.mfaEnrollmentBypassIssuedNotice)
  }

  async function handleRevokeMfaEnrollmentBypass() {
    await run(async () => {
      await revokeMfaEnrollmentBypass(csrfToken, user.id)
    }, t.mfaEnrollmentBypassRevokedNotice)
  }

  async function handleRequiredAction(action: string, present: boolean) {
    await run(
      async () => {
        if (present) {
          await clearAdminUserRequiredAction(csrfToken, user.id, action)
        } else {
          await setAdminUserRequiredAction(csrfToken, user.id, action)
        }
        await reload()
      },
      present ? t.requiredActionClearedNotice : t.requiredActionSetNotice,
    )
  }

  return (
    <>
      <AdminShell
        active="users"
        actorUsername={actorUsername}
        title={user.name || user.preferred_username}
        description={`@${user.preferred_username}`}
        actions={
          <div className="flex items-center gap-2">
            <a
              href={tenantURL('/admin/users')}
              className="inline-flex items-center gap-1.5 rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm font-semibold text-slate-700 transition-colors hover:bg-slate-50"
            >
              <IconArrowLeft size={16} aria-hidden="true" />
              {t.backToUserList}
            </a>
            <Button asChild>
              <a href={tenantURL(`/admin/users/${encodeURIComponent(user.id)}/edit`)}>
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
                  aria-label={t.userActionsAriaLabel}
                  disabled={busy}
                >
                  <IconDotsVertical size={18} aria-hidden="true" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                {userLifecycleStatus(user) === 'pending_deletion' ? (
                  <>
                    <DropdownMenuItem onSelect={() => void handleRestore()}>
                      <IconRefresh size={17} aria-hidden="true" />
                      {t.restoreAccount}
                    </DropdownMenuItem>
                    <DropdownMenuSeparator className="my-1 h-px bg-slate-200" />
                    <DropdownMenuItem className="text-red-700" onSelect={() => setShowPurge(true)}>
                      <IconTrash size={17} aria-hidden="true" />
                      {t.permanentlyDelete}
                    </DropdownMenuItem>
                  </>
                ) : (
                  <>
                    {!user.mfa_enrolled ? (
                      <>
                        <DropdownMenuItem onSelect={() => void handleIssueMfaEnrollmentBypass()}>
                          <IconShield size={17} aria-hidden="true" />
                          {t.issueMfaEnrollmentBypass}
                        </DropdownMenuItem>
                        <DropdownMenuItem onSelect={() => void handleRevokeMfaEnrollmentBypass()}>
                          <IconX size={17} aria-hidden="true" />
                          {t.revokeMfaEnrollmentBypass}
                        </DropdownMenuItem>
                        <DropdownMenuSeparator className="my-1 h-px bg-slate-200" />
                      </>
                    ) : null}
                    <DropdownMenuItem
                      className={user.disabled_at ? undefined : 'text-red-700'}
                      onSelect={() => requestDisable()}
                    >
                      {user.disabled_at ? (
                        <IconCheck size={17} aria-hidden="true" />
                      ) : (
                        <IconBan size={17} aria-hidden="true" />
                      )}
                      {user.disabled_at ? t.reEnableAccount : t.disableAccount}
                    </DropdownMenuItem>
                    <DropdownMenuSeparator className="my-1 h-px bg-slate-200" />
                    <DropdownMenuItem className="text-red-700" onSelect={() => setShowDelete(true)}>
                      <IconTrash size={17} aria-hidden="true" />
                      {t.deleteAccount}
                    </DropdownMenuItem>
                  </>
                )}
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        }
      >
        {error && <Alert>{error}</Alert>}
        <Toast message={notice} onDismiss={() => setNotice('')} />

        <div className="flex items-center gap-3">
          <UserAvatar user={user} large />
          <div className="min-w-0">
            <div className="flex items-center gap-2">
              <h2 className="truncate text-lg font-semibold text-slate-950">
                {user.name || user.preferred_username}
              </h2>
              <StatusBadge status={userLifecycleStatus(user)} compact />
              {user.scim_source && (
                <span className="rounded-md bg-blue-100 px-2 py-0.5 text-xs font-semibold text-blue-800">
                  {t.scimSyncBadge.replace('{source}', user.scim_source)}
                </span>
              )}
            </div>
            <p className="mt-0.5 text-sm text-slate-500">@{user.preferred_username}</p>
          </div>
        </div>

        <div className="grid gap-5 xl:grid-cols-3">
          <div className="flex flex-col gap-5 xl:col-span-2">
            <Card className="p-5">
              <h3 className="text-xs font-bold uppercase tracking-[0.1em] text-slate-400">
                {t.profileHeading}
              </h3>
              <dl className="mt-3 grid gap-3 text-sm sm:grid-cols-2">
                <DetailRow icon={IconUser} label={t.name} value={user.name || t.notSet} />
                <DetailRow
                  icon={IconUser}
                  label={t.givenName}
                  value={user.given_name || t.notSet}
                />
                <DetailRow
                  icon={IconUser}
                  label={t.familyName}
                  value={user.family_name || t.notSet}
                />
                <DetailRow
                  icon={IconUser}
                  label={t.username}
                  value={user.preferred_username}
                  mono
                />
                <DetailRow icon={IconMail} label={t.email} value={user.email || t.notSet} />
                <DetailRow
                  icon={IconShieldCheck}
                  label={t.emailVerification}
                  value={user.email_verified ? t.verified : t.unverified}
                />
                <DetailRow
                  icon={IconKey}
                  label={t.authMethod}
                  value={user.mfa_enrolled ? `Password + MFA` : t.passwordBadge}
                />
                <DetailRow icon={IconUser} label={t.userId} value={user.id} mono />
              </dl>
              {user.scim_source && (
                <div className="mt-4 border-t border-slate-100 pt-4 flex items-center gap-2 text-xs text-blue-700">
                  <IconAlertTriangle size={15} />
                  <span>{t.scimManagedNotice}</span>
                </div>
              )}
            </Card>

            <Card className="p-5">
              <h3 className="text-xs font-bold uppercase tracking-[0.1em] text-slate-400">
                {t.attributesHeading}
              </h3>
              <p className="mt-1 text-xs text-slate-500">{t.attributesDescription}</p>
              <div className="mt-4 flex flex-col gap-5">
                {groupedAttributeDefs(attributeDefs, tLabels).map((group) => (
                  <AttributeGroup
                    key={group.key}
                    title={group.title}
                    defs={group.defs}
                    user={user}
                  />
                ))}
              </div>
            </Card>
          </div>

          <div className="flex flex-col gap-5">
            <Card className="p-5">
              <h3 className="text-xs font-bold uppercase tracking-[0.1em] text-slate-400">
                {t.lifecycleHeading}
              </h3>
              <dl className="mt-3 grid gap-3 text-sm">
                <DetailRow
                  icon={IconCircleCheck}
                  label={t.statusLabel}
                  value={
                    {
                      active: t.statusActive,
                      disabled: t.statusDisabled,
                      pending_deletion: t.statusPendingDeletion,
                    }[userLifecycleStatus(user)]
                  }
                />
                {userLifecycleStatus(user) === 'pending_deletion' && user.purge_after && (
                  <DetailRow
                    icon={IconClock}
                    label={t.purgeScheduled}
                    value={`${formatDateTime(user.purge_after, locale)}${
                      daysUntil(user.purge_after) !== null
                        ? ` ${t.daysRemaining.replace('{days}', String(daysUntil(user.purge_after)))}`
                        : ''
                    }`}
                  />
                )}
                <DetailRow
                  icon={IconClock}
                  label={t.createdAt}
                  value={formatDateTime(user.created_at, locale)}
                />
                <DetailRow
                  icon={IconClock}
                  label={t.updatedAt}
                  value={formatDateTime(user.updated_at, locale)}
                />
                <DetailRow
                  icon={IconClock}
                  label={t.lastLogin}
                  value={
                    user.last_login_at
                      ? formatDateTime(user.last_login_at, locale)
                      : t.neverLoggedIn
                  }
                />
                <DetailRow
                  icon={IconKey}
                  label={t.passwordChanged}
                  value={
                    user.password_changed_at
                      ? formatDateTime(user.password_changed_at, locale)
                      : t.noRecord
                  }
                />
              </dl>
              <div className="mt-5 border-t border-slate-200 pt-5">
                <UserRequiredActionsSection
                  user={user}
                  busy={busy}
                  onToggle={handleRequiredAction}
                />
              </div>
            </Card>

            <Card className="p-5">
              <UserGroupsSection user={user} csrfToken={csrfToken} />
            </Card>
          </div>
        </div>
      </AdminShell>

      {showDelete && (
        <DeleteUserDialog
          user={user}
          busy={busy}
          mode="soft"
          onClose={() => setShowDelete(false)}
          onConfirm={() => void handleDelete()}
        />
      )}
      {showPurge && (
        <DeleteUserDialog
          user={user}
          busy={busy}
          mode="purge"
          onClose={() => setShowPurge(false)}
          onConfirm={() => void handlePurge()}
        />
      )}
      {showDisable && (
        <DisableUserDialog
          user={user}
          busy={busy}
          onClose={() => setShowDisable(false)}
          onConfirm={() => void handleDisabled()}
        />
      )}
    </>
  )
}

// AttributeGroup は区分内で「値が設定されている」属性だけを読み取り表示する。
// 全 def を出すと未設定行で埋もれるため、設定済みのみを示し、無ければその旨を出す。
function AttributeGroup({
  title,
  defs,
  user,
}: {
  title: string
  defs: UserAttributeDef[]
  user: AdminUser
}) {
  const t = useDictionary(adminUsersDictionary)
  const rows = defs
    .map((def) => ({ def, value: user.attributes?.[def.key] }))
    .filter((row): row is { def: UserAttributeDef; value: AttributeValue } => Boolean(row.value))
  return (
    <div>
      <h4 className="text-[0.68rem] font-bold uppercase tracking-wide text-slate-400">{title}</h4>
      {rows.length === 0 ? (
        <p className="mt-2 text-xs text-slate-400">{t.noSetItems}</p>
      ) : (
        <dl className="mt-2 grid gap-2 text-sm sm:grid-cols-2">
          {rows.map(({ def, value }) => (
            <div key={def.key} className="grid gap-0.5">
              <dt className="text-xs text-slate-500">{attributeLabel(def)}</dt>
              <dd className="min-w-0 break-all text-slate-800">{attributeValueToText(value)}</dd>
            </div>
          ))}
        </dl>
      )}
    </div>
  )
}

// UserDetails は右ペインの詳細ビュー。同一画面で複数ユーザーを見比べられるよう
// 情報量は厚めに残しつつ、上部に「詳細 / 編集」を置いて専用詳細ページや
// 編集モーダルへすぐ飛べるようにする (wi-39)。
function UserDetails({
  user,
  csrfToken,
  busy,
  editHref,
  onDisabled,
  onDelete,
  onRestore,
  onPurge,
  onRequiredAction,
}: {
  user: AdminUser
  csrfToken: string
  busy: boolean
  editHref: string
  onDisabled: () => void
  onDelete: () => void
  onRestore: () => void
  onPurge: () => void
  onRequiredAction: (action: string, present: boolean) => void
}) {
  const pending = userLifecycleStatus(user) === 'pending_deletion'
  const t = useDictionary(adminUsersDictionary)
  const { locale } = useLocale()
  return (
    <div className="flex h-full flex-col">
      <div className="border-b border-slate-200 bg-white p-5">
        <div className="flex items-start gap-3">
          <UserAvatar user={user} large />
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-2">
              <h2 className="truncate text-lg font-semibold text-slate-950">
                {user.name || user.preferred_username}
              </h2>
              <StatusBadge status={userLifecycleStatus(user)} compact />
            </div>
            <p className="mt-0.5 text-sm text-slate-500">@{user.preferred_username}</p>
          </div>
        </div>

        {pending && <PendingDeletionNotice user={user} />}

        <div className="mt-4">
          <AdminPaneActions
            detailHref={tenantURL(`/admin/users/${encodeURIComponent(user.id)}`)}
            busy={busy}
            editHref={editHref}
            actions={
              pending
                ? [
                    { label: t.restoreAccount, icon: IconRefresh, onClick: onRestore },
                    {
                      label: t.permanentlyDelete,
                      icon: IconTrash,
                      onClick: onPurge,
                      tone: 'danger',
                    },
                  ]
                : [
                    {
                      label: user.disabled_at ? t.reEnableAccount : t.disableAccount,
                      icon: user.disabled_at ? IconCheck : IconBan,
                      onClick: onDisabled,
                      tone: user.disabled_at ? 'default' : 'danger',
                    },
                    {
                      label: t.deleteAccount,
                      icon: IconTrash,
                      onClick: onDelete,
                      tone: 'danger',
                    },
                  ]
            }
          />
        </div>
      </div>

      <div className="flex flex-1 flex-col gap-6 p-5">
        <section>
          <h3 className="text-xs font-bold uppercase tracking-[0.1em] text-slate-400">
            {t.profileHeading}
          </h3>
          <dl className="mt-3 grid gap-3 text-sm">
            <DetailRow icon={IconMail} label={t.email} value={user.email || t.notSet} />
            <DetailRow
              icon={IconShieldCheck}
              label={t.emailVerification}
              value={user.email_verified ? t.verified : t.unverified}
            />
            <DetailRow
              icon={IconKey}
              label={t.authMethod}
              value={user.mfa_enrolled ? `Password + MFA` : t.passwordBadge}
            />
            <DetailRow
              icon={IconClock}
              label={t.createdAt}
              value={formatDateTime(user.created_at, locale)}
            />
            <DetailRow
              icon={IconClock}
              label={t.lastLogin}
              value={
                user.last_login_at ? formatDateTime(user.last_login_at, locale) : t.neverLoggedIn
              }
            />
            <DetailRow
              icon={IconKey}
              label={t.passwordChanged}
              value={
                user.password_changed_at
                  ? formatDateTime(user.password_changed_at, locale)
                  : t.noRecord
              }
            />
            <DetailRow icon={IconUser} label={t.subjectId} value={user.id} mono />
          </dl>
        </section>

        <UserRequiredActionsSection user={user} busy={busy} onToggle={onRequiredAction} />

        <UserGroupsSection user={user} csrfToken={csrfToken} />
      </div>
    </div>
  )
}

// PendingDeletionNotice は削除予約中のユーザーに、猶予残日数と自動完全削除の
// 予定を伝える amber バナー。復元動線 (メニューの「アカウントを復元」) を促す。
function PendingDeletionNotice({ user }: { user: AdminUser }) {
  const remaining = daysUntil(user.purge_after)
  const t = useDictionary(adminUsersDictionary)
  const { locale } = useLocale()
  return (
    <div className="mt-4 rounded-xl border border-amber-200 bg-amber-50 p-3 text-xs leading-5 text-amber-900">
      <p className="font-semibold">{t.pendingDeletionTitle}</p>
      <p className="mt-1">
        {remaining !== null
          ? t.pendingDeletionRemaining.replace('{days}', String(remaining))
          : t.pendingDeletionRemainingUnknown}
        {t.pendingDeletionRestoreNotice}
      </p>
      {user.purge_after && (
        <p className="mt-1 text-amber-700">
          {t.purgeScheduledAt.replace('{date}', formatDateTime(user.purge_after, locale))}
        </p>
      )}
    </div>
  )
}

function UserRequiredActionsSection({
  user,
  busy,
  onToggle,
}: {
  user: AdminUser
  busy: boolean
  onToggle: (action: string, present: boolean) => void
}) {
  const active = new Set(user.required_actions ?? [])
  const tLabels = useDictionary(domainLabelsDictionary)
  const t = useDictionary(adminUsersDictionary)
  return (
    <section>
      <h3 className="text-xs font-bold uppercase tracking-[0.1em] text-slate-400">
        {t.requiredActionsHeading}
      </h3>
      <p className="mt-1 text-xs text-slate-500">{t.requiredActionsDescription}</p>
      <div className="mt-3 flex flex-wrap gap-2">
        {REQUIRED_ACTIONS.map((action) => {
          const present = active.has(action)
          return (
            <button
              key={action}
              type="button"
              disabled={busy}
              onClick={() => onToggle(action, present)}
              aria-pressed={present}
              className={cn(
                'rounded-full border px-3 py-1 text-xs font-medium transition-colors disabled:opacity-50',
                present
                  ? 'border-amber-300 bg-amber-50 text-amber-800 hover:bg-amber-100'
                  : 'border-slate-200 bg-white text-slate-500 hover:bg-slate-50',
              )}
            >
              {present ? '✓ ' : '+ '}
              {requiredActionLabel(action, tLabels)}
            </button>
          )
        })}
      </div>
    </section>
  )
}

function UserGroupsSection({ user, csrfToken }: { user: AdminUser; csrfToken: string }) {
  const [data, setData] = useState<AdminUserGroups | null>(null)
  const [allGroups, setAllGroups] = useState<AdminGroup[]>([])
  const [error, setError] = useState('')
  const [adding, setAdding] = useState(false)
  const [selectedGroup, setSelectedGroup] = useState('')
  const { id } = user
  const t = useDictionary(adminUsersDictionary)

  const load = useCallback(async () => {
    try {
      const [groups, all] = await Promise.all([getAdminUserGroups(id), listAdminGroups()])
      setData(groups)
      setAllGroups(all)
      setError('')
    } catch (err) {
      setError(err instanceof AuthenticationAPIError ? err.message : t.groupFetchError)
    }
  }, [id, t.groupFetchError])

  useEffect(() => {
    setData(null)
    setSelectedGroup('')
    void load()
  }, [load])

  async function handleAdd() {
    if (!selectedGroup) return
    setAdding(true)
    try {
      await addAdminGroupMember(csrfToken, selectedGroup, user.id)
      setSelectedGroup('')
      await load()
    } catch (err) {
      setError(err instanceof AuthenticationAPIError ? err.message : t.groupAddError)
    } finally {
      setAdding(false)
    }
  }

  const memberIDs = new Set(data?.groups.map((g) => g.id) ?? [])
  const addable = allGroups.filter((g) => !memberIDs.has(g.id))

  return (
    <section className="border-t border-slate-200 pt-5">
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-sm font-semibold text-slate-900">{t.rolesAndGroupsHeading}</h3>
          <p className="mt-0.5 text-xs text-slate-500">{t.effectiveRolesDescription}</p>
        </div>
        <IconShield size={18} className="text-slate-400" aria-hidden="true" />
      </div>

      {error && (
        <Alert variant="destructive" className="mt-3">
          {error}
        </Alert>
      )}

      <div className="mt-3 space-y-3">
        <RoleGroup label={t.explicitRoles} roles={user.roles} />
        <RoleGroup label={t.groupDerivedRoles} roles={data?.group_roles ?? []} />
        <RoleGroup label={t.effectiveRoles} roles={data?.effective_roles ?? user.roles} emphasis />
      </div>

      <div className="mt-4">
        <div className="flex items-center gap-2">
          <IconUsersGroup size={16} className="text-slate-400" aria-hidden="true" />
          <h4 className="text-xs font-bold uppercase tracking-[0.1em] text-slate-400">
            {t.memberGroupsHeading}
          </h4>
        </div>
        <div className="mt-2 rounded-xl border border-slate-200 bg-white p-3">
          {data && data.groups.length === 0 ? (
            <span className="text-xs text-slate-400">{t.notInAnyGroup}</span>
          ) : (
            <ul className="flex flex-col gap-2">
              {data?.groups.map((group) => (
                <li key={group.id} className="flex items-center justify-between gap-2">
                  <a
                    href={tenantURL(`/admin/groups?group=${encodeURIComponent(group.id)}`)}
                    className="text-sm font-medium text-indigo-700 hover:underline"
                  >
                    {group.name}
                  </a>
                  <RoleList roles={group.roles} />
                </li>
              ))}
            </ul>
          )}
        </div>

        {addable.length > 0 && (
          <div className="mt-2 flex items-center gap-2">
            <select
              value={selectedGroup}
              onChange={(event) => setSelectedGroup(event.target.value)}
              disabled={adding}
              className="h-9 flex-1 rounded-lg border border-slate-200 bg-white px-2 text-sm text-slate-700"
            >
              <option value="">{t.selectGroupPlaceholder}</option>
              {addable.map((group) => (
                <option key={group.id} value={group.id}>
                  {group.name}
                </option>
              ))}
            </select>
            <Button
              type="button"
              disabled={adding || !selectedGroup}
              onClick={() => void handleAdd()}
            >
              <IconUserPlus size={16} aria-hidden="true" />
              {t.add}
            </Button>
          </div>
        )}
      </div>
    </section>
  )
}

function RoleGroup({
  label,
  roles,
  emphasis = false,
}: {
  label: string
  roles: string[]
  emphasis?: boolean
}) {
  return (
    <div
      className={cn(
        'rounded-xl border p-3',
        emphasis ? 'border-indigo-200 bg-indigo-50/50' : 'border-slate-200 bg-white',
      )}
    >
      <p className="mb-2 text-[0.68rem] font-semibold uppercase tracking-wide text-slate-400">
        {label}
      </p>
      <RoleList roles={roles} />
    </div>
  )
}

function RoleDiff({
  title,
  roles,
  tone,
}: {
  title: string
  roles: string[]
  tone: 'add' | 'remove'
}) {
  const t = useDictionary(adminUsersDictionary)
  return (
    <div>
      <p className="text-xs font-semibold text-slate-500">{title}</p>
      <div className="mt-2 flex min-h-16 flex-wrap content-start gap-1.5 rounded-xl border border-slate-200 bg-white p-3">
        {roles.length === 0 ? (
          <span className="text-xs text-slate-400">{t.roleNone}</span>
        ) : (
          roles.map((role) => (
            <span
              key={role}
              className={cn(
                'rounded-md px-2 py-1 text-xs font-semibold',
                tone === 'add' ? 'bg-emerald-50 text-emerald-700' : 'bg-red-50 text-red-700',
              )}
            >
              {tone === 'add' ? '+' : '-'} {role}
            </span>
          ))
        )}
      </div>
    </div>
  )
}

function attributeValueToText(value: AttributeValue): string {
  switch (value.type) {
    case 'string':
      return value.string ?? ''
    case 'date':
      return value.date ?? ''
    case 'number':
      return value.number?.toString() ?? ''
    case 'boolean':
      return value.boolean ? 'true' : 'false'
    case 'string_array':
      return (value.string_array ?? []).join(', ')
    default:
      return ''
  }
}

function textToAttributeValue(def: UserAttributeDef, text: string): AttributeValue | undefined {
  const trimmed = text.trim()
  switch (def.type) {
    case 'boolean':
      return { type: 'boolean', boolean: text === 'true' }
    case 'number':
      return trimmed ? { type: 'number', number: Number(trimmed) } : undefined
    case 'date':
      return trimmed ? { type: 'date', date: trimmed } : undefined
    case 'string_array': {
      const items = trimmed
        .split(',')
        .map((item) => item.trim())
        .filter((item) => item.length > 0)
      return items.length ? { type: 'string_array', string_array: items } : undefined
    }
    default:
      return trimmed ? { type: 'string', string: trimmed } : undefined
  }
}

function attributeDraftFromUser(user: AdminUser, defs: UserAttributeDef[]): Record<string, string> {
  const draft: Record<string, string> = {}
  for (const def of defs) {
    const value = user.attributes?.[def.key]
    draft[def.key] = value ? attributeValueToText(value) : ''
  }
  return draft
}

function attributeMapFromDraft(
  draft: Record<string, string>,
  defs: UserAttributeDef[],
): Record<string, AttributeValue> {
  const map: Record<string, AttributeValue> = {}
  for (const def of defs) {
    const value = textToAttributeValue(def, draft[def.key] ?? '')
    if (value) {
      map[def.key] = value
    }
  }
  return map
}

function AdminAttributeField({
  def,
  value,
  onChange,
  readOnly,
}: {
  def: UserAttributeDef
  value: string
  onChange: (next: string) => void
  readOnly?: boolean
}) {
  const id = `user-editor-attr-${def.key}`
  const label = attributeLabel(def)
  const t = useDictionary(adminUsersDictionary)
  if (def.type === 'boolean') {
    return (
      <label htmlFor={id} className="inline-flex items-center gap-2 text-sm text-slate-700">
        <input
          id={id}
          type="checkbox"
          checked={value === 'true'}
          onChange={(event) => !readOnly && onChange(event.target.checked ? 'true' : 'false')}
          disabled={readOnly}
          className="size-4 rounded border-slate-300 disabled:opacity-50"
        />
        <span className="font-mono">{label}</span>
      </label>
    )
  }
  return (
    <div className="grid gap-1.5">
      <Label htmlFor={id} className="font-mono text-xs">
        {label}
      </Label>
      <Input
        id={id}
        type={def.type === 'number' ? 'number' : def.type === 'date' ? 'date' : 'text'}
        value={value}
        placeholder={def.type === 'string_array' ? t.commaSeparated : undefined}
        onChange={(event) => onChange(event.target.value)}
        readOnly={readOnly}
        className={readOnly ? 'bg-slate-50' : undefined}
      />
    </div>
  )
}

function groupedAttributeDefs(defs: UserAttributeDef[], t: DomainLabelsDictionary) {
  const groups = new Map<ReturnType<typeof attributeGroupKey>, UserAttributeDef[]>()
  for (const def of defs) {
    const key = attributeGroupKey(def)
    groups.set(key, [...(groups.get(key) ?? []), def])
  }
  return (['profile', 'organization', 'custom'] as const)
    .map((key) => ({ key, title: attributeGroupTitle(key, t), defs: groups.get(key) ?? [] }))
    .filter((group) => group.defs.length > 0)
}

function AdminAttributeEditorGroups({
  defs,
  values,
  onChange,
  readOnly,
}: {
  defs: UserAttributeDef[]
  values: Record<string, string>
  onChange: (key: string, next: string) => void
  readOnly?: boolean
}) {
  const tLabels = useDictionary(domainLabelsDictionary)
  const t = useDictionary(adminUsersDictionary)
  const groups = groupedAttributeDefs(defs, tLabels)
  if (groups.length === 0) return null
  return (
    <section className="grid gap-4 border-t border-slate-200 pt-5">
      <div>
        <h3 className="text-xs font-bold uppercase tracking-normal text-slate-400">
          {t.accountInfoHeading}
        </h3>
        <p className="mt-1 text-xs leading-5 text-slate-500">{t.accountInfoDescription}</p>
      </div>
      {groups.map((group) => (
        <fieldset key={group.key} className="grid gap-3 rounded-lg border border-slate-200 p-4">
          <legend className="px-1 text-xs font-bold uppercase tracking-normal text-slate-500">
            {group.title}
          </legend>
          {group.defs.map((def) => (
            <AdminAttributeField
              key={def.key}
              def={def}
              value={values[def.key] ?? ''}
              onChange={(next) => onChange(def.key, next)}
              readOnly={readOnly}
            />
          ))}
        </fieldset>
      ))}
    </section>
  )
}

// AdminUserEditPage はユーザー編集の専用画面 (wi-126 §6)。従来モーダルだった編集
// フォームを詳細→編集ポリシーに沿って独立画面へ移し、保存後は詳細画面へ戻す。
// ロール変更を含む場合は保存前に確認ステップ (confirming) を同一画面で挟む。
export function AdminUserEditPage({
  csrfToken,
  actorUsername,
  user,
  schema,
}: {
  csrfToken: string
  actorUsername?: string
  user: AdminUser
  schema: TenantUserAttributeSchema
}) {
  const attributeDefs = [...schema.builtin, ...schema.attributes]
  const detailPath = tenantURL(`/admin/users/${encodeURIComponent(user.id)}`)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const t = useDictionary(adminUsersDictionary)

  async function persist(input: UpdateAdminUserInput) {
    setBusy(true)
    setError('')
    try {
      await updateAdminUser(csrfToken, user.id, input)
      window.location.assign(detailPath)
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.updateFailed)
      setBusy(false)
    }
  }

  const initialUsername = user.preferred_username
  const initialName = user.name ?? ''
  const initialGivenName = user.given_name ?? ''
  const initialFamilyName = user.family_name ?? ''
  const initialEmail = user.email ?? ''
  const initialEmailVerified = user.email_verified
  const initialAttrDraft = attributeDraftFromUser(user, attributeDefs)

  const [username, setUsername] = useState(initialUsername)
  const [name, setName] = useState(initialName)
  const [givenName, setGivenName] = useState(initialGivenName)
  const [familyName, setFamilyName] = useState(initialFamilyName)
  const [email, setEmail] = useState(initialEmail)
  const [emailVerified, setEmailVerified] = useState(initialEmailVerified)
  const [emailVerifiedTouched, setEmailVerifiedTouched] = useState(false)
  const [roles, setRoles] = useState(user.roles.join(', '))
  const [attrDraft, setAttrDraft] = useState<Record<string, string>>(initialAttrDraft)
  const [confirming, setConfirming] = useState(false)

  const emailChanged = email !== initialEmail
  const effectiveEmailVerified = emailChanged && !emailVerifiedTouched ? false : emailVerified
  const trimmedUsername = username.trim()
  const usernameInvalid = trimmedUsername === ''
  const attributesChanged = attributeDefs.some(
    (def) => (attrDraft[def.key] ?? '') !== (initialAttrDraft[def.key] ?? ''),
  )
  const profileChanged =
    trimmedUsername !== initialUsername ||
    name !== initialName ||
    givenName !== initialGivenName ||
    familyName !== initialFamilyName ||
    email !== initialEmail ||
    effectiveEmailVerified !== initialEmailVerified ||
    attributesChanged
  const nextRoles = parseRoles(roles)
  const addedRoles = nextRoles.filter((role) => !user.roles.includes(role))
  const removedRoles = user.roles.filter((role) => !nextRoles.includes(role))
  const rolesChanged = addedRoles.length > 0 || removedRoles.length > 0
  const changed = profileChanged || rolesChanged

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (usernameInvalid || !changed) return
    if (rolesChanged && !confirming) {
      setConfirming(true)
      return
    }
    const input: UpdateAdminUserInput = {}
    if (trimmedUsername !== initialUsername) input.preferred_username = trimmedUsername
    if (name !== initialName) input.name = name
    if (givenName !== initialGivenName) input.given_name = givenName
    if (familyName !== initialFamilyName) input.family_name = familyName
    if (email !== initialEmail) input.email = email
    if (effectiveEmailVerified !== initialEmailVerified) {
      input.email_verified = effectiveEmailVerified
    }
    if (rolesChanged) input.roles = nextRoles
    // admin は属性バッグ全体を置換するため、ドラフトから完全な map を再構成する。
    if (attributesChanged) input.attributes = attributeMapFromDraft(attrDraft, attributeDefs)
    void persist(input)
  }

  return (
    <AdminShell
      active="users"
      actorUsername={actorUsername}
      title={t.editUserTitle}
      description={`${user.name || user.preferred_username} (@${user.preferred_username})`}
      actions={
        <Button asChild variant="outline">
          <a href={detailPath}>
            <IconArrowLeft size={16} aria-hidden="true" />
            {t.userDetail}
          </a>
        </Button>
      }
    >
      {error && <Alert>{error}</Alert>}
      <Card className="mx-auto w-full max-w-3xl overflow-hidden">
        <div className="border-b border-slate-200 px-6 py-5">
          <p className="text-xs font-bold uppercase tracking-[0.12em] text-blue-700">
            {t.profileAndAccessLabel}
          </p>
          <h2 className="mt-1 text-xl font-semibold">
            {confirming ? t.confirmChangesHeading : t.editUserTitle}
          </h2>
        </div>

        <form onSubmit={handleSubmit} className="flex flex-col">
          <div>
            {confirming ? (
              <div className="p-6">
                <div className="rounded-xl border border-amber-200 bg-amber-50 p-4">
                  <div className="flex gap-3">
                    <IconShield
                      size={19}
                      className="mt-0.5 shrink-0 text-amber-700"
                      aria-hidden="true"
                    />
                    <div>
                      <p className="text-sm font-semibold text-amber-950">
                        {t.roleChangeWarningTitle}
                      </p>
                      <p className="mt-1 text-xs leading-5 text-amber-800">
                        {t.roleChangeWarningDescription}
                      </p>
                    </div>
                  </div>
                </div>
                <div className="mt-5 grid gap-4 sm:grid-cols-2">
                  <RoleDiff title={t.rolesAdded} roles={addedRoles} tone="add" />
                  <RoleDiff title={t.rolesRemoved} roles={removedRoles} tone="remove" />
                </div>
                {profileChanged && (
                  <p className="mt-4 text-xs leading-5 text-slate-500">{t.profileChangeNotice}</p>
                )}
              </div>
            ) : (
              <div className="grid gap-6 p-6">
                {user.scim_source && (
                  <Alert>
                    <div className="flex gap-3">
                      <IconAlertTriangle className="mt-0.5 shrink-0 text-blue-700" size={19} />
                      <div>
                        <p className="text-sm font-semibold text-blue-950">{t.scimSyncUserTitle}</p>
                        <p className="mt-1 text-xs leading-5 text-blue-800">
                          {t.scimSyncUserDescription.replace('{source}', user.scim_source)}
                        </p>
                      </div>
                    </div>
                  </Alert>
                )}

                <section className="grid gap-4">
                  <h3 className="text-xs font-bold uppercase tracking-[0.1em] text-slate-400">
                    {t.profileHeading}
                  </h3>
                  <div className="grid gap-2">
                    <Label htmlFor="user-editor-username">{t.username}</Label>
                    <Input
                      id="user-editor-username"
                      value={username}
                      onChange={(event) => setUsername(event.target.value)}
                      autoFocus={!user.scim_source}
                      required
                      aria-invalid={usernameInvalid}
                      readOnly={!!user.scim_source}
                      className={user.scim_source ? 'bg-slate-50' : undefined}
                    />
                    <p className="text-xs leading-5 text-slate-500">{t.usernameHelp}</p>
                  </div>
                  <div className="grid gap-2">
                    <Label htmlFor="user-editor-name">{t.displayName}</Label>
                    <Input
                      id="user-editor-name"
                      value={name}
                      onChange={(event) => setName(event.target.value)}
                      readOnly={!!user.scim_source}
                      className={user.scim_source ? 'bg-slate-50' : undefined}
                    />
                  </div>
                  <div className="grid gap-4 sm:grid-cols-2">
                    <div className="grid gap-2">
                      <Label htmlFor="user-editor-given-name">{t.givenName} (given_name)</Label>
                      <Input
                        id="user-editor-given-name"
                        value={givenName}
                        onChange={(event) => setGivenName(event.target.value)}
                        readOnly={!!user.scim_source}
                        className={user.scim_source ? 'bg-slate-50' : undefined}
                      />
                    </div>
                    <div className="grid gap-2">
                      <Label htmlFor="user-editor-family-name">{t.familyName} (family_name)</Label>
                      <Input
                        id="user-editor-family-name"
                        value={familyName}
                        onChange={(event) => setFamilyName(event.target.value)}
                        readOnly={!!user.scim_source}
                        className={user.scim_source ? 'bg-slate-50' : undefined}
                      />
                    </div>
                  </div>
                  <div className="grid gap-2">
                    <Label htmlFor="user-editor-email">{t.emailFieldLabel}</Label>
                    <Input
                      id="user-editor-email"
                      type="email"
                      value={email}
                      onChange={(event) => {
                        setEmail(event.target.value)
                        setEmailVerifiedTouched(false)
                      }}
                      readOnly={!!user.scim_source}
                      className={user.scim_source ? 'bg-slate-50' : undefined}
                    />
                    {emailChanged && (
                      <p className="text-xs leading-5 text-amber-700">
                        {t.emailChangedVerificationNotice}
                      </p>
                    )}
                  </div>
                  <label className="flex items-start gap-3 rounded-xl border border-slate-200 bg-slate-50 p-4 text-sm text-slate-700">
                    <input
                      type="checkbox"
                      className="mt-0.5 size-4 rounded border-slate-300 disabled:opacity-50"
                      checked={effectiveEmailVerified}
                      onChange={(event) => {
                        setEmailVerified(event.target.checked)
                        setEmailVerifiedTouched(true)
                      }}
                      disabled={!!user.scim_source}
                    />
                    <span>
                      <span className="block font-semibold text-slate-900">{t.saveAsVerified}</span>
                      <span className="mt-0.5 block text-xs leading-5 text-slate-500">
                        {t.verifiedOwnershipNotice}
                      </span>
                    </span>
                  </label>
                </section>
                <section className="grid gap-2 border-t border-slate-200 pt-5">
                  <h3 className="text-xs font-bold uppercase tracking-[0.1em] text-slate-400">
                    {t.rolesHeading}
                  </h3>
                  <Label htmlFor="user-editor-roles">{t.rolesHeading}</Label>
                  <Input
                    id="user-editor-roles"
                    value={roles}
                    onChange={(event) => setRoles(event.target.value)}
                    placeholder="admin, support"
                  />
                  <p className="text-xs leading-5 text-slate-500">{t.rolesHelp}</p>
                </section>
                <AdminAttributeEditorGroups
                  defs={attributeDefs}
                  values={attrDraft}
                  onChange={(key, next) => setAttrDraft((current) => ({ ...current, [key]: next }))}
                  readOnly={!!user.scim_source}
                />
              </div>
            )}
          </div>
          <div className="flex justify-end gap-2 border-t border-slate-200 bg-slate-50 px-6 py-4">
            {confirming ? (
              <Button type="button" variant="outline" onClick={() => setConfirming(false)}>
                {t.back}
              </Button>
            ) : (
              <Button asChild variant="outline">
                <a href={detailPath}>{t.cancel}</a>
              </Button>
            )}
            <Button type="submit" disabled={busy || usernameInvalid || !changed}>
              {confirming ? t.confirmChanges : rolesChanged ? t.confirmChangesHeading : t.save}
            </Button>
          </div>
        </form>
      </Card>
    </AdminShell>
  )
}

// REQUIRE_USERNAME_CONFIRMATION は削除確認としてユーザー名の再入力を求める
// オプション機能のスイッチ。既定では無効。誤操作の最終防御を強めたい運用では
// true にすると、削除前に対象のユーザー名タイピングを要求する。
const REQUIRE_USERNAME_CONFIRMATION: boolean = false

// DeleteUserDialog は削除前の確認ダイアログ。mode='soft' は削除予約 (復元可能)、
// mode='purge' は完全削除 (匿名化・不可逆)。ユーザー名の再入力確認は
// REQUIRE_USERNAME_CONFIRMATION が true のときだけ求める (既定は無効)。
function DeleteUserDialog({
  user,
  busy,
  mode,
  onClose,
  onConfirm,
}: {
  user: AdminUser
  busy: boolean
  mode: 'soft' | 'purge'
  onClose: () => void
  onConfirm: () => void
}) {
  const [confirmName, setConfirmName] = useState('')
  const canConfirm = !REQUIRE_USERNAME_CONFIRMATION || confirmName === user.preferred_username
  const purge = mode === 'purge'
  const t = useDictionary(adminUsersDictionary)

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!canConfirm) return
    onConfirm()
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/35 p-5 backdrop-blur-[2px]"
      role="dialog"
      aria-modal="true"
      aria-labelledby="delete-user-title"
    >
      <button
        type="button"
        className="absolute inset-0 cursor-default"
        aria-label={t.close}
        onClick={onClose}
      />
      <Card className="relative w-full max-w-lg overflow-hidden shadow-2xl">
        <div className="flex items-start justify-between border-b border-slate-200 px-6 py-5">
          <div className="flex gap-3">
            <span
              className={cn(
                'flex size-9 shrink-0 items-center justify-center rounded-full',
                purge ? 'bg-red-50 text-red-700' : 'bg-amber-50 text-amber-700',
              )}
            >
              <IconAlertTriangle size={18} aria-hidden="true" />
            </span>
            <div>
              <p
                className={cn(
                  'text-xs font-bold uppercase tracking-[0.12em]',
                  purge ? 'text-red-700' : 'text-amber-700',
                )}
              >
                {purge ? t.irreversibleAction : t.reversibleFor30Days}
              </p>
              <h2 id="delete-user-title" className="mt-1 text-xl font-semibold">
                {purge ? t.deleteUserPermanently : t.deleteUser}
              </h2>
              <p className="mt-1 text-sm text-slate-500">
                {user.name || user.preferred_username} (@{user.preferred_username})
              </p>
            </div>
          </div>
          <Button variant="ghost" className="px-2.5" onClick={onClose} aria-label={t.close}>
            <IconX size={18} aria-hidden="true" />
          </Button>
        </div>

        <form onSubmit={handleSubmit}>
          <div className="grid gap-5 p-6">
            {purge ? (
              <div className="rounded-xl border border-red-200 bg-red-50 p-4 text-xs leading-5 text-red-900">
                <p className="font-semibold">{t.purgeConsequencesHeading}</p>
                <ul className="mt-1.5 list-disc pl-5">
                  <li>{t.purgeConsequenceConsents}</li>
                  <li>{t.purgeConsequenceSessions}</li>
                  <li>{t.purgeConsequenceMfa}</li>
                  <li>{t.purgeConsequenceDeviceAuth}</li>
                </ul>
                <p className="mt-2">
                  {t.purgeSubNoteLead} <code>sub</code> {t.purgeSubNoteMid}
                  <strong>{t.purgeSubNoteStrong}</strong>
                </p>
              </div>
            ) : (
              <div className="rounded-xl border border-amber-200 bg-amber-50 p-4 text-xs leading-5 text-amber-900">
                <p className="font-semibold">{t.softDeleteRestorableNotice}</p>
                <p className="mt-1.5">{t.softDeleteDescription}</p>
              </div>
            )}

            {REQUIRE_USERNAME_CONFIRMATION && (
              <div className="grid gap-2">
                <Label htmlFor="delete-user-confirm">
                  {t.confirmUsernamePrefix}{' '}
                  <span className="font-mono text-slate-700">{user.preferred_username}</span>{' '}
                  {t.confirmUsernameSuffix}
                </Label>
                <Input
                  id="delete-user-confirm"
                  value={confirmName}
                  onChange={(event) => setConfirmName(event.target.value)}
                  autoFocus
                  autoComplete="off"
                />
              </div>
            )}
          </div>

          <div className="flex justify-end gap-2 border-t border-slate-200 bg-slate-50 px-6 py-4">
            <Button type="button" variant="outline" onClick={onClose} disabled={busy}>
              {t.cancel}
            </Button>
            <Button type="submit" variant="destructive" disabled={busy || !canConfirm}>
              <IconTrash size={16} aria-hidden="true" />
              {purge ? t.purgeConfirm : t.confirmDelete}
            </Button>
          </div>
        </form>
      </Card>
    </div>
  )
}

// DisableUserDialog は無効化 (disable) 前に挟む軽い確認ダイアログ。削除と違い
// 復元可能なため username typing は求めず、影響と復元動線の説明だけで確定させる
// (enable 方向はダイアログ無しで即時実行する)。
function DisableUserDialog({
  user,
  busy,
  onClose,
  onConfirm,
}: {
  user: AdminUser
  busy: boolean
  onClose: () => void
  onConfirm: () => void
}) {
  const t = useDictionary(adminUsersDictionary)
  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/35 p-5 backdrop-blur-[2px]"
      role="dialog"
      aria-modal="true"
      aria-labelledby="disable-user-title"
    >
      <button
        type="button"
        className="absolute inset-0 cursor-default"
        aria-label={t.close}
        onClick={onClose}
      />
      <Card className="relative w-full max-w-lg overflow-hidden shadow-2xl">
        <div className="flex items-start justify-between border-b border-slate-200 px-6 py-5">
          <div className="flex gap-3">
            <span className="flex size-9 shrink-0 items-center justify-center rounded-full bg-red-50 text-red-700">
              <IconBan size={18} aria-hidden="true" />
            </span>
            <div>
              <p className="text-xs font-bold uppercase tracking-[0.12em] text-red-700">
                {t.accountAccessBadge}
              </p>
              <h2 id="disable-user-title" className="mt-1 text-xl font-semibold">
                {t.disableAccount}
              </h2>
              <p className="mt-1 text-sm text-slate-500">
                {user.name || user.preferred_username} (@{user.preferred_username})
              </p>
            </div>
          </div>
          <Button variant="ghost" className="px-2.5" onClick={onClose} aria-label={t.close}>
            <IconX size={18} aria-hidden="true" />
          </Button>
        </div>

        <div className="grid gap-5 p-6">
          <div className="rounded-xl border border-red-200 bg-red-50 p-4 text-xs leading-5 text-red-900">
            <p className="font-semibold">{t.disableConsequencesHeading}</p>
            <ul className="mt-1.5 list-disc pl-5">
              <li>{t.disableConsequenceLogin}</li>
              <li>{t.disableConsequenceSessions}</li>
              <li>{t.disableConsequenceRefresh}</li>
            </ul>
            <p className="mt-2">
              {t.disableUndoNotePrefix}{' '}
              <span className="font-semibold">{t.disableUndoNoteEmphasis}</span>{' '}
              {t.disableUndoNoteSuffix}
            </p>
          </div>
        </div>

        <div className="flex justify-end gap-2 border-t border-slate-200 bg-slate-50 px-6 py-4">
          <Button type="button" variant="outline" onClick={onClose} disabled={busy}>
            {t.cancel}
          </Button>
          <Button type="button" variant="destructive" disabled={busy} onClick={onConfirm}>
            <IconBan size={16} aria-hidden="true" />
            {t.disableConfirm}
          </Button>
        </div>
      </Card>
    </div>
  )
}

export function AdminUserCreatePage({
  csrfToken,
  actorUsername,
}: {
  csrfToken: string
  actorUsername?: string
}) {
  const listPath = tenantURL('/admin/users')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const t = useDictionary(adminUsersDictionary)

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const form = event.currentTarget
    const data = new FormData(form)
    const username = String(data.get('preferred_username') ?? '').trim()
    const password = String(data.get('password') ?? '')

    if (!username || !password) return

    setBusy(true)
    setError('')

    try {
      const created = await createAdminUser(csrfToken, {
        preferred_username: username,
        password: password,
        name: optionalValue(data.get('name')),
        email: optionalValue(data.get('email')),
        email_verified: data.get('email_verified') === 'on',
        roles: parseRoles(String(data.get('roles') ?? '')),
      })
      window.location.assign(tenantURL(`/admin/users/${encodeURIComponent(created.id)}`))
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.createUserFailedError)
      setBusy(false)
    }
  }

  return (
    <AdminShell
      active="users"
      actorUsername={actorUsername}
      title={t.addUser}
      description={t.createUserDescription}
    >
      <div className="flex items-center gap-3">
        <a
          href={listPath}
          className="inline-flex size-9 items-center justify-center rounded-lg border border-slate-200 bg-white text-slate-700 transition hover:bg-slate-50 hover:text-slate-900"
          aria-label={t.backToUserListAria}
        >
          <IconArrowLeft size={18} aria-hidden="true" />
        </a>
        <h1 className="text-2xl font-bold tracking-tight text-slate-900">{t.addUser}</h1>
      </div>

      <div className="mt-6 max-w-2xl">
        <Card className="shadow-[0_1px_2px_rgb(15_23_42/4%)]">
          <form onSubmit={handleSubmit}>
            <div className="grid gap-6 p-6">
              {error && <Alert>{error}</Alert>}

              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                <Field id="preferred_username" label={t.username} required />
                <Field id="name" label={t.displayName} />
              </div>

              <Field id="email" label={t.emailFieldLabel} type="email" />

              <Field
                id="password"
                label={t.initialPasswordLabel}
                type="password"
                required
                minLength={12}
                description={t.initialPasswordDescription}
              />

              <Field
                id="roles"
                label={t.initialRolesLabel}
                placeholder="support, admin"
                description={t.initialRolesDescription}
              />

              <label className="flex items-start gap-3 rounded-xl border border-slate-200 bg-slate-50 p-4 text-sm text-slate-700 cursor-pointer">
                <input
                  name="email_verified"
                  type="checkbox"
                  className="mt-0.5 size-4 rounded border-slate-300"
                />
                <span>
                  <span className="block font-semibold text-slate-900">{t.createAsVerified}</span>
                  <span className="mt-0.5 block text-xs leading-5 text-slate-500">
                    {t.verifiedOwnershipNotice}
                  </span>
                </span>
              </label>
            </div>

            <div className="flex justify-end gap-2 border-t border-slate-200 bg-slate-50 px-6 py-4">
              <a
                href={listPath}
                className="inline-flex h-9 items-center justify-center rounded-lg border border-slate-200 bg-white px-4 text-sm font-medium text-slate-700 shadow-sm transition hover:bg-slate-50 hover:text-slate-900"
              >
                {t.cancel}
              </a>
              <Button type="submit" disabled={busy}>
                <IconUserPlus size={16} aria-hidden="true" />
                {t.create}
              </Button>
            </div>
          </form>
        </Card>
      </div>
    </AdminShell>
  )
}

const USER_IMPORT_CSV_TEMPLATE = 'preferred_username,email,name,roles\n'
const USER_IMPORT_POLL_INTERVAL_MS = 1000
const USER_IMPORT_POLL_MAX_ATTEMPTS = 30

class UserImportTimeoutError extends Error {}
class UserImportJobFailedError extends Error {}

async function pollUserImportJob(jobId: string): Promise<UserImportResult> {
  for (let attempt = 0; attempt < USER_IMPORT_POLL_MAX_ATTEMPTS; attempt++) {
    const job = await getAdminUserImport(jobId)
    if (job.status === 'succeeded') {
      return job.result ?? { total_rows: 0, accepted_rows: 0, rejected_rows: 0 }
    }
    if (job.status === 'failed' || job.status === 'canceled') {
      throw new UserImportJobFailedError()
    }
    await new Promise((resolve) => setTimeout(resolve, USER_IMPORT_POLL_INTERVAL_MS))
  }
  throw new UserImportTimeoutError()
}

// stable error code だけを既知の翻訳に対応付ける。未登録 code は backend の原文 (code 自体) を
// そのまま出す (errorMessage.ts と同じ方針)。
function importRowErrorMessage(t: typeof adminUsersDictionary.ja, code: string): string {
  switch (code) {
    case 'csv_too_large':
      return t.importErrorCsvTooLarge
    case 'too_many_rows':
      return t.importErrorTooManyRows
    case 'field_too_large':
      return t.importErrorFieldTooLarge
    case 'invalid_header':
      return t.importErrorInvalidHeader
    case 'invalid_csv':
      return t.importErrorInvalidCsv
    case 'invalid_column_count':
      return t.importErrorInvalidColumnCount
    case 'required':
      return t.importErrorRequired
    case 'duplicate_username':
      return t.importErrorDuplicateUsername
    case 'invalid_email':
      return t.importErrorInvalidEmail
    case 'username_conflict':
      return t.importErrorUsernameConflict
    case 'invalid_user':
      return t.importErrorInvalidUser
    default:
      return code
  }
}

function importColumnLabel(t: typeof adminUsersDictionary.ja, column: string | undefined): string {
  switch (column) {
    case 'preferred_username':
      return t.username
    case 'email':
      return t.emailFieldLabel
    case 'name':
      return t.displayName
    case 'roles':
      return t.rolesHeading
    default:
      return column ?? ''
  }
}

function importSubmitErrorMessage(t: typeof adminUsersDictionary.ja, cause: unknown): string {
  if (cause instanceof UserImportTimeoutError) return t.importTimeoutError
  if (cause instanceof UserImportJobFailedError) return t.importJobFailedError
  if (cause instanceof AuthenticationAPIError) {
    if (cause.code) {
      const mapped = importRowErrorMessage(t, cause.code)
      if (mapped !== cause.code) return mapped
    }
    return cause.message
  }
  return t.genericActionError
}

function UserImportResultSummary({
  t,
  title,
  result,
  success = false,
}: {
  t: typeof adminUsersDictionary.ja
  title: string
  result: UserImportResult
  success?: boolean
}) {
  return (
    <div className="grid gap-3">
      <h2 className="text-sm font-semibold text-slate-900">{title}</h2>
      <div className="grid grid-cols-3 gap-3">
        {[
          { label: t.importTotalRows, value: result.total_rows },
          { label: t.importAcceptedRows, value: result.accepted_rows },
          { label: t.importRejectedRows, value: result.rejected_rows },
        ].map((item) => (
          <div
            key={item.label}
            className="rounded-lg border border-slate-200 bg-white p-3 text-center"
          >
            <p className="text-xl font-semibold text-slate-900">{item.value}</p>
            <p className="text-xs text-slate-500">{item.label}</p>
          </div>
        ))}
      </div>
      {success && (
        <Alert variant="success">
          {t.importApplySuccessNotice.replace('{count}', String(result.accepted_rows))}
        </Alert>
      )}
      {result.errors && result.errors.length > 0 && (
        <div className="overflow-hidden rounded-xl border border-slate-200">
          <table className="w-full text-left text-sm">
            <thead className="bg-slate-50 text-xs font-semibold uppercase tracking-wide text-slate-500">
              <tr>
                <th className="px-4 py-2">{t.importRowColumnHeader}</th>
                <th className="px-4 py-2">{t.importFieldColumnHeader}</th>
                <th className="px-4 py-2">{t.importErrorColumnHeader}</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {result.errors.map((rowError: UserImportRowError) => (
                <tr key={`${rowError.row}-${rowError.column ?? ''}-${rowError.code}`}>
                  <td className="px-4 py-2 font-mono text-xs">{rowError.row}</td>
                  <td className="px-4 py-2 text-xs text-slate-600">
                    {importColumnLabel(t, rowError.column)}
                  </td>
                  <td className="px-4 py-2">{importRowErrorMessage(t, rowError.code)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}

function ApplyImportConfirmDialog({
  result,
  busy,
  onClose,
  onConfirm,
}: {
  result: UserImportResult
  busy: boolean
  onClose: () => void
  onConfirm: () => void
}) {
  const t = useDictionary(adminUsersDictionary)
  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/35 p-5 backdrop-blur-[2px]"
      role="dialog"
      aria-modal="true"
      aria-labelledby="apply-import-title"
    >
      <button
        type="button"
        className="absolute inset-0 cursor-default"
        aria-label={t.close}
        onClick={onClose}
      />
      <Card className="relative w-full max-w-lg overflow-hidden shadow-2xl">
        <div className="flex items-start justify-between border-b border-slate-200 px-6 py-5">
          <div className="flex gap-3">
            <span className="flex size-9 shrink-0 items-center justify-center rounded-full bg-amber-50 text-amber-700">
              <IconAlertTriangle size={18} aria-hidden="true" />
            </span>
            <div>
              <h2 id="apply-import-title" className="text-xl font-semibold">
                {t.applyImportConfirmTitle}
              </h2>
              <p className="mt-1 text-sm text-slate-500">
                {t.applyImportConfirmDescription.replace('{count}', String(result.accepted_rows))}
              </p>
            </div>
          </div>
          <Button variant="ghost" className="px-2.5" onClick={onClose} aria-label={t.close}>
            <IconX size={18} aria-hidden="true" />
          </Button>
        </div>
        <div className="flex justify-end gap-2 border-t border-slate-200 bg-slate-50 px-6 py-4">
          <Button type="button" variant="outline" onClick={onClose} disabled={busy}>
            {t.cancel}
          </Button>
          <Button type="button" onClick={onConfirm} disabled={busy}>
            <IconCheck size={16} aria-hidden="true" />
            {t.applyImportConfirmButton}
          </Button>
        </div>
      </Card>
    </div>
  )
}

type UserImportStep =
  | 'select'
  | 'dry_run_running'
  | 'dry_run_result'
  | 'apply_running'
  | 'apply_result'

// AdminUserImportPage は CSV アップロード → dry-run 検証プレビュー → 明示確認 → apply の
// ウィザード (wi-202)。CSV は常にクライアント側で UTF-8 text として読み、apply は必ず
// dry-run と同じ内容を送る (差し替え防止のため再アップロードは求めない)。
export function AdminUserImportPage({
  csrfToken,
  actorUsername,
}: {
  csrfToken: string
  actorUsername?: string
}) {
  const listPath = tenantURL('/admin/users')
  const t = useDictionary(adminUsersDictionary)
  const [step, setStep] = useState<UserImportStep>('select')
  const [fileName, setFileName] = useState('')
  const [csvText, setCsvText] = useState('')
  const [dryRunResult, setDryRunResult] = useState<UserImportResult | null>(null)
  const [applyResult, setApplyResult] = useState<UserImportResult | null>(null)
  const [showApplyConfirm, setShowApplyConfirm] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  function downloadTemplate() {
    const blob = new Blob([USER_IMPORT_CSV_TEMPLATE], { type: 'text/csv;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const anchor = document.createElement('a')
    anchor.href = url
    anchor.download = 'user-import-template.csv'
    anchor.click()
    URL.revokeObjectURL(url)
  }

  async function handleFileChange(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0]
    event.target.value = ''
    if (!file) return
    setError('')
    setDryRunResult(null)
    setApplyResult(null)
    setStep('select')
    try {
      setCsvText(await file.text())
      setFileName(file.name)
    } catch {
      setError(t.importFileReadError)
    }
  }

  async function runDryRun() {
    if (!csvText) return
    setBusy(true)
    setError('')
    setStep('dry_run_running')
    try {
      const job = await importAdminUsers(csrfToken, { csv: csvText, mode: 'dry_run' })
      setDryRunResult(await pollUserImportJob(job.id))
      setStep('dry_run_result')
    } catch (cause) {
      setError(importSubmitErrorMessage(t, cause))
      setStep('select')
    } finally {
      setBusy(false)
    }
  }

  async function runApply() {
    if (!csvText) return
    setBusy(true)
    setError('')
    setShowApplyConfirm(false)
    setStep('apply_running')
    try {
      const job = await importAdminUsers(csrfToken, { csv: csvText, mode: 'apply' })
      setApplyResult(await pollUserImportJob(job.id))
      setStep('apply_result')
    } catch (cause) {
      setError(importSubmitErrorMessage(t, cause))
      setStep('dry_run_result')
    } finally {
      setBusy(false)
    }
  }

  function reset() {
    setStep('select')
    setFileName('')
    setCsvText('')
    setDryRunResult(null)
    setApplyResult(null)
    setError('')
  }

  const canRunDryRun = Boolean(csvText) && step === 'select' && !busy
  const canApply = (dryRunResult?.accepted_rows ?? 0) > 0

  return (
    <AdminShell
      active="users"
      actorUsername={actorUsername}
      title={t.importUsers}
      description={t.importUsersDescription}
    >
      <div className="flex items-center gap-3">
        <a
          href={listPath}
          className="inline-flex size-9 items-center justify-center rounded-lg border border-slate-200 bg-white text-slate-700 transition hover:bg-slate-50 hover:text-slate-900"
          aria-label={t.backToUserListAria}
        >
          <IconArrowLeft size={18} aria-hidden="true" />
        </a>
        <h1 className="text-2xl font-bold tracking-tight text-slate-900">{t.importUsers}</h1>
      </div>

      <div className="mt-6 max-w-3xl">
        {error && <Alert className="mb-4">{error}</Alert>}

        <Card className="shadow-[0_1px_2px_rgb(15_23_42/4%)]">
          <div className="grid gap-6 p-6">
            <div className="rounded-xl border border-slate-200 bg-slate-50 p-4 text-sm leading-6 text-slate-700">
              <p>{t.importInstructions}</p>
              <p className="mt-1 font-semibold text-slate-900">
                {t.importPasswordColumnRejectedNotice}
              </p>
              <Button type="button" variant="outline" className="mt-3" onClick={downloadTemplate}>
                <IconDownload size={16} aria-hidden="true" />
                {t.downloadTemplate}
              </Button>
            </div>

            <div className="grid gap-2">
              <Label htmlFor="import-csv-file">{t.selectCsvFile}</Label>
              <input
                id="import-csv-file"
                type="file"
                accept=".csv,text/csv"
                onChange={(event) => void handleFileChange(event)}
                disabled={busy}
                className="block w-full text-sm text-slate-700 file:mr-3 file:rounded-lg file:border-0 file:bg-slate-950 file:px-3 file:py-2 file:text-sm file:font-semibold file:text-white"
              />
              {fileName && (
                <p className="text-xs text-slate-500">
                  {t.selectedFileLabel.replace('{name}', fileName)}
                </p>
              )}
            </div>

            {(step === 'dry_run_running' || step === 'apply_running') && (
              <p className="flex items-center gap-2 text-sm text-slate-600">
                <IconClock size={16} className="animate-pulse" aria-hidden="true" />
                {step === 'dry_run_running' ? t.dryRunRunning : t.applyRunning}
              </p>
            )}

            {dryRunResult && step !== 'apply_running' && step !== 'apply_result' && (
              <UserImportResultSummary t={t} title={t.dryRunResultTitle} result={dryRunResult} />
            )}

            {applyResult && step === 'apply_result' && (
              <UserImportResultSummary
                t={t}
                title={t.applyResultTitle}
                result={applyResult}
                success
              />
            )}
          </div>

          <div className="flex justify-end gap-2 border-t border-slate-200 bg-slate-50 px-6 py-4">
            <a
              href={listPath}
              className="inline-flex h-9 items-center justify-center rounded-lg border border-slate-200 bg-white px-4 text-sm font-medium text-slate-700 shadow-sm transition hover:bg-slate-50 hover:text-slate-900"
            >
              {step === 'apply_result' ? t.backToUserList : t.cancel}
            </a>
            {step === 'select' && (
              <Button type="button" disabled={!canRunDryRun} onClick={() => void runDryRun()}>
                <IconUpload size={16} aria-hidden="true" />
                {t.runDryRun}
              </Button>
            )}
            {step === 'dry_run_result' && (
              <>
                <Button type="button" variant="outline" onClick={reset} disabled={busy}>
                  {t.startOver}
                </Button>
                <Button
                  type="button"
                  disabled={!canApply || busy}
                  onClick={() => setShowApplyConfirm(true)}
                >
                  <IconCheck size={16} aria-hidden="true" />
                  {t.applyImport}
                </Button>
              </>
            )}
          </div>
        </Card>
      </div>

      {showApplyConfirm && dryRunResult && (
        <ApplyImportConfirmDialog
          result={dryRunResult}
          busy={busy}
          onClose={() => setShowApplyConfirm(false)}
          onConfirm={() => void runApply()}
        />
      )}
    </AdminShell>
  )
}
