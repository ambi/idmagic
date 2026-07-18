import {
  IconAdjustments,
  IconBan,
  IconCheck,
  IconCircleCheck,
  IconClock,
  IconKey,
  IconMail,
  IconRefresh,
  IconSearch,
  IconShield,
  IconShieldCheck,
  IconTrash,
  IconUpload,
  IconUser,
  IconUserPlus,
  IconUsers,
} from '@tabler/icons-react'
import { useMemo, useState } from 'react'
import {
  AuthenticationAPIError,
  clearAdminUserRequiredAction,
  deleteAdminUser,
  listAdminUsers,
  restoreAdminUser,
  setAdminUserDisabled,
  setAdminUserRequiredAction,
  tenantURL,
} from '../../api'
import { AdminPaneActions } from '../../components/AdminPaneActions'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Toast } from '../../components/ui/toast'
import { useDictionary, useLocale } from '../../lib/i18n'
import { cn } from '../../lib/utils'
import type { AdminUser } from '../../types'
import { DeleteUserDialog, DisableUserDialog } from './AdminUserDialogs'
import { adminUsersDictionary } from './AdminUsersPage.i18n'
import {
  daysUntil,
  DetailRow,
  formatDateTime,
  Metric,
  RoleList,
  StatusBadge,
  UserAvatar,
  userLifecycleStatus,
} from './AdminUsersPrimitives'
import { UserGroupsSection, UserRequiredActionsSection } from './AdminUsersShared'

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
