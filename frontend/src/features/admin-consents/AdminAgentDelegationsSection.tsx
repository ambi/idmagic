import { Card } from '../../components/ui/card'
import { useDictionary, useFormatters } from '../../lib/i18n'
import type { AdminAgentDelegation } from '../../types'
import { adminConsentsDictionary } from './AdminConsentsPage.i18n'
import { ConsentStateBadge, NameWithId } from './AdminConsentsPage'

// 代行の付与は参照だけで、管理者は付与も撤回もできない (REQ-OAUTH2-059)。
export function AdminAgentDelegationsSection({
  delegations,
}: {
  delegations: AdminAgentDelegation[]
}) {
  const t = useDictionary(adminConsentsDictionary)
  const { formatDate } = useFormatters()
  return (
    <section className="grid gap-3" aria-labelledby="admin-agent-delegations-heading">
      <div>
        <h2 id="admin-agent-delegations-heading" className="text-base font-semibold text-slate-900">
          {t.delegationsHeading}
        </h2>
        <p className="mt-1 text-sm text-slate-500">{t.delegationsDescription}</p>
      </div>
      <Card className="overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-slate-50 text-left text-xs font-semibold uppercase tracking-wide text-slate-500">
            <tr>
              <th className="px-4 py-3">{t.tableHeaderUser}</th>
              <th className="px-4 py-3">{t.delegationsHeaderAgent}</th>
              <th className="px-4 py-3">{t.tableHeaderStatus}</th>
              <th className="px-4 py-3">{t.delegationsHeaderScopes}</th>
              <th className="px-4 py-3">{t.delegationsHeaderExpires}</th>
              <th className="px-4 py-3">{t.delegationsHeaderLastUsed}</th>
            </tr>
          </thead>
          <tbody>
            {delegations.length === 0 ? (
              <tr>
                <td colSpan={6} className="px-4 py-12 text-center text-sm text-slate-500">
                  {t.delegationsEmpty}
                </td>
              </tr>
            ) : null}
            {delegations.map((delegation) => (
              <tr
                key={`${delegation.user_id}:${delegation.agent_id}`}
                className="border-t border-slate-100"
              >
                <td className="px-4 py-3">
                  <NameWithId
                    name={delegation.preferred_username ?? delegation.user_id}
                    id={delegation.user_id}
                  />
                </td>
                <td className="px-4 py-3">
                  <NameWithId name={delegation.agent_name} id={delegation.agent_id} />
                </td>
                <td className="px-4 py-3">
                  <ConsentStateBadge state={delegation.state} />
                </td>
                <td className="px-4 py-3 font-mono text-xs text-slate-600">
                  {delegation.scopes.join(' ')}
                </td>
                <td className="px-4 py-3 text-xs text-slate-500">
                  {formatDate(delegation.expires_at)}
                </td>
                <td className="px-4 py-3 text-xs text-slate-500">
                  {delegation.last_used_at ? formatDate(delegation.last_used_at) : '—'}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </Card>
    </section>
  )
}
