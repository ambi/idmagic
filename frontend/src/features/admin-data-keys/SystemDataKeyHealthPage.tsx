import { IconRefresh, IconShieldCheck } from '@tabler/icons-react'
import { useState } from 'react'
import { AuthenticationAPIError, listTenantDataKeyHealth } from '../../api'
import { SystemShell } from '../../components/SystemShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { useDictionary, useFormatters } from '../../lib/i18n'
import type { TenantDataKeyHealth } from '../../types'
import { systemDataKeyHealthDictionary } from './SystemDataKeyHealthPage.i18n'

export function SystemDataKeyHealthPage({
  actorUsername,
  tenants: initial,
}: {
  actorUsername?: string
  tenants: TenantDataKeyHealth[]
}) {
  const [tenants, setTenants] = useState(initial)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const t = useDictionary(systemDataKeyHealthDictionary)

  async function refresh() {
    setBusy(true)
    setError('')
    try {
      setTenants(await listTenantDataKeyHealth())
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.fetchFailedError)
    } finally {
      setBusy(false)
    }
  }

  return (
    <SystemShell
      active="data-key-health"
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

      <DataKeyHealthTable tenants={tenants} />
    </SystemShell>
  )
}

const statusLabelKeys = {
  active: 'statusActive',
  retiring: 'statusRetiring',
  disabled: 'statusDisabled',
  destroyed: 'statusDestroyed',
} as const

export function DataKeyHealthTable({ tenants }: { tenants: TenantDataKeyHealth[] }) {
  const t = useDictionary(systemDataKeyHealthDictionary)
  const { formatDateTime } = useFormatters()
  return (
    <Card className="overflow-hidden">
      <table className="w-full text-sm">
        <thead className="bg-slate-50 text-left text-xs font-semibold uppercase tracking-wide text-slate-500">
          <tr>
            <th className="px-4 py-3">{t.tableHeaderTenant}</th>
            <th className="px-4 py-3">{t.tableHeaderActiveVersion}</th>
            <th className="px-4 py-3">{t.tableHeaderStatus}</th>
            <th className="px-4 py-3">{t.tableHeaderProvider}</th>
            <th className="px-4 py-3">{t.tableHeaderProviderStatus}</th>
            <th className="px-4 py-3">{t.tableHeaderRotatedAt}</th>
          </tr>
        </thead>
        <tbody>
          {tenants.map((tenant) => (
            <tr key={tenant.tenant_id} className="border-t border-slate-100">
              <td className="px-4 py-3 font-medium text-slate-800">{tenant.tenant_id}</td>
              <td className="px-4 py-3 font-mono text-xs">{tenant.active_version}</td>
              <td className="px-4 py-3 text-xs">
                {
                  t[
                    statusLabelKeys[tenant.status as keyof typeof statusLabelKeys] ?? 'statusActive'
                  ]
                }
              </td>
              <td className="px-4 py-3 text-xs">{tenant.provider}</td>
              <td className="px-4 py-3">
                {tenant.provider_reachable ? (
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
              <td className="px-4 py-3 text-slate-600">
                {tenant.rotated_at ? formatDateTime(tenant.rotated_at) : '—'}
              </td>
            </tr>
          ))}
          {tenants.length === 0 ? (
            <tr>
              <td colSpan={6} className="px-4 py-6 text-center text-sm text-slate-500">
                {t.noTenantsNotice}
              </td>
            </tr>
          ) : null}
        </tbody>
      </table>
    </Card>
  )
}
