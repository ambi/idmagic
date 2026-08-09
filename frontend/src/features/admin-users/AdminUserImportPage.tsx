import { IconArrowLeft, IconCheck, IconClock, IconDownload, IconUpload } from '@tabler/icons-react'
import { type ChangeEvent, useState } from 'react'
import {
  applyAdminUserImport,
  AuthenticationAPIError,
  getAdminUserImport,
  previewAdminUsers,
  tenantURL,
} from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Label } from '../../components/ui/label'
import { useDictionary } from '../../lib/i18n'
import type { UserImportResult } from '../../types'
import { adminUsersDictionary } from './AdminUsersPage.i18n'
import {
  ApplyImportConfirmDialog,
  importRowErrorMessage,
  UserImportResultSummary,
  type UserImportResultView,
} from './AdminUserImportResult'

const USER_IMPORT_CSV_TEMPLATE =
  'id,preferred_username,name,given_name,family_name,email,email_verified,roles,required_actions,mfa_enrolled,status,created_at,updated_at\n'
const USER_IMPORT_POLL_INTERVAL_MS = 1000
const USER_IMPORT_POLL_MAX_ATTEMPTS = 30

class UserImportTimeoutError extends Error {}
class UserImportJobFailedError extends Error {}

const EMPTY_IMPORT_RESULT: UserImportResult = {
  total_rows: 0,
  created_rows: 0,
  updated_rows: 0,
  unchanged_rows: 0,
  rejected_rows: 0,
  error_total: 0,
}

async function pollUserImportJob(jobId: string): Promise<UserImportResultView> {
  for (let attempt = 0; attempt < USER_IMPORT_POLL_MAX_ATTEMPTS; attempt++) {
    const page = await getAdminUserImport(jobId)
    const job = page.body
    if (job.status === 'succeeded') {
      return {
        ...(job.result ?? EMPTY_IMPORT_RESULT),
        errors: job.errors,
        jobId,
        previousCursor: page.previousCursor,
        nextCursor: page.nextCursor,
        currentPage: page.currentPage,
        totalPages: page.totalPages,
      }
    }
    if (job.status === 'failed' || job.status === 'canceled') {
      throw new UserImportJobFailedError()
    }
    await new Promise((resolve) => setTimeout(resolve, USER_IMPORT_POLL_INTERVAL_MS))
  }
  throw new UserImportTimeoutError()
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

type UserImportStep =
  | 'select'
  | 'preview_running'
  | 'preview_result'
  | 'apply_running'
  | 'apply_result'

// CSV は preview へ一度だけ送信し、apply は成功済み preview ID のみを参照する。
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
  const [selectedFile, setSelectedFile] = useState<File | null>(null)
  const [previewJobId, setPreviewJobId] = useState('')
  const [previewResult, setPreviewResult] = useState<UserImportResultView | null>(null)
  const [applyResult, setApplyResult] = useState<UserImportResultView | null>(null)
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

  function handleFileChange(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0]
    event.target.value = ''
    if (!file) return
    setError('')
    setPreviewResult(null)
    setApplyResult(null)
    setStep('select')
    setSelectedFile(file)
    setFileName(file.name)
  }

  async function runPreview() {
    if (!selectedFile) return
    setBusy(true)
    setError('')
    setStep('preview_running')
    try {
      const job = await previewAdminUsers(csrfToken, selectedFile)
      setPreviewJobId(job.id)
      setPreviewResult(await pollUserImportJob(job.id))
      setStep('preview_result')
    } catch (cause) {
      setError(importSubmitErrorMessage(t, cause))
      setStep('select')
    } finally {
      setBusy(false)
    }
  }

  async function runApply() {
    if (!previewJobId) return
    setBusy(true)
    setError('')
    setShowApplyConfirm(false)
    setStep('apply_running')
    try {
      const job = await applyAdminUserImport(csrfToken, previewJobId)
      setApplyResult(await pollUserImportJob(job.id))
      setStep('apply_result')
    } catch (cause) {
      setError(importSubmitErrorMessage(t, cause))
      setStep('preview_result')
    } finally {
      setBusy(false)
    }
  }

  function reset() {
    setStep('select')
    setFileName('')
    setSelectedFile(null)
    setPreviewJobId('')
    setPreviewResult(null)
    setApplyResult(null)
    setError('')
  }

  async function loadErrorPage(result: UserImportResultView, cursor: string) {
    setBusy(true)
    setError('')
    try {
      const page = await getAdminUserImport(result.jobId, cursor)
      const next = {
        ...result,
        errors: page.body.errors,
        previousCursor: page.previousCursor,
        nextCursor: page.nextCursor,
        currentPage: page.currentPage,
        totalPages: page.totalPages,
      }
      if (result.jobId === previewJobId) setPreviewResult(next)
      else setApplyResult(next)
    } catch (cause) {
      setError(importSubmitErrorMessage(t, cause))
    } finally {
      setBusy(false)
    }
  }

  const canRunPreview = Boolean(selectedFile) && step === 'select' && !busy
  const canApply =
    (previewResult?.created_rows ?? 0) +
      (previewResult?.updated_rows ?? 0) +
      (previewResult?.unchanged_rows ?? 0) >
    0

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
              <p className="mt-2 text-xs text-slate-600">{t.importTransferPolicyNotice}</p>
              <p className="text-xs text-slate-600">{t.importSplitNotice}</p>
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
                onChange={handleFileChange}
                disabled={busy}
                aria-label={t.selectCsvFile}
                className="sr-only"
              />
              <label
                htmlFor="import-csv-file"
                className="inline-flex h-9 w-fit cursor-pointer items-center justify-center rounded-md bg-slate-950 px-3 text-sm font-semibold text-white hover:bg-slate-800"
              >
                {t.chooseCsvFile}
              </label>
              {fileName && (
                <p className="text-xs text-slate-500">
                  {t.selectedFileLabel.replace('{name}', fileName)}
                </p>
              )}
            </div>

            {(step === 'preview_running' || step === 'apply_running') && (
              <p className="flex items-center gap-2 text-sm text-slate-600">
                <IconClock size={16} className="animate-pulse" aria-hidden="true" />
                {step === 'preview_running' ? t.previewRunning : t.applyRunning}
              </p>
            )}

            {previewResult && step !== 'apply_running' && step !== 'apply_result' && (
              <UserImportResultSummary
                t={t}
                title={t.previewResultTitle}
                result={previewResult}
                busy={busy}
                onErrorPage={(cursor) => void loadErrorPage(previewResult, cursor)}
              />
            )}

            {applyResult && step === 'apply_result' && (
              <UserImportResultSummary
                t={t}
                title={t.applyResultTitle}
                result={applyResult}
                success
                busy={busy}
                onErrorPage={(cursor) => void loadErrorPage(applyResult, cursor)}
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
              <Button type="button" disabled={!canRunPreview} onClick={() => void runPreview()}>
                <IconUpload size={16} aria-hidden="true" />
                {t.runPreview}
              </Button>
            )}
            {step === 'preview_result' && (
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

      {showApplyConfirm && previewResult && (
        <ApplyImportConfirmDialog
          result={previewResult}
          busy={busy}
          onClose={() => setShowApplyConfirm(false)}
          onConfirm={() => void runApply()}
        />
      )}
    </AdminShell>
  )
}
