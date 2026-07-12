import { IconRefresh, IconShieldCheck } from '@tabler/icons-react'
import { useState } from 'react'
import { AuthenticationAPIError, listTenantKeyHealth } from '../../api'
import { SystemShell } from '../../components/SystemShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { useDictionary } from '../../lib/i18n'
import type { TenantKeyHealth } from '../../types'
import { systemKeyHealthDictionary } from './SystemKeyHealthPage.i18n'

export function SystemKeyHealthPage({
  actorUsername,
  tenants: initial,
}: {
  actorUsername?: string
  tenants: TenantKeyHealth[]
}) {
  const [tenants, setTenants] = useState(initial)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const t = useDictionary(systemKeyHealthDictionary)

  async function refresh() {
    setBusy(true)
    setError('')
    try {
      setTenants(await listTenantKeyHealth())
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.fetchFailedError)
    } finally {
      setBusy(false)
    }
  }

  return (
    <SystemShell
      active="key-health"
      actorUsername={actorUsername}
      title={t.pageTitle}
      description={t.pageDescription}
      actions={
        <Button
          variant="outline"
          className="size-9 px-0"
          aria-label={t.reloadAriaLabel}
          onClick={refresh}
          disabled={busy}
        >
          <IconRefresh size={16} aria-hidden="true" />
        </Button>
      }
    >
      {error ? <Alert variant="destructive">{error}</Alert> : null}

      <KeyHealthTable tenants={tenants} />
    </SystemShell>
  )
}

export function KeyHealthTable({ tenants }: { tenants: TenantKeyHealth[] }) {
  const t = useDictionary(systemKeyHealthDictionary)
  return (
    <Card className="overflow-hidden">
      <table className="w-full text-sm">
        <thead className="bg-slate-50 text-left text-xs font-semibold uppercase tracking-wide text-slate-500">
          <tr>
            <th className="px-4 py-3">{t.tableHeaderTenant}</th>
            <th className="px-4 py-3">{t.tableHeaderProvider}</th>
            <th className="px-4 py-3">{t.tableHeaderActiveKid}</th>
            <th className="px-4 py-3">{t.tableHeaderKeyCount}</th>
            <th className="px-4 py-3">{t.tableHeaderProviderStatus}</th>
          </tr>
        </thead>
        <tbody>
          {tenants.map((tenant) => (
            <tr key={tenant.tenant_id} className="border-t border-slate-100">
              <td className="px-4 py-3 font-medium text-slate-800">{tenant.tenant_id}</td>
              <td className="px-4 py-3 text-xs">{tenant.provider}</td>
              <td className="px-4 py-3 font-mono text-xs">{tenant.active_kid || '—'}</td>
              <td className="px-4 py-3 text-slate-600">{tenant.jwks_key_count}</td>
              <td className="px-4 py-3">
                {tenant.provider_healthy ? (
                  <span className="inline-flex items-center gap-1 rounded-md bg-emerald-50 px-2 py-0.5 text-xs font-semibold text-emerald-700">
                    <IconShieldCheck size={13} aria-hidden="true" />
                    {t.healthy}
                  </span>
                ) : (
                  <span className="rounded-md bg-red-50 px-2 py-0.5 text-xs font-semibold text-red-700">
                    {t.unreachable}
                  </span>
                )}
              </td>
            </tr>
          ))}
          {tenants.length === 0 ? (
            <tr>
              <td colSpan={5} className="px-4 py-6 text-center text-sm text-slate-500">
                {t.noTenantsNotice}
              </td>
            </tr>
          ) : null}
        </tbody>
      </table>
    </Card>
  )
}
