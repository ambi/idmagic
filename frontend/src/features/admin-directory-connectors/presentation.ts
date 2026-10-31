import { tenantURL } from '../../api'
import type { DirectoryConnector, IngestionRunDetail, IngestionRunSummary } from '../../types'
import type { adminDirectoryConnectorsDictionary } from './AdminDirectoryConnectorsPage.i18n'

export type Dictionary = (typeof adminDirectoryConnectorsDictionary)['en']

export function listURL(): string {
  return tenantURL('/admin/directory-connectors')
}

export function newURL(): string {
  return tenantURL('/admin/directory-connectors/new')
}

export function detailURL(connectorID: string): string {
  return tenantURL(`/admin/directory-connectors/${encodeURIComponent(connectorID)}`)
}

export function connectorStatusLabel(status: DirectoryConnector['status'], t: Dictionary): string {
  if (status === 'active') return t.statusActive
  if (status === 'disabled') return t.statusDisabled
  return t.statusAwaitingEnrollment
}

// canManage は再発行と無効化の操作を出すかを決める。無効化は終端であり、再発行は 409 で拒否される。
export function canManage(connector: DirectoryConnector): boolean {
  return connector.status !== 'disabled'
}

// enrollmentExpired は有効期限の時刻以降を期限切れとして表示する。登録の可否を決めるのは API である。
export function enrollmentExpired(connector: DirectoryConnector, now: Date): boolean {
  if (connector.status !== 'awaiting_enrollment' || !connector.enrollment_expires_at) return false
  return new Date(connector.enrollment_expires_at).getTime() <= now.getTime()
}

// hiddenFailureCount は API の上限 100 件で切られた失敗の数を求める。
export function hiddenFailureCount(run: IngestionRunDetail): number {
  return Math.max(0, run.counts.failed - (run.failures?.length ?? 0))
}

const RUN_STATUS_KEYS: Record<IngestionRunSummary['status'], keyof Dictionary> = {
  running: 'runStatusRunning',
  awaiting_approval: 'runStatusAwaitingApproval',
  applying: 'runStatusApplying',
  discarded: 'runStatusDiscarded',
  completed: 'runStatusCompleted',
  superseded: 'runStatusSuperseded',
}

export function runStatusLabel(status: IngestionRunSummary['status'], t: Dictionary): string {
  const key = RUN_STATUS_KEYS[status]
  return key ? t[key] : status
}

const ITEM_ERROR_KEYS: Record<string, keyof Dictionary> = {
  username_conflict: 'itemErrorUsernameConflict',
  email_conflict: 'itemErrorEmailConflict',
  attribute_conflict: 'itemErrorAttributeConflict',
  invalid_attribute: 'itemErrorInvalidAttribute',
  quota_exceeded: 'itemErrorQuotaExceeded',
  user_deleted: 'itemErrorUserDeleted',
}

// itemErrorLabel は仕様が増やしたエラーコードを隠さないよう、引けない値を生のまま返す。
export function itemErrorLabel(code: string, t: Dictionary): string {
  const key = ITEM_ERROR_KEYS[code]
  return key ? t[key] : code
}
