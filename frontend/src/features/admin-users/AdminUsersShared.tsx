import { IconDeviceLaptop, IconShield, IconUserPlus, IconUsersGroup } from '@tabler/icons-react'
import { useCallback, useEffect, useState } from 'react'
import {
  addAdminGroupMember,
  AuthenticationAPIError,
  getAdminUserGroups,
  listAdminGroups,
  listAdminUserSessions,
  revokeAdminUserSession,
  revokeAllAdminUserSessions,
  tenantURL,
} from '../../api'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { useDictionary, useLocale } from '../../lib/i18n'
import {
  domainLabelsDictionary,
  type DomainLabelsDictionary,
} from '../../lib/i18n/domainLabels.i18n'
import { attributeGroupKey, attributeGroupTitle, cn } from '../../lib/utils'
import { REQUIRED_ACTIONS, requiredActionLabel } from '../../types'
import type {
  AdminGroup,
  AdminSessionRecord,
  AdminUser,
  AdminUserGroups,
  AttributeValue,
  UserAttributeDef,
} from '../../types'
import { adminUsersDictionary } from './AdminUsersPage.i18n'
import { formatDateTime, RoleList } from './AdminUsersPrimitives'

export function attributeValueToText(value: AttributeValue): string {
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

export function groupedAttributeDefs(defs: UserAttributeDef[], t: DomainLabelsDictionary) {
  const groups = new Map<ReturnType<typeof attributeGroupKey>, UserAttributeDef[]>()
  for (const def of defs) {
    const key = attributeGroupKey(def)
    groups.set(key, [...(groups.get(key) ?? []), def])
  }
  return (['profile', 'organization', 'custom'] as const)
    .map((key) => ({ key, title: attributeGroupTitle(key, t), defs: groups.get(key) ?? [] }))
    .filter((group) => group.defs.length > 0)
}

export function UserRequiredActionsSection({
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

// SectionVariant は「他コンテンツの下に積む sub-section」と「単独の card として
// 独立させる」の2通りの見出しレベルを切り替える。同じ AdminUser* section を、
// AdminUsersListPage (Profile section の下に積む一覧右ペイン) と
// AdminUserDetailPage (Profile/Attributes/Lifecycle と並ぶ独立カード) の
// 両方で見た目の階層を揃えて使い回すための共通 prop。
type SectionVariant = 'section' | 'card'

const sectionHeadingClassName: Record<SectionVariant, string> = {
  section: 'text-sm font-semibold text-slate-900',
  card: 'text-xs font-bold uppercase tracking-[0.1em] text-slate-400',
}

// allowEditing=false は一覧右ペインのような参照専用の文脈で使う。所属ロール /
// グループは読み取り表示のみとし、グループ追加の操作 UI は出さない。実際の追加は
// 専用の詳細画面 (allowEditing=true) から行う。
export function UserGroupsSection({
  user,
  csrfToken,
  variant = 'section',
  allowEditing = true,
}: {
  user: AdminUser
  csrfToken: string
  variant?: SectionVariant
  allowEditing?: boolean
}) {
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
    <section className={cn(variant === 'section' && 'border-t border-slate-200 pt-5')}>
      <div className="flex items-center justify-between">
        <div>
          <h3 className={sectionHeadingClassName[variant]}>{t.rolesAndGroupsHeading}</h3>
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
        <div className="mt-2 rounded-xl border border-slate-200 bg-white">
          <div className="p-3">
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

          {allowEditing && addable.length > 0 && (
            <div className="flex items-center gap-2 border-t border-slate-100 p-3">
              <select
                value={selectedGroup}
                onChange={(event) => setSelectedGroup(event.target.value)}
                disabled={adding}
                aria-label={t.selectGroupPlaceholder}
                className="h-10 flex-1 rounded-lg border border-slate-200 bg-white px-2 text-sm text-slate-700"
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
                className="h-10"
                disabled={adding || !selectedGroup}
                onClick={() => void handleAdd()}
              >
                <IconUserPlus size={16} aria-hidden="true" />
                {t.add}
              </Button>
            </div>
          )}
        </div>
      </div>
    </section>
  )
}

// sessionAmrSummary は AMR コードを人間可読なラベル列に変換する
// (frontend/src/features/account/AccountActivityPage.tsx の同名処理と対をなす
// admin 版。管理者向け辞書 (adminUsersDictionary) の sessionAmr* キーを使う)。
export function sessionAmrSummary(amr: string[], t: Record<string, string>): string {
  if (amr.length === 0) return t.sessionAmrUnknown
  const labels: Record<string, string> = {
    pwd: t.sessionAmrPwd,
    otp: t.sessionAmrOtp,
    webauthn: t.sessionAmrWebauthn,
    rc: t.sessionAmrRc,
    mfa: t.sessionAmrMfa,
    hwk: t.sessionAmrHwk,
    swk: t.sessionAmrSwk,
  }
  return amr.map((code) => labels[code] ?? code).join(' + ')
}

// Go's zero time.Time marshals as this exact string. A session that was only
// ever used to issue OAuth tokens (never resolved through the browser-cookie
// path) never touches LastSeenAt, so admins would otherwise see a bogus
// "started 0001-01-01" row instead of a plain "never" indicator.
const ZERO_TIME_PREFIX = '0001-01-01'

export function sessionLastSeenLabel(
  lastSeenAt: string,
  t: Record<string, string>,
  locale: 'ja' | 'en',
): string {
  if (lastSeenAt.startsWith(ZERO_TIME_PREFIX)) return t.sessionNeverSeen
  return t.sessionLastSeen.replace('{date}', formatDateTime(lastSeenAt, locale))
}

function AdminSessionRow({
  session,
  busy,
  onRevoke,
}: {
  session: AdminSessionRecord
  busy: boolean
  onRevoke: () => void
}) {
  const t = useDictionary(adminUsersDictionary)
  const { locale } = useLocale()
  return (
    <li className="flex items-start justify-between gap-3 px-4 py-3">
      <div className="min-w-0">
        <p className="text-sm text-slate-700">{sessionAmrSummary(session.amr, t)}</p>
        <p className="mt-0.5 text-xs text-slate-500">
          {t.sessionStarted.replace('{date}', formatDateTime(session.started_at, locale))}
        </p>
        <p className="text-xs text-slate-400">
          {sessionLastSeenLabel(session.last_seen_at, t, locale)}
        </p>
      </div>
      <Button
        type="button"
        variant="outline"
        className="h-8 shrink-0 self-center px-3 text-xs"
        disabled={busy}
        onClick={onRevoke}
      >
        {busy ? t.endingSession : t.endSession}
      </Button>
    </li>
  )
}

// UserSessionsSection は admin がユーザー詳細画面から対象ユーザーの有効な
// セッションを確認・個別終了・全終了できるようにする (wi-28 T007, ADR-127 決定9)。
// 終了 (revoke) はサーバー側で同じ sid を共有する RefreshTokenRecord も
// family/client を横断して失効させる (T004 の RevokeTokensBySid)。
export function UserSessionsSection({
  user,
  csrfToken,
  variant = 'section',
}: {
  user: AdminUser
  csrfToken: string
  variant?: SectionVariant
}) {
  const [sessions, setSessions] = useState<AdminSessionRecord[] | null>(null)
  const [error, setError] = useState('')
  const [busyId, setBusyId] = useState<string | null>(null)
  const [busyAll, setBusyAll] = useState(false)
  const { id } = user
  const t = useDictionary(adminUsersDictionary)

  const load = useCallback(async () => {
    try {
      setSessions(await listAdminUserSessions(id))
      setError('')
    } catch (err) {
      setError(err instanceof AuthenticationAPIError ? err.message : t.sessionsFetchError)
    }
  }, [id, t.sessionsFetchError])

  useEffect(() => {
    setSessions(null)
    void load()
  }, [load])

  async function handleRevoke(sessionID: string) {
    setBusyId(sessionID)
    setError('')
    try {
      await revokeAdminUserSession(csrfToken, id, sessionID)
      await load()
    } catch (err) {
      setError(err instanceof AuthenticationAPIError ? err.message : t.sessionRevokeError)
    } finally {
      setBusyId(null)
    }
  }

  async function handleRevokeAll() {
    if (!window.confirm(t.sessionsRevokeAllConfirm)) return
    setBusyAll(true)
    setError('')
    try {
      await revokeAllAdminUserSessions(csrfToken, id)
      await load()
    } catch (err) {
      setError(err instanceof AuthenticationAPIError ? err.message : t.sessionsRevokeAllError)
    } finally {
      setBusyAll(false)
    }
  }

  return (
    <section className={cn(variant === 'section' && 'border-t border-slate-200 pt-5')}>
      <div className="flex items-center justify-between">
        <div>
          <h3 className={sectionHeadingClassName[variant]}>{t.sessionsHeading}</h3>
          <p className="mt-0.5 text-xs text-slate-500">{t.sessionsDescription}</p>
        </div>
        {sessions && sessions.length > 0 ? (
          <Button
            type="button"
            variant="outline"
            className="h-9 px-3 text-xs"
            disabled={busyAll}
            onClick={() => void handleRevokeAll()}
          >
            {t.revokeAllSessions}
          </Button>
        ) : null}
      </div>

      {error && (
        <Alert variant="destructive" className="mt-3">
          {error}
        </Alert>
      )}

      <div className="mt-3 overflow-hidden rounded-xl border border-slate-200">
        {!sessions || sessions.length === 0 ? (
          <div className="flex items-center gap-3 px-4 py-6 text-sm text-slate-500">
            <IconDeviceLaptop size={18} className="text-slate-400" aria-hidden="true" />
            {sessions ? t.noSessions : ''}
          </div>
        ) : (
          <ul className="divide-y divide-slate-100">
            {sessions.map((session) => (
              <AdminSessionRow
                key={session.id}
                session={session}
                busy={busyId === session.id}
                onRevoke={() => void handleRevoke(session.id)}
              />
            ))}
          </ul>
        )}
      </div>
    </section>
  )
}
