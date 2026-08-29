import { IconArrowLeft, IconCheck, IconClock, IconDownload, IconUpload } from '@tabler/icons-react'
import { type ChangeEvent, useState } from 'react'
import {
  applyAdminGroupImport,
  AuthenticationAPIError,
  getAdminGroupImport,
  previewAdminGroups,
  tenantURL,
} from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Label } from '../../components/ui/label'
import { useDictionary } from '../../lib/i18n'
import type { GroupImportResult } from '../../types'
import {
  ApplyGroupImportConfirmDialog,
  GroupImportResultSummary,
  groupImportRowErrorMessage,
  type GroupImportResultView,
} from './AdminGroupImportResult'
import { adminGroupsDictionary } from './AdminGroupsPage.i18n'

// テンプレートは export と同じ import 互換列を、同じ並びで持つ。lifecycle_action は
// 常に空で出力されるため、テンプレートでも空のままにする。
const GROUP_IMPORT_CSV_TEMPLATE =
  'id,name,description,membership_type,roles,dynamic_rule_expression,dynamic_rule_enabled,lifecycle_action,created_at,updated_at\n'
const GROUP_IMPORT_POLL_INTERVAL_MS = 1000
const GROUP_IMPORT_POLL_MAX_ATTEMPTS = 30

class GroupImportTimeoutError extends Error {}
class GroupImportJobFailedError extends Error {}

const EMPTY_IMPORT_RESULT: GroupImportResult = {
  total_rows: 0,
  created_rows: 0,
  updated_rows: 0,
  unchanged_rows: 0,
  deleted_rows: 0,
  deleted_memberships: 0,
  rejected_rows: 0,
  error_total: 0,
}

async function pollGroupImportJob(jobId: string): Promise<GroupImportResultView> {
  for (let attempt = 0; attempt < GROUP_IMPORT_POLL_MAX_ATTEMPTS; attempt++) {
    const page = await getAdminGroupImport(jobId)
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
      throw new GroupImportJobFailedError()
    }
    await new Promise((resolve) => setTimeout(resolve, GROUP_IMPORT_POLL_INTERVAL_MS))
  }
  throw new GroupImportTimeoutError()
}

function importSubmitErrorMessage(t: typeof adminGroupsDictionary.ja, cause: unknown): string {
  if (cause instanceof GroupImportTimeoutError) return t.importTimeoutError
  if (cause instanceof GroupImportJobFailedError) return t.importJobFailedError
  if (cause instanceof AuthenticationAPIError) {
    if (cause.code) {
      const mapped = groupImportRowErrorMessage(t, cause.code)
      if (mapped !== cause.code) return mapped
    }
    return cause.message
  }
  return t.genericActionError
}

type GroupImportStep =
  | 'select'
  | 'preview_running'
  | 'preview_result'
  | 'apply_running'
  | 'apply_result'

// CSV は preview へ一度だけ送信し、apply は成功済み preview ID のみを参照する。
export function AdminGroupImportPage({
  csrfToken,
  actorUsername,
}: {
  csrfToken: string
  actorUsername?: string
}) {
  const listPath = tenantURL('/admin/groups')
  const t = useDictionary(adminGroupsDictionary)
  const [step, setStep] = useState<GroupImportStep>('select')
  const [fileName, setFileName] = useState('')
  const [selectedFile, setSelectedFile] = useState<File | null>(null)
  const [previewJobId, setPreviewJobId] = useState('')
  const [previewResult, setPreviewResult] = useState<GroupImportResultView | null>(null)
  const [applyResult, setApplyResult] = useState<GroupImportResultView | null>(null)
  const [showApplyConfirm, setShowApplyConfirm] = useState(false)
  const [deletionAcknowledged, setDeletionAcknowledged] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  function downloadTemplate() {
    const blob = new Blob([GROUP_IMPORT_CSV_TEMPLATE], { type: 'text/csv;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const anchor = document.createElement('a')
    anchor.href = url
    anchor.download = 'group-import-template.csv'
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
      const job = await previewAdminGroups(csrfToken, selectedFile)
      setPreviewJobId(job.id)
      setPreviewResult(await pollGroupImportJob(job.id))
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
      const job = await applyAdminGroupImport(csrfToken, previewJobId)
      setApplyResult(await pollGroupImportJob(job.id))
      setStep('apply_result')
    } catch (cause) {
      setError(importSubmitErrorMessage(t, cause))
      setStep('preview_result')
    } finally {
      setBusy(false)
      setDeletionAcknowledged(false)
    }
  }

  function reset() {
    setStep('select')
    setFileName('')
    setSelectedFile(null)
    setPreviewJobId('')
    setPreviewResult(null)
    setApplyResult(null)
    setDeletionAcknowledged(false)
    setError('')
  }

  async function loadErrorPage(result: GroupImportResultView, cursor: string) {
    setBusy(true)
    setError('')
    try {
      const page = await getAdminGroupImport(result.jobId, cursor)
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
      (previewResult?.unchanged_rows ?? 0) +
      (previewResult?.deleted_rows ?? 0) >
    0

  return (
    <AdminShell
      active="groups"
      actorUsername={actorUsername}
      title={t.importGroups}
      description={t.importGroupsDescription}
    >
      <div className="flex items-center gap-3">
        <a
          href={listPath}
          className="inline-flex size-9 items-center justify-center rounded-lg border border-slate-200 bg-white text-slate-700 transition hover:bg-slate-50 hover:text-slate-900"
          aria-label={t.backToGroupListAria}
        >
          <IconArrowLeft size={18} aria-hidden="true" />
        </a>
        <h1 className="text-2xl font-bold tracking-tight text-slate-900">{t.importGroups}</h1>
      </div>

      <div className="mt-6 max-w-3xl">
        {error && <Alert className="mb-4">{error}</Alert>}

        <Card className="shadow-[0_1px_2px_rgb(15_23_42/4%)]">
          <div className="grid gap-6 p-6">
            <div className="rounded-xl border border-slate-200 bg-slate-50 p-4 text-sm leading-6 text-slate-700">
              <p>{t.importInstructions}</p>
              <p className="mt-1">{t.importImmutableFieldNotice}</p>
              <p className="mt-1 font-semibold text-amber-800">{t.importDeleteNotice}</p>
              <p className="mt-2 text-xs text-slate-600">{t.importTransferPolicyNotice}</p>
              <p className="text-xs text-slate-600">{t.importSplitNotice}</p>
              <Button type="button" variant="outline" className="mt-3" onClick={downloadTemplate}>
                <IconDownload size={16} aria-hidden="true" />
                {t.downloadTemplate}
              </Button>
            </div>

            <div className="grid gap-2">
              <Label htmlFor="group-import-csv-file">{t.selectCsvFile}</Label>
              <input
                id="group-import-csv-file"
                type="file"
                accept=".csv,text/csv"
                onChange={handleFileChange}
                disabled={busy}
                aria-label={t.selectCsvFile}
                className="sr-only"
              />
              <label
                htmlFor="group-import-csv-file"
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
              <GroupImportResultSummary
                t={t}
                title={t.previewResultTitle}
                result={previewResult}
                busy={busy}
                onErrorPage={(cursor) => void loadErrorPage(previewResult, cursor)}
              />
            )}

            {applyResult && step === 'apply_result' && (
              <GroupImportResultSummary
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
              {step === 'apply_result' ? t.backToGroupList : t.cancel}
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
        <ApplyGroupImportConfirmDialog
          result={previewResult}
          busy={busy}
          acknowledged={deletionAcknowledged}
          onAcknowledgedChange={setDeletionAcknowledged}
          onClose={() => {
            setShowApplyConfirm(false)
            setDeletionAcknowledged(false)
          }}
          onConfirm={() => void runApply()}
        />
      )}
    </AdminShell>
  )
}
