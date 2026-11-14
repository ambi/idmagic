import { IconBan, IconCheck } from '@tabler/icons-react'
import { useState } from 'react'
import { AuthenticationAPIError, setAdminTenantDisabled } from '../../api'
import { ConfirmActionDialog } from '../../components/ConfirmActionDialog'
import { StepUpCancelledError, useStepUpGuard } from '../../components/StepUpDialog'
import { Button } from '../../components/ui/button'
import { useDictionary } from '../../lib/i18n'
import type { AdminTenant } from '../../types'
import { systemTenantsDictionary } from './SystemTenantsPage.i18n'

export function StatusBadge({ status }: { status: AdminTenant['status'] }) {
  return status === 'active' ? (
    <span className="rounded-md bg-emerald-50 px-2 py-0.5 text-xs font-semibold text-emerald-700">
      active
    </span>
  ) : (
    <span className="rounded-md bg-rose-50 px-2 py-0.5 text-xs font-semibold text-rose-700">
      disabled
    </span>
  )
}

export function formatDate(value: string | undefined, locale: 'ja' | 'en'): string {
  if (!value) return '—'
  try {
    return new Date(value).toLocaleString(locale === 'ja' ? 'ja-JP' : 'en-US')
  } catch {
    return value
  }
}

// テナントの画面は管理 API と同じく realm で対象を指す。
// 作成画面を /system/tenants の下に置かないのは、realm "new" の詳細と経路が衝突するためである。
export const SYSTEM_TENANTS_PATH = '/system/tenants'
export const SYSTEM_TENANT_CREATE_PATH = '/system/new-tenant'

export type TenantSubpage = 'edit' | 'quota' | 'endpoint-style'

export function tenantPath(realm: string, subpage?: TenantSubpage): string {
  const base = `${SYSTEM_TENANTS_PATH}/${encodeURIComponent(realm)}`
  return subpage ? `${base}/${subpage}` : base
}

// 保存や作成の成功は、遷移先の詳細で一度だけ通知する。
export type TenantReturnNotice = 'created' | 'updated'

export function tenantDetailURL(realm: string, notice: TenantReturnNotice): string {
  return `${tenantPath(realm)}?notice=${notice}`
}

export function readTenantReturnNotice(): TenantReturnNotice | undefined {
  const notice = new URLSearchParams(window.location.search).get('notice')
  return notice === 'created' || notice === 'updated' ? notice : undefined
}

// 詳細が通知を読んだら URL から消し、再読み込みで同じ通知を出さない。
export function clearTenantReturnNotice() {
  const params = new URLSearchParams(window.location.search)
  if (!params.has('notice')) return
  params.delete('notice')
  const search = params.toString()
  window.history.replaceState(
    window.history.state,
    '',
    `${window.location.pathname}${search ? `?${search}` : ''}`,
  )
}

export const QUOTA_KEYS = [
  'users',
  'groups',
  'agents',
  'applications',
  'oauth2_clients',
  'active_sessions',
  'consents',
  'active_jobs',
  'ssf_streams',
  'audit_events_retained',
  'export_artifacts_bytes',
] as const

export type QuotaKey = (typeof QUOTA_KEYS)[number]

export const QUOTA_LABELS: Record<QuotaKey, keyof (typeof systemTenantsDictionary)['en']> = {
  users: 'quotaUsers',
  groups: 'quotaGroups',
  agents: 'quotaAgents',
  applications: 'quotaApplications',
  oauth2_clients: 'quotaOAuth2Clients',
  active_sessions: 'quotaActiveSessions',
  consents: 'quotaConsents',
  active_jobs: 'quotaActiveJobs',
  ssf_streams: 'quotaSsfStreams',
  audit_events_retained: 'quotaAuditEventsRetained',
  export_artifacts_bytes: 'quotaExportArtifactsBytes',
}

// TenantStatusDialog は無効化と再開を確認してから要求する。
// 一覧の行と詳細の両方が使い、確認を経ない経路を作らない（REQ-FRONTEND-004）。
export function TenantStatusDialog({
  tenant,
  csrfToken,
  onClose,
  onChanged,
}: {
  tenant: AdminTenant
  csrfToken: string
  onClose: () => void
  onChanged: (notice: string) => Promise<void> | void
}) {
  const t = useDictionary(systemTenantsDictionary)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  // 制御面の変更は直近の MFA の認証を求める (REQ-SYSTEM-033)。
  const { guard, dialog } = useStepUpGuard()
  const disabling = tenant.status === 'active'

  async function confirm() {
    setBusy(true)
    setError('')
    try {
      await guard(() => setAdminTenantDisabled(csrfToken, tenant.realm, disabling))
      await onChanged(disabling ? t.tenantDisabledNotice : t.tenantReenabledNotice)
    } catch (cause) {
      if (!(cause instanceof StepUpCancelledError)) {
        setError(
          cause instanceof AuthenticationAPIError ? cause.message : t.tenantActionFailedError,
        )
      }
      setBusy(false)
    }
  }

  return (
    <>
      <ConfirmActionDialog
        title={disabling ? t.confirmDisableTitle : t.confirmEnableTitle}
        target={`${tenant.display_name} (${tenant.realm})`}
        description={disabling ? t.confirmDisableDescription : t.confirmEnableDescription}
        confirmLabel={disabling ? t.confirmDisable : t.confirmEnable}
        busy={busy}
        error={error}
        onClose={onClose}
        onConfirm={() => void confirm()}
      />
      {dialog}
    </>
  )
}
export function TenantStatusButton({
  tenant,
  busy,
  onClick,
}: {
  tenant: AdminTenant
  busy: boolean
  onClick: () => void
}) {
  const t = useDictionary(systemTenantsDictionary)
  const disabling = tenant.status === 'active'
  return (
    <Button variant={disabling ? 'destructive' : 'outline'} disabled={busy} onClick={onClick}>
      {disabling ? (
        <IconBan size={14} aria-hidden="true" />
      ) : (
        <IconCheck size={14} aria-hidden="true" />
      )}
      {disabling ? t.disable : t.enable}
    </Button>
  )
}
