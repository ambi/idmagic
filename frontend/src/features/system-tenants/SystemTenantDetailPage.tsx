import { IconArrowLeft, IconPencil, IconRoute } from '@tabler/icons-react'
import { type ReactNode, useEffect, useState } from 'react'
import { useRouter } from '@tanstack/react-router'
import { SystemShell } from '../../components/SystemShell'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Toast } from '../../components/ui/toast'
import { useDictionary, useLocale } from '../../lib/i18n'
import type { AdminTenant } from '../../types'
import { systemTenantsDictionary } from './SystemTenantsPage.i18n'
import {
  formatDate,
  QUOTA_KEYS,
  QUOTA_LABELS,
  StatusBadge,
  SYSTEM_TENANTS_PATH,
  clearTenantReturnNotice,
  readTenantReturnNotice,
  TenantStatusButton,
  TenantStatusDialog,
  tenantPath,
  type TenantSubpage,
} from './SystemTenantsShared'

// SystemTenantDetailPage はテナントの現在の値を読み取り専用で示し、
// 変更は名前の付いた入口から専用の画面へ移す（REQ-FRONTEND-003）。
export function SystemTenantDetailPage({
  csrfToken,
  actorUsername,
  tenant,
}: {
  csrfToken: string
  actorUsername?: string
  tenant: AdminTenant
}) {
  const t = useDictionary(systemTenantsDictionary)
  const { locale } = useLocale()
  const router = useRouter()
  const [confirmStatus, setConfirmStatus] = useState(false)
  const [notice, setNotice] = useState(() => {
    const returned = readTenantReturnNotice()
    if (returned === 'created') return t.tenantCreatedNotice
    if (returned === 'updated') return t.tenantUpdatedNotice
    return ''
  })
  useEffect(() => clearTenantReturnNotice(), [])
  const policy = tenant.password_policy_override

  return (
    <SystemShell
      active="tenants"
      actorUsername={actorUsername}
      title={tenant.display_name}
      description={tenant.realm}
      actions={
        <>
          <Button variant="outline" nativeButton={false} render={<a href={SYSTEM_TENANTS_PATH} />}>
            <IconArrowLeft size={16} aria-hidden="true" />
            {t.backToList}
          </Button>
          {tenant.realm !== 'default' ? (
            <TenantStatusButton
              tenant={tenant}
              busy={confirmStatus}
              onClick={() => setConfirmStatus(true)}
            />
          ) : null}
        </>
      }
    >
      <Toast message={notice} onDismiss={() => setNotice('')} />

      <div className="grid max-w-4xl gap-6">
        <DetailSection
          title={t.detailsHeading}
          action={<EditLink realm={tenant.realm} subpage="edit" label={t.editSettings} />}
        >
          <dl className="grid grid-cols-[minmax(0,180px)_minmax(0,1fr)] gap-x-4 gap-y-3 text-sm">
            <dt className="text-slate-500">Realm</dt>
            <dd className="font-mono text-xs">{tenant.realm}</dd>
            <dt className="text-slate-500">{t.idLabel}</dt>
            <dd className="break-all font-mono text-xs">{tenant.id}</dd>
            <dt className="text-slate-500">{t.displayNameLabel}</dt>
            <dd>{tenant.display_name}</dd>
            <dt className="text-slate-500">{t.tableHeaderStatus}</dt>
            <dd>
              <StatusBadge status={tenant.status} />
            </dd>
            <dt className="text-slate-500">{t.createdLabel}</dt>
            <dd>{formatDate(tenant.created_at, locale)}</dd>
            <dt className="text-slate-500">{t.updatedLabel}</dt>
            <dd>{formatDate(tenant.updated_at, locale)}</dd>
            <dt className="text-slate-500">{t.disabledLabel}</dt>
            <dd>{formatDate(tenant.disabled_at, locale)}</dd>
            <dt className="text-slate-500">{t.passwordPolicyOverrideHeading}</dt>
            <dd>
              {policy && Object.keys(policy).length > 0 ? (
                <span className="font-mono text-xs">
                  {t.minLabel} {policy.min_length ?? '—'} / {t.maxLabel} {policy.max_length ?? '—'}{' '}
                  / {t.historyLabel} {policy.history_depth ?? '—'}
                </span>
              ) : (
                t.inheritedValue
              )}
            </dd>
          </dl>
        </DetailSection>

        <DetailSection
          title={t.endpointStyleLabel}
          action={
            <EditLink realm={tenant.realm} subpage="endpoint-style" label={t.switchEndpointStyle} />
          }
        >
          <p className="font-mono text-sm">{tenant.endpoint_style ?? 'path'}</p>
        </DetailSection>

        <DetailSection
          title={t.quotaHeading}
          action={<EditLink realm={tenant.realm} subpage="quota" label={t.editQuota} />}
        >
          {tenant.usage || tenant.quota ? (
            <table className="w-full text-sm">
              <thead className="text-left text-xs font-semibold uppercase tracking-wide text-slate-500">
                <tr>
                  <th className="py-2">{t.quotaResourceHeader}</th>
                  <th className="py-2 text-right">{t.quotaUsageHeader}</th>
                  <th className="py-2 text-right">{t.quotaLimitHeader}</th>
                </tr>
              </thead>
              <tbody>
                {QUOTA_KEYS.map((key) => (
                  <tr key={key} className="border-t border-slate-100">
                    <td className="py-2">{t[QUOTA_LABELS[key]]}</td>
                    <td className="py-2 text-right font-mono text-xs">
                      {tenant.usage?.[key] ?? '—'}
                    </td>
                    <td className="py-2 text-right font-mono text-xs">
                      {tenant.quota?.[key] ?? t.inheritedValue}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          ) : (
            <p className="text-sm text-slate-500">{t.quotaUnavailable}</p>
          )}
        </DetailSection>
      </div>

      {confirmStatus ? (
        <TenantStatusDialog
          tenant={tenant}
          csrfToken={csrfToken}
          onClose={() => setConfirmStatus(false)}
          onChanged={async (success) => {
            await router.invalidate()
            setConfirmStatus(false)
            setNotice(success)
          }}
        />
      ) : null}
    </SystemShell>
  )
}

function DetailSection({
  title,
  action,
  children,
}: {
  title: string
  action: ReactNode
  children: ReactNode
}) {
  return (
    <Card className="p-5">
      <div className="mb-4 flex flex-wrap items-center justify-between gap-2">
        <h2 className="text-sm font-semibold text-slate-700">{title}</h2>
        {action}
      </div>
      {children}
    </Card>
  )
}

function EditLink({
  realm,
  subpage,
  label,
}: {
  realm: string
  subpage: TenantSubpage
  label: string
}) {
  return (
    <Button variant="outline" nativeButton={false} render={<a href={tenantPath(realm, subpage)} />}>
      {subpage === 'endpoint-style' ? (
        <IconRoute size={16} aria-hidden="true" />
      ) : (
        <IconPencil size={16} aria-hidden="true" />
      )}
      {label}
    </Button>
  )
}
