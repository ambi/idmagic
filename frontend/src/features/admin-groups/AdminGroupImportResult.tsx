import { IconAlertTriangle, IconCheck, IconX } from '@tabler/icons-react'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { useDictionary } from '../../lib/i18n'
import type { GroupImportResult, GroupImportRowError } from '../../types'
import { adminGroupsDictionary } from './AdminGroupsPage.i18n'

export type GroupImportResultView = GroupImportResult & {
  errors: GroupImportRowError[]
  jobId: string
  previousCursor: string | null
  nextCursor: string | null
  currentPage: number
  totalPages: number
}

// 安定コードだけを訳し、未知のコードはバックエンドが返したまま見せる。
export function groupImportRowErrorMessage(
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
      return t.importErrorInvalidHeader
    case 'invalid_csv':
      return t.importErrorInvalidCsv
    case 'invalid_column_count':
      return t.importErrorInvalidColumnCount
    case 'required':
      return t.importErrorRequired
    case 'missing_identifier':
      return t.importErrorMissingIdentifier
    case 'duplicate_target':
      return t.importErrorDuplicateTarget
    case 'duplicate_name':
      return t.importErrorDuplicateName
    case 'identifier_mismatch':
      return t.importErrorIdentifierMismatch
    case 'target_not_found':
      return t.importErrorTargetNotFound
    case 'source_managed':
      return t.importErrorSourceManaged
    case 'immutable_membership_type':
      return t.importErrorImmutableMembershipType
    case 'invalid_membership_type':
      return t.importErrorInvalidMembershipType
    case 'invalid_roles':
      return t.importErrorInvalidRoles
    case 'invalid_dynamic_rule':
      return t.importErrorInvalidDynamicRule
    case 'invalid_lifecycle_action':
      return t.importErrorInvalidLifecycleAction
    case 'conflicting_lifecycle_action':
      return t.importErrorConflictingLifecycleAction
    case 'invalid_group':
      return t.importErrorInvalidGroup
    case 'apply_failed':
      return t.importErrorApplyFailed
    default:
      return code
  }
}

function groupImportColumnLabel(
  t: typeof adminGroupsDictionary.ja,
  column: string | undefined,
): string {
  switch (column) {
    case 'name':
      return t.groupNameLabel
    case 'description':
      return t.descriptionLabel
    case 'roles':
      return t.rolesLabel
    default:
      return column ?? ''
  }
}

export function GroupImportResultSummary({
  t,
  title,
  result,
  success = false,
  busy = false,
  onErrorPage,
}: {
  t: typeof adminGroupsDictionary.ja
  title: string
  result: GroupImportResultView
  success?: boolean
  busy?: boolean
  onErrorPage?: (cursor: string) => void
}) {
  return (
    <div className="grid gap-3">
      <h2 className="text-sm font-semibold text-slate-900">{title}</h2>
      <div className="grid grid-cols-2 gap-3 sm:grid-cols-5">
        {[
          { label: t.importTotalRows, value: result.total_rows },
          { label: t.importCreatedRows, value: result.created_rows },
          { label: t.importUpdatedRows, value: result.updated_rows },
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

      {/* 削除は不可逆で cascade するため、他の操作と同じ並びに混ぜず独立して見せる。 */}
      <div className="rounded-xl border border-amber-200 bg-amber-50 p-3">
        <h3 className="text-xs font-semibold uppercase tracking-wide text-amber-800">
          {t.importDeletionHeading}
        </h3>
        <div className="mt-2 grid grid-cols-2 gap-3">
          {[
            { label: t.importDeletedRows, value: result.deleted_rows },
            { label: t.importDeletedMemberships, value: result.deleted_memberships },
          ].map((item) => (
            <div
              key={item.label}
              className="rounded-lg border border-amber-200 bg-white p-3 text-center"
            >
              <p className="text-xl font-semibold text-amber-900">{item.value}</p>
              <p className="text-xs text-amber-700">{item.label}</p>
            </div>
          ))}
        </div>
      </div>

      {success && (
        <Alert variant="success">
          {t.importApplySuccessNotice.replace(
            '{count}',
            String(result.created_rows + result.updated_rows),
          )}
        </Alert>
      )}
      {success && result.deleted_rows > 0 && (
        <Alert>
          {t.importApplyDeletedNotice
            .replace('{groups}', String(result.deleted_rows))
            .replace('{memberships}', String(result.deleted_memberships))}
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
                    {groupImportColumnLabel(t, rowError.column)}
                  </td>
                  <td className="px-4 py-2">{groupImportRowErrorMessage(t, rowError.code)}</td>
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

// 削除を含む適用は、件数を読んだうえで明示的に確認させる。CSV の 1 列が大量削除を
// 発火しうるため、確認は多層の防御のうちの 1 つである。
export function ApplyGroupImportConfirmDialog({
  result,
  busy,
  acknowledged,
  onAcknowledgedChange,
  onClose,
  onConfirm,
}: {
  result: GroupImportResultView
  busy: boolean
  acknowledged: boolean
  onAcknowledgedChange: (value: boolean) => void
  onClose: () => void
  onConfirm: () => void
}) {
  const t = useDictionary(adminGroupsDictionary)
  const deletes = result.deleted_rows > 0
  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/35 p-5 backdrop-blur-[2px]"
      role="dialog"
      aria-modal="true"
      aria-labelledby="apply-group-import-title"
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
              <h2 id="apply-group-import-title" className="text-xl font-semibold">
                {t.applyImportConfirmTitle}
              </h2>
              <p className="mt-1 text-sm text-slate-500">
                {t.applyImportConfirmDescription.replace(
                  '{count}',
                  String(result.created_rows + result.updated_rows),
                )}
              </p>
            </div>
          </div>
          <Button variant="ghost" className="px-2.5" onClick={onClose} aria-label={t.close}>
            <IconX size={18} aria-hidden="true" />
          </Button>
        </div>
        {deletes && (
          <div className="border-b border-amber-200 bg-amber-50 px-6 py-4">
            <p className="text-sm font-semibold text-amber-900">
              {t.applyImportConfirmDeleteWarning
                .replace('{groups}', String(result.deleted_rows))
                .replace('{memberships}', String(result.deleted_memberships))}
            </p>
            <label className="mt-3 flex items-center gap-2 text-sm text-amber-900">
              <input
                type="checkbox"
                checked={acknowledged}
                onChange={(event) => onAcknowledgedChange(event.target.checked)}
              />
              {t.applyImportConfirmDeleteAcknowledge}
            </label>
          </div>
        )}
        <div className="flex justify-end gap-2 border-t border-slate-200 bg-slate-50 px-6 py-4">
          <Button type="button" variant="outline" onClick={onClose} disabled={busy}>
            {t.cancel}
          </Button>
          <Button type="button" onClick={onConfirm} disabled={busy || (deletes && !acknowledged)}>
            <IconCheck size={16} aria-hidden="true" />
            {t.applyImportConfirmButton}
          </Button>
        </div>
      </Card>
    </div>
  )
}
