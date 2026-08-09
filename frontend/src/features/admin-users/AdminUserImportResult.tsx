import { IconAlertTriangle, IconCheck, IconX } from '@tabler/icons-react'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { useDictionary } from '../../lib/i18n'
import type { UserImportResult, UserImportRowError } from '../../types'
import { adminUsersDictionary } from './AdminUsersPage.i18n'

export type UserImportResultView = UserImportResult & {
  errors: UserImportRowError[]
  jobId: string
  previousCursor: string | null
  nextCursor: string | null
  currentPage: number
  totalPages: number
}

// Stable codes are localized; unknown codes remain visible as returned by the backend.
export function importRowErrorMessage(t: typeof adminUsersDictionary.ja, code: string): string {
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
    case 'duplicate_username':
      return t.importErrorDuplicateUsername
    case 'invalid_email':
      return t.importErrorInvalidEmail
    case 'username_conflict':
      return t.importErrorUsernameConflict
    case 'source_managed':
      return t.importErrorSourceManaged
    case 'identifier_mismatch':
      return t.importErrorIdentifierMismatch
    case 'target_not_found':
      return t.importErrorTargetNotFound
    case 'apply_failed':
      return t.importErrorApplyFailed
    case 'invalid_user':
      return t.importErrorInvalidUser
    default:
      return code
  }
}

function importColumnLabel(t: typeof adminUsersDictionary.ja, column: string | undefined): string {
  switch (column) {
    case 'preferred_username':
      return t.username
    case 'email':
      return t.emailFieldLabel
    case 'name':
      return t.displayName
    case 'roles':
      return t.rolesHeading
    case 'required_actions':
      return t.requiredActionsHeading
    default:
      return column ?? ''
  }
}

export function UserImportResultSummary({
  t,
  title,
  result,
  success = false,
  busy = false,
  onErrorPage,
}: {
  t: typeof adminUsersDictionary.ja
  title: string
  result: UserImportResultView
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
      {success && (
        <Alert variant="success">
          {t.importApplySuccessNotice.replace(
            '{count}',
            String(result.created_rows + result.updated_rows),
          )}
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
                    {importColumnLabel(t, rowError.column)}
                  </td>
                  <td className="px-4 py-2">{importRowErrorMessage(t, rowError.code)}</td>
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

export function ApplyImportConfirmDialog({
  result,
  busy,
  onClose,
  onConfirm,
}: {
  result: UserImportResultView
  busy: boolean
  onClose: () => void
  onConfirm: () => void
}) {
  const t = useDictionary(adminUsersDictionary)
  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/35 p-5 backdrop-blur-[2px]"
      role="dialog"
      aria-modal="true"
      aria-labelledby="apply-import-title"
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
              <h2 id="apply-import-title" className="text-xl font-semibold">
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
        <div className="flex justify-end gap-2 border-t border-slate-200 bg-slate-50 px-6 py-4">
          <Button type="button" variant="outline" onClick={onClose} disabled={busy}>
            {t.cancel}
          </Button>
          <Button type="button" onClick={onConfirm} disabled={busy}>
            <IconCheck size={16} aria-hidden="true" />
            {t.applyImportConfirmButton}
          </Button>
        </div>
      </Card>
    </div>
  )
}
