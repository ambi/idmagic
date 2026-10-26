import { useState } from 'react'
import { AuthenticationAPIError, setAdminGroupRoleDelegation } from '../../api'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { useDictionary } from '../../lib/i18n'
import type { AdminGroup } from '../../types'
import { groupRoleDelegationDictionary } from './GroupRoleDelegationPanel.i18n'

export function GroupRoleDelegationPanel({
  csrfToken,
  group,
  unsavedChanges,
}: {
  csrfToken: string
  group: AdminGroup
  unsavedChanges: boolean
}) {
  const t = useDictionary(groupRoleDelegationDictionary)
  const [delegation, setDelegation] = useState(group.role_delegation)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const expression = group.dynamic_rule?.expression ?? ''
  const approved =
    delegation?.expression === expression &&
    JSON.stringify(delegation.roles) === JSON.stringify(group.roles)
  const cannotApprove =
    unsavedChanges ||
    group.roles.length === 0 ||
    group.roles.some((role) => role === 'admin' || role === 'system_admin') ||
    (group.membership_type === 'dynamic' && !group.dynamic_rule)

  async function save(revoke: boolean) {
    setBusy(true)
    setError('')
    try {
      const saved = await setAdminGroupRoleDelegation(csrfToken, group.id, {
        roles: revoke ? [] : group.roles,
        expression: revoke ? '' : expression,
      })
      setDelegation(saved.role_delegation)
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.failed)
    } finally {
      setBusy(false)
    }
  }

  return (
    <Card className="grid gap-4 p-6">
      <h2 className="font-semibold text-slate-900">{t.title}</h2>
      <p className="text-sm text-slate-600">{t.explanation}</p>
      <p className="text-sm text-slate-600">{t.adminNotice}</p>
      <p className="text-sm font-medium" role="status">
        {approved ? t.approved : delegation ? t.stale : t.pending}
      </p>
      <p className="font-mono text-sm">{group.roles.join(', ') || t.noRoles}</p>
      {group.membership_type === 'dynamic' && (
        <div className="grid gap-2">
          <p className="text-sm">{t.savedRule}</p>
          <pre className="overflow-x-auto whitespace-pre-wrap text-sm">{expression || '—'}</pre>
        </div>
      )}
      {unsavedChanges && <p className="text-sm text-amber-700">{t.saveFirst}</p>}
      {error && <Alert variant="destructive">{error}</Alert>}
      <div className="flex flex-wrap gap-2">
        <Button
          type="button"
          disabled={busy || cannotApprove || approved}
          onClick={() => void save(false)}
        >
          {t.approve}
        </Button>
        {delegation && (
          <Button type="button" variant="outline" disabled={busy} onClick={() => void save(true)}>
            {t.revoke}
          </Button>
        )}
      </div>
    </Card>
  )
}
