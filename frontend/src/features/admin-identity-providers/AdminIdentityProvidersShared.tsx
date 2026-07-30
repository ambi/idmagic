import { IconAlertTriangle, IconCircleCheck, IconX } from '@tabler/icons-react'
import type { IdentityProviderConnection, IdentityProviderConnectionTestResult } from '../../api'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { useDictionary } from '../../lib/i18n'
import { cn } from '../../lib/utils'
import { identityProvidersDictionary } from './AdminIdentityProvidersPage.i18n'

const STATUS_BADGE_STYLE: Record<'active' | 'disabled', { badge: string; dot: string }> = {
  active: { badge: 'bg-emerald-50 text-emerald-700', dot: 'bg-emerald-500' },
  disabled: { badge: 'bg-slate-100 text-slate-600', dot: 'bg-slate-400' },
}

export function StatusBadge({
  status,
  labels,
}: {
  status: 'active' | 'disabled'
  labels: { statusActive: string; statusDisabled: string }
}) {
  const style = STATUS_BADGE_STYLE[status]
  const label = status === 'active' ? labels.statusActive : labels.statusDisabled
  return (
    <span
      className={cn(
        'inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-xs font-semibold',
        style.badge,
      )}
    >
      <span className={cn('size-1.5 rounded-full', style.dot)} />
      {label}
    </span>
  )
}

// TestResultBanner はテスト実行結果を成功/失敗が明確に分かるバナーとして表示する。
// 従来の「常に同じ汎用トーストを出すだけ」の挙動を置き換える (wi-309)。
export function TestResultBanner({
  result,
  onDismiss,
}: {
  result: IdentityProviderConnectionTestResult
  onDismiss: () => void
}) {
  const t = useDictionary(identityProvidersDictionary)
  return (
    <Card
      className={cn(
        'flex items-start gap-3 border p-4',
        result.success ? 'border-emerald-200 bg-emerald-50' : 'border-red-200 bg-red-50',
      )}
    >
      {result.success ? (
        <IconCircleCheck
          size={20}
          className="mt-0.5 shrink-0 text-emerald-600"
          aria-hidden="true"
        />
      ) : (
        <IconAlertTriangle size={20} className="mt-0.5 shrink-0 text-red-600" aria-hidden="true" />
      )}
      <div className="min-w-0 flex-1">
        <p className={cn('font-semibold', result.success ? 'text-emerald-800' : 'text-red-800')}>
          {t.testResultHeading}
        </p>
        {result.success ? (
          <p className="mt-1 text-sm text-emerald-700">{t.testResultSuccess}</p>
        ) : (
          <>
            <p className="mt-1 text-sm text-red-700">{t.testResultFailureHeading}</p>
            <ul className="mt-1.5 list-inside list-disc text-sm text-red-700">
              {result.failures.map((failure) => (
                <li key={failure}>{failure}</li>
              ))}
            </ul>
          </>
        )}
      </div>
      <button
        type="button"
        onClick={onDismiss}
        aria-label={t.cancel}
        className="shrink-0 rounded-md p-1 text-slate-400 hover:bg-black/5"
      >
        <IconX size={16} aria-hidden="true" />
      </button>
    </Card>
  )
}

// DeleteConnectionDialog は Active な接続を削除する前の確認ダイアログ (wi-309)。
// Disabled な接続 (旧 Draft 相当) はいつでも確認なしで削除できる ([[ADR-149]])。
export function DeleteConnectionDialog({
  connection,
  busy,
  onClose,
  onConfirm,
}: {
  connection: IdentityProviderConnection
  busy: boolean
  onClose: () => void
  onConfirm: () => void
}) {
  const t = useDictionary(identityProvidersDictionary)
  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/35 p-5 backdrop-blur-[2px]"
      role="dialog"
      aria-modal="true"
      aria-labelledby="delete-identity-provider-title"
    >
      <button
        type="button"
        className="absolute inset-0 cursor-default"
        aria-label={t.cancel}
        onClick={onClose}
      />
      <Card className="relative w-full max-w-md overflow-hidden shadow-2xl">
        <div className="border-b border-slate-200 px-6 py-5">
          <div className="flex gap-3">
            <span className="flex size-10 shrink-0 items-center justify-center rounded-full bg-red-50 text-red-600">
              <IconAlertTriangle size={20} aria-hidden="true" />
            </span>
            <div>
              <h2 id="delete-identity-provider-title" className="font-semibold text-slate-950">
                {t.deleteConfirmTitle}
              </h2>
              <p className="mt-1 text-sm text-slate-600">
                {connection.status === 'active'
                  ? t.deleteConfirmActiveBody
                  : t.deleteConfirmDisabledBody}
              </p>
            </div>
          </div>
        </div>
        <div className="flex justify-end gap-2 bg-slate-50 px-6 py-4">
          <Button type="button" variant="outline" disabled={busy} onClick={onClose}>
            {t.cancel}
          </Button>
          <Button type="button" variant="destructive" disabled={busy} onClick={onConfirm}>
            {t.deleteConfirmButton}
          </Button>
        </div>
      </Card>
    </div>
  )
}
