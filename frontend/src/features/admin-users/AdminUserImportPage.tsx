import {
  IconAlertTriangle,
  IconArrowLeft,
  IconCheck,
  IconClock,
  IconDownload,
  IconUpload,
  IconX,
} from '@tabler/icons-react'
import { type ChangeEvent, useState } from 'react'
import { AuthenticationAPIError, getAdminUserImport, importAdminUsers, tenantURL } from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Label } from '../../components/ui/label'
import { useDictionary } from '../../lib/i18n'
import type { UserImportResult, UserImportRowError } from '../../types'
import { adminUsersDictionary } from './AdminUsersPage.i18n'

const USER_IMPORT_CSV_TEMPLATE = 'preferred_username,email,name,roles\n'
const USER_IMPORT_POLL_INTERVAL_MS = 1000
const USER_IMPORT_POLL_MAX_ATTEMPTS = 30

class UserImportTimeoutError extends Error {}
class UserImportJobFailedError extends Error {}

async function pollUserImportJob(jobId: string): Promise<UserImportResult> {
  for (let attempt = 0; attempt < USER_IMPORT_POLL_MAX_ATTEMPTS; attempt++) {
    const job = await getAdminUserImport(jobId)
    if (job.status === 'succeeded') {
      return job.result ?? { total_rows: 0, accepted_rows: 0, rejected_rows: 0 }
    }
    if (job.status === 'failed' || job.status === 'canceled') {
      throw new UserImportJobFailedError()
    }
    await new Promise((resolve) => setTimeout(resolve, USER_IMPORT_POLL_INTERVAL_MS))
  }
  throw new UserImportTimeoutError()
}

// stable error code だけを既知の翻訳に対応付ける。未登録 code は backend の原文 (code 自体) を
// そのまま出す (errorMessage.ts と同じ方針)。
function importRowErrorMessage(t: typeof adminUsersDictionary.ja, code: string): string {
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
    default:
      return column ?? ''
  }
}

function importSubmitErrorMessage(t: typeof adminUsersDictionary.ja, cause: unknown): string {
  if (cause instanceof UserImportTimeoutError) return t.importTimeoutError
  if (cause instanceof UserImportJobFailedError) return t.importJobFailedError
  if (cause instanceof AuthenticationAPIError) {
    if (cause.code) {
      const mapped = importRowErrorMessage(t, cause.code)
      if (mapped !== cause.code) return mapped
    }
    return cause.message
  }
  return t.genericActionError
}

function UserImportResultSummary({
  t,
  title,
  result,
  success = false,
}: {
  t: typeof adminUsersDictionary.ja
  title: string
  result: UserImportResult
  success?: boolean
}) {
  return (
    <div className="grid gap-3">
      <h2 className="text-sm font-semibold text-slate-900">{title}</h2>
      <div className="grid grid-cols-3 gap-3">
        {[
          { label: t.importTotalRows, value: result.total_rows },
          { label: t.importAcceptedRows, value: result.accepted_rows },
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
          {t.importApplySuccessNotice.replace('{count}', String(result.accepted_rows))}
        </Alert>
      )}
      {result.errors && result.errors.length > 0 && (
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
              {result.errors.map((rowError: UserImportRowError) => (
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
    </div>
  )
}

function ApplyImportConfirmDialog({
  result,
  busy,
  onClose,
  onConfirm,
}: {
  result: UserImportResult
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
                {t.applyImportConfirmDescription.replace('{count}', String(result.accepted_rows))}
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

type UserImportStep =
  | 'select'
  | 'dry_run_running'
  | 'dry_run_result'
  | 'apply_running'
  | 'apply_result'

// AdminUserImportPage は CSV アップロード → dry-run 検証プレビュー → 明示確認 → apply の
// ウィザード (wi-202)。CSV は常にクライアント側で UTF-8 text として読み、apply は必ず
// dry-run と同じ内容を送る (差し替え防止のため再アップロードは求めない)。
export function AdminUserImportPage({
  csrfToken,
  actorUsername,
}: {
  csrfToken: string
  actorUsername?: string
}) {
  const listPath = tenantURL('/admin/users')
  const t = useDictionary(adminUsersDictionary)
  const [step, setStep] = useState<UserImportStep>('select')
  const [fileName, setFileName] = useState('')
  const [csvText, setCsvText] = useState('')
  const [dryRunResult, setDryRunResult] = useState<UserImportResult | null>(null)
  const [applyResult, setApplyResult] = useState<UserImportResult | null>(null)
  const [showApplyConfirm, setShowApplyConfirm] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  function downloadTemplate() {
    const blob = new Blob([USER_IMPORT_CSV_TEMPLATE], { type: 'text/csv;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const anchor = document.createElement('a')
    anchor.href = url
    anchor.download = 'user-import-template.csv'
    anchor.click()
    URL.revokeObjectURL(url)
  }

  async function handleFileChange(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0]
    event.target.value = ''
    if (!file) return
    setError('')
    setDryRunResult(null)
    setApplyResult(null)
    setStep('select')
    try {
      setCsvText(await file.text())
      setFileName(file.name)
    } catch {
      setError(t.importFileReadError)
    }
  }

  async function runDryRun() {
    if (!csvText) return
    setBusy(true)
    setError('')
    setStep('dry_run_running')
    try {
      const job = await importAdminUsers(csrfToken, { csv: csvText, mode: 'dry_run' })
      setDryRunResult(await pollUserImportJob(job.id))
      setStep('dry_run_result')
    } catch (cause) {
      setError(importSubmitErrorMessage(t, cause))
      setStep('select')
    } finally {
      setBusy(false)
    }
  }

  async function runApply() {
    if (!csvText) return
    setBusy(true)
    setError('')
    setShowApplyConfirm(false)
    setStep('apply_running')
    try {
      const job = await importAdminUsers(csrfToken, { csv: csvText, mode: 'apply' })
      setApplyResult(await pollUserImportJob(job.id))
      setStep('apply_result')
    } catch (cause) {
      setError(importSubmitErrorMessage(t, cause))
      setStep('dry_run_result')
    } finally {
      setBusy(false)
    }
  }

  function reset() {
    setStep('select')
    setFileName('')
    setCsvText('')
    setDryRunResult(null)
    setApplyResult(null)
    setError('')
  }

  const canRunDryRun = Boolean(csvText) && step === 'select' && !busy
  const canApply = (dryRunResult?.accepted_rows ?? 0) > 0

  return (
    <AdminShell
      active="users"
      actorUsername={actorUsername}
      title={t.importUsers}
      description={t.importUsersDescription}
    >
      <div className="flex items-center gap-3">
        <a
          href={listPath}
          className="inline-flex size-9 items-center justify-center rounded-lg border border-slate-200 bg-white text-slate-700 transition hover:bg-slate-50 hover:text-slate-900"
          aria-label={t.backToUserListAria}
        >
          <IconArrowLeft size={18} aria-hidden="true" />
        </a>
        <h1 className="text-2xl font-bold tracking-tight text-slate-900">{t.importUsers}</h1>
      </div>

      <div className="mt-6 max-w-3xl">
        {error && <Alert className="mb-4">{error}</Alert>}

        <Card className="shadow-[0_1px_2px_rgb(15_23_42/4%)]">
          <div className="grid gap-6 p-6">
            <div className="rounded-xl border border-slate-200 bg-slate-50 p-4 text-sm leading-6 text-slate-700">
              <p>{t.importInstructions}</p>
              <p className="mt-1 font-semibold text-slate-900">
                {t.importPasswordColumnRejectedNotice}
              </p>
              <Button type="button" variant="outline" className="mt-3" onClick={downloadTemplate}>
                <IconDownload size={16} aria-hidden="true" />
                {t.downloadTemplate}
              </Button>
            </div>

            <div className="grid gap-2">
              <Label htmlFor="import-csv-file">{t.selectCsvFile}</Label>
              <input
                id="import-csv-file"
                type="file"
                accept=".csv,text/csv"
                onChange={(event) => void handleFileChange(event)}
                disabled={busy}
                className="block w-full text-sm text-slate-700 file:mr-3 file:rounded-lg file:border-0 file:bg-slate-950 file:px-3 file:py-2 file:text-sm file:font-semibold file:text-white"
              />
              {fileName && (
                <p className="text-xs text-slate-500">
                  {t.selectedFileLabel.replace('{name}', fileName)}
                </p>
              )}
            </div>

            {(step === 'dry_run_running' || step === 'apply_running') && (
              <p className="flex items-center gap-2 text-sm text-slate-600">
                <IconClock size={16} className="animate-pulse" aria-hidden="true" />
                {step === 'dry_run_running' ? t.dryRunRunning : t.applyRunning}
              </p>
            )}

            {dryRunResult && step !== 'apply_running' && step !== 'apply_result' && (
              <UserImportResultSummary t={t} title={t.dryRunResultTitle} result={dryRunResult} />
            )}

            {applyResult && step === 'apply_result' && (
              <UserImportResultSummary
                t={t}
                title={t.applyResultTitle}
                result={applyResult}
                success
              />
            )}
          </div>

          <div className="flex justify-end gap-2 border-t border-slate-200 bg-slate-50 px-6 py-4">
            <a
              href={listPath}
              className="inline-flex h-9 items-center justify-center rounded-lg border border-slate-200 bg-white px-4 text-sm font-medium text-slate-700 shadow-sm transition hover:bg-slate-50 hover:text-slate-900"
            >
              {step === 'apply_result' ? t.backToUserList : t.cancel}
            </a>
            {step === 'select' && (
              <Button type="button" disabled={!canRunDryRun} onClick={() => void runDryRun()}>
                <IconUpload size={16} aria-hidden="true" />
                {t.runDryRun}
              </Button>
            )}
            {step === 'dry_run_result' && (
              <>
                <Button type="button" variant="outline" onClick={reset} disabled={busy}>
                  {t.startOver}
                </Button>
                <Button
                  type="button"
                  disabled={!canApply || busy}
                  onClick={() => setShowApplyConfirm(true)}
                >
                  <IconCheck size={16} aria-hidden="true" />
                  {t.applyImport}
                </Button>
              </>
            )}
          </div>
        </Card>
      </div>

      {showApplyConfirm && dryRunResult && (
        <ApplyImportConfirmDialog
          result={dryRunResult}
          busy={busy}
          onClose={() => setShowApplyConfirm(false)}
          onConfirm={() => void runApply()}
        />
      )}
    </AdminShell>
  )
}
