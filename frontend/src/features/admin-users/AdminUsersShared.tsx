import { IconShield, IconUserPlus, IconUsersGroup } from '@tabler/icons-react'
import { useCallback, useEffect, useState } from 'react'
import {
  addAdminGroupMember,
  AuthenticationAPIError,
  getAdminUserGroups,
  listAdminGroups,
  tenantURL,
} from '../../api'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { useDictionary } from '../../lib/i18n'
import {
  domainLabelsDictionary,
  type DomainLabelsDictionary,
} from '../../lib/i18n/domainLabels.i18n'
import { attributeGroupKey, attributeGroupTitle, cn } from '../../lib/utils'
import { REQUIRED_ACTIONS, requiredActionLabel } from '../../types'
import type {
  AdminGroup,
  AdminUser,
  AdminUserGroups,
  AttributeValue,
  UserAttributeDef,
} from '../../types'
import { adminUsersDictionary } from './AdminUsersPage.i18n'
import { RoleList } from './AdminUsersPrimitives'

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

export function UserGroupsSection({ user, csrfToken }: { user: AdminUser; csrfToken: string }) {
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
