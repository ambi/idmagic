import { IconEye, IconPencil, IconPlus, IconRefresh } from '@tabler/icons-react'
import { useState } from 'react'
import { listAdminTenants } from '../../api'
import { SystemShell } from '../../components/SystemShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Toast } from '../../components/ui/toast'
import { useDictionary } from '../../lib/i18n'
import type { AdminTenant } from '../../types'
import { systemTenantsDictionary } from './SystemTenantsPage.i18n'
import {
  StatusBadge,
  SYSTEM_TENANT_CREATE_PATH,
  TenantStatusButton,
  TenantStatusDialog,
  tenantPath,
} from './SystemTenantsShared'

export { StatusBadge } from './SystemTenantsShared'

// SystemTenantsPage はテナントの一覧である。値の確認と変更は詳細と変更画面が担う。
export function SystemTenantsPage({
  csrfToken,
  actorUsername,
  tenants: initial,
}: {
  csrfToken: string
  actorUsername?: string
  tenants: AdminTenant[]
}) {
  const [tenants, setTenants] = useState(initial)
  const [statusTarget, setStatusTarget] = useState<AdminTenant | null>(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const t = useDictionary(systemTenantsDictionary)

  // 再取得に失敗した場合は、直前の一覧を残して失敗を示す（REQ-FRONTEND-001）。
  async function refresh(success: string) {
    setBusy(true)
    setError('')
    try {
      setTenants(await listAdminTenants())
      setNotice(success)
    } catch {
      setError(t.tenantListFailedError)
    } finally {
      setBusy(false)
    }
  }

  return (
    <SystemShell
      active="tenants"
      actorUsername={actorUsername}
      title={t.pageTitle}
      description={t.pageDescription}
      actions={
        <>
          <Button
            variant="outline"
            className="size-9 px-0"
            aria-label={t.reloadListAriaLabel}
            onClick={() => void refresh(t.listRefreshedNotice)}
            disabled={busy}
          >
            <IconRefresh size={16} aria-hidden="true" />
          </Button>
          <Button nativeButton={false} render={<a href={SYSTEM_TENANT_CREATE_PATH} />}>
            <IconPlus size={16} aria-hidden="true" />
            {t.newTenant}
          </Button>
        </>
      }
    >
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      <Toast message={notice} onDismiss={() => setNotice('')} />

      <TenantTable tenants={tenants} busy={busy} onChangeStatus={setStatusTarget} />

      {statusTarget ? (
        <TenantStatusDialog
          tenant={statusTarget}
          csrfToken={csrfToken}
          onClose={() => setStatusTarget(null)}
          onChanged={async (success) => {
            setStatusTarget(null)
            await refresh(success)
          }}
        />
      ) : null}
    </SystemShell>
  )
}

export function TenantTable({
  tenants,
  busy,
  onChangeStatus,
}: {
  tenants: AdminTenant[]
  busy: boolean
  onChangeStatus: (tenant: AdminTenant) => void
}) {
  const t = useDictionary(systemTenantsDictionary)
  if (tenants.length === 0) {
    return (
      <Card className="p-5">
        <p className="text-sm text-slate-500">{t.emptyTenantList}</p>
      </Card>
    )
  }
  return (
    <Card className="overflow-x-auto">
      <table className="w-full text-sm">
        <thead className="bg-slate-50 text-left text-xs font-semibold uppercase tracking-wide text-slate-500">
          <tr>
            <th className="px-4 py-3">Realm</th>
            <th className="px-4 py-3">{t.tableHeaderDisplayName}</th>
            <th className="px-4 py-3">{t.tableHeaderStatus}</th>
            <th className="px-4 py-3">
              <span className="sr-only">{t.tableHeaderActions}</span>
            </th>
          </tr>
        </thead>
        <tbody>
          {tenants.map((tenant) => (
            <tr key={tenant.id} className="border-t border-slate-100 hover:bg-slate-50">
              <td className="px-4 py-3 font-mono text-xs">
                <a className="text-blue-700 hover:underline" href={tenantPath(tenant.realm)}>
                  {tenant.realm}
                </a>
              </td>
              <td className="px-4 py-3">{tenant.display_name}</td>
              <td className="px-4 py-3">
                <StatusBadge status={tenant.status} />
              </td>
              <td className="px-4 py-3">
                <div className="flex justify-end gap-1">
                  <Button
                    variant="ghost"
                    nativeButton={false}
                    render={<a href={tenantPath(tenant.realm)} />}
                    aria-label={`${t.viewDetails}: ${tenant.realm}`}
                  >
                    <IconEye size={14} aria-hidden="true" />
                    {t.viewDetails}
                  </Button>
                  <Button
                    variant="ghost"
                    nativeButton={false}
                    render={<a href={tenantPath(tenant.realm, 'edit')} />}
                    aria-label={`${t.edit}: ${tenant.realm}`}
                  >
                    <IconPencil size={14} aria-hidden="true" />
                    {t.edit}
                  </Button>
                  {tenant.realm !== 'default' ? (
                    <TenantStatusButton
                      tenant={tenant}
                      busy={busy}
                      onClick={() => onChangeStatus(tenant)}
                    />
                  ) : null}
                </div>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </Card>
  )
}
