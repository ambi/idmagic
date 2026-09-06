import { IconAlertTriangle, IconCheck, IconX } from '@tabler/icons-react'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { useDictionary } from '../../lib/i18n'
import type { GroupMembershipImportResult, GroupMembershipImportRowError } from '../../types'
import { adminGroupsDictionary } from './AdminGroupsPage.i18n'

export type GroupMemberImportResultView = GroupMembershipImportResult & {
  errors: GroupMembershipImportRowError[]
  jobId: string
  previousCursor: string | null
  nextCursor: string | null
  currentPage: number
  totalPages: number
}

// 安定コードだけを訳し、未知のコードはバックエンドが返したまま見せる。
export function groupMemberImportRowErrorMessage(
  t: typeof adminGroupsDictionary.ja,
  code: string,
): string {
  switch (code) {
    case 'csv_too_large':
      return t.importErrorCsvTooLarge
    case 'too_many_rows':
      return t.importErrorTooManyRows
    case 'field_too_large':
      return t.importErrorFieldTooLarge
    case 'invalid_header':
      return t.membershipImportErrorMissingStateColumn
    case 'invalid_csv':
      return t.importErrorInvalidCsv
    case 'invalid_column_count':
      return t.importErrorInvalidColumnCount
    case 'invalid_membership_state':
      return t.membershipImportErrorInvalidMembershipState
    case 'missing_identifier':
      return t.membershipImportErrorMissingIdentifier
    case 'duplicate_target':
      return t.membershipImportErrorDuplicateTarget
    case 'identifier_mismatch':
      return t.membershipImportErrorIdentifierMismatch
    case 'target_not_found':
      return t.membershipImportErrorTargetNotFound
    case 'group_mismatch':
      return t.membershipImportErrorGroupMismatch
    case 'dynamic_group':
      return t.membershipImportErrorDynamicGroup
    case 'dynamic_membership':
      return t.membershipImportErrorDynamicMembership
    case 'source_managed':
      return t.membershipImportErrorSourceManaged
    case 'apply_failed':
      return t.importErrorApplyFailed
    default:
      return code
  }
}

function groupMemberImportColumnLabel(
  t: typeof adminGroupsDictionary.ja,
  column: string | undefined,
): string {
  switch (column) {
    case 'user_id':
      return t.membershipColumnUserId
    case 'preferred_username':
      return t.membershipColumnUsername
    case 'membership_state':
      return t.membershipColumnState
    case 'group_id':
      return t.membershipColumnGroupId
    case 'group_name':
      return t.membershipColumnGroupName
    default:
      return column ?? ''
  }
}

export function GroupMemberImportResultSummary({
  t,
  title,
  result,
  success = false,
  busy = false,
  onErrorPage,
}: {
  t: typeof adminGroupsDictionary.ja
  title: string
  result: GroupMemberImportResultView
  success?: boolean
  busy?: boolean
  onErrorPage?: (cursor: string) => void
}) {
  return (
    <div className="grid gap-3">
      <h2 className="text-sm font-semibold text-slate-900">{title}</h2>
      <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
        {[
          { label: t.importTotalRows, value: result.total_rows },
          { label: t.membershipImportAddedRows, value: result.added_rows },
          { label: t.importUnchangedRows, value: result.unchanged_rows },
          { label: t.importRejectedRows, value: result.rejected_rows },
        ].map((item) => (
          <div
            key={item.label}
            className="rounded-lg border border-slate-200 bg-white p-3 text-center"
          >
            <p className="text-xl font-semibold text-slate-900">{item.value}</p>
            <p className="text-xs text-slate-500">{item.label}</p>
          </div>
        ))}
      </div>

      {/* 解除は実効ロールを取り去るため、他の操作と同じ並びに混ぜず独立して見せる。 */}
      <div className="rounded-xl border border-amber-200 bg-amber-50 p-3">
        <h3 className="text-xs font-semibold uppercase tracking-wide text-amber-800">
          {t.membershipImportReleaseHeading}
        </h3>
        <div className="mt-2 rounded-lg border border-amber-200 bg-white p-3 text-center">
          <p className="text-xl font-semibold text-amber-900">{result.removed_rows}</p>
          <p className="text-xs text-amber-700">{t.membershipImportRemovedRows}</p>
        </div>
        <p className="mt-2 text-xs text-amber-800">{t.membershipImportReleaseNotice}</p>
      </div>

      {success && (
        <Alert variant="success">
          {t.membershipImportApplySuccessNotice
            .replace('{added}', String(result.added_rows))
            .replace('{removed}', String(result.removed_rows))}
        </Alert>
      )}
      {result.errors.length > 0 && (
        <div className="overflow-hidden rounded-xl border border-slate-200">
          <table className="w-full text-left text-sm">
            <thead className="bg-slate-50 text-xs font-semibold uppercase tracking-wide text-slate-500">
              <tr>
                <th className="px-4 py-2">{t.importRowColumnHeader}</th>
                <th className="px-4 py-2">{t.importFieldColumnHeader}</th>
                <th className="px-4 py-2">{t.importErrorColumnHeader}</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {result.errors.map((rowError) => (
                <tr key={`${rowError.row}-${rowError.column ?? ''}-${rowError.code}`}>
                  <td className="px-4 py-2 font-mono text-xs">{rowError.row}</td>
                  <td className="px-4 py-2 text-xs text-slate-600">
                    {groupMemberImportColumnLabel(t, rowError.column)}
                  </td>
                  <td className="px-4 py-2">
                    {groupMemberImportRowErrorMessage(t, rowError.code)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      {result.error_total > result.errors.length && (
        <div className="flex items-center justify-between text-xs text-slate-500">
          <span>
            {t.importErrorPageStatus
              .replace('{current}', String(result.currentPage))
              .replace('{total}', String(result.totalPages))}
          </span>
          <div className="flex gap-2">
            <Button
              type="button"
              variant="outline"
              disabled={!result.previousCursor || busy}
              onClick={() => result.previousCursor && onErrorPage?.(result.previousCursor)}
            >
              {t.previousPage}
            </Button>
            <Button
              type="button"
              variant="outline"
              disabled={!result.nextCursor || busy}
              onClick={() => result.nextCursor && onErrorPage?.(result.nextCursor)}
            >
              {t.nextPage}
            </Button>
          </div>
        </div>
      )}
    </div>
  )
}

// 解除を含む適用は、件数を読んだうえで明示的に確認させる。CSV の 1 列が大量の
// 権限剥奪を発火しうるため、確認は多層の防御のうちの 1 つである。
export function ApplyGroupMemberImportConfirmDialog({
  result,
  busy,
  acknowledged,
  onAcknowledgedChange,
  onClose,
  onConfirm,
}: {
  result: GroupMemberImportResultView
  busy: boolean
  acknowledged: boolean
  onAcknowledgedChange: (value: boolean) => void
  onClose: () => void
  onConfirm: () => void
}) {
  const t = useDictionary(adminGroupsDictionary)
  const releases = result.removed_rows > 0
  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/35 p-5 backdrop-blur-[2px]"
      role="dialog"
      aria-modal="true"
      aria-labelledby="apply-group-member-import-title"
    >
      <button
        type="button"
        className="absolute inset-0 cursor-default"
        aria-label={t.close}
        onClick={onClose}
      />
      <Card className="relative w-full max-w-lg overflow-hidden shadow-2xl">
        <div className="flex items-start justify-between border-b border-slate-200 px-6 py-5">
          <div className="flex gap-3">
            <span className="flex size-9 shrink-0 items-center justify-center rounded-full bg-amber-50 text-amber-700">
              <IconAlertTriangle size={18} aria-hidden="true" />
            </span>
            <div>
              <h2 id="apply-group-member-import-title" className="text-xl font-semibold">
                {t.membershipImportConfirmTitle}
              </h2>
              <p className="mt-1 text-sm text-slate-500">
                {t.membershipImportConfirmDescription.replace('{added}', String(result.added_rows))}
              </p>
            </div>
          </div>
          <Button variant="ghost" className="px-2.5" onClick={onClose} aria-label={t.close}>
            <IconX size={18} aria-hidden="true" />
          </Button>
        </div>
        {releases && (
          <div className="border-b border-amber-200 bg-amber-50 px-6 py-4">
            <p className="text-sm font-semibold text-amber-900">
              {t.membershipImportConfirmReleaseWarning.replace(
                '{removed}',
                String(result.removed_rows),
              )}
            </p>
            <label className="mt-3 flex items-center gap-2 text-sm text-amber-900">
              <input
                type="checkbox"
                checked={acknowledged}
                onChange={(event) => onAcknowledgedChange(event.target.checked)}
              />
              {t.membershipImportConfirmReleaseAcknowledge}
            </label>
          </div>
        )}
        <div className="flex justify-end gap-2 border-t border-slate-200 bg-slate-50 px-6 py-4">
          <Button type="button" variant="outline" onClick={onClose} disabled={busy}>
            {t.cancel}
          </Button>
          <Button type="button" onClick={onConfirm} disabled={busy || (releases && !acknowledged)}>
            <IconCheck size={16} aria-hidden="true" />
            {t.applyImportConfirmButton}
          </Button>
        </div>
      </Card>
    </div>
  )
}
