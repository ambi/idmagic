import {
  IconAlertTriangle,
  IconArrowLeft,
  IconBan,
  IconCheck,
  IconCircleCheck,
  IconClock,
  IconDotsVertical,
  IconKey,
  IconMail,
  IconPencil,
  IconRefresh,
  IconShield,
  IconShieldCheck,
  IconTrash,
  IconUser,
  IconX,
} from '@tabler/icons-react'
import { useState } from 'react'
import {
  AuthenticationAPIError,
  clearAdminUserRequiredAction,
  deleteAdminUser,
  getAdminUser,
  issueMfaEnrollmentBypass,
  restoreAdminUser,
  revokeMfaEnrollmentBypass,
  setAdminUserDisabled,
  setAdminUserRequiredAction,
  tenantURL,
} from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Toast } from '../../components/ui/toast'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '../../components/ui/dropdown-menu'
import { useDictionary, useLocale } from '../../lib/i18n'
import { domainLabelsDictionary } from '../../lib/i18n/domainLabels.i18n'
import { attributeLabel } from '../../lib/utils'
import type {
  AdminUser,
  AttributeValue,
  TenantUserAttributeSchema,
  UserAttributeDef,
} from '../../types'
import { DeleteUserDialog, DisableUserDialog } from './AdminUserDialogs'
import { adminUsersDictionary } from './AdminUsersPage.i18n'
import {
  daysUntil,
  DetailRow,
  formatDateTime,
  StatusBadge,
  UserAvatar,
  userLifecycleStatus,
} from './AdminUsersPrimitives'
import {
  attributeValueToText,
  groupedAttributeDefs,
  UserGroupsSection,
  UserRequiredActionsSection,
} from './AdminUsersShared'

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
