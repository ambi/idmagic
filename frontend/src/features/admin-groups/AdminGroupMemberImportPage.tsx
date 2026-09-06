import { IconArrowLeft, IconCheck, IconClock, IconDownload, IconUpload } from '@tabler/icons-react'
import { type ChangeEvent, useState } from 'react'
import {
  applyAdminGroupMemberImport,
  AuthenticationAPIError,
  getAdminGroupMemberImport,
  previewAdminGroupMembers,
  tenantURL,
} from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Label } from '../../components/ui/label'
import { useDictionary } from '../../lib/i18n'
import type { GroupMembershipImportResult } from '../../types'
import {
  ApplyGroupMemberImportConfirmDialog,
  GroupMemberImportResultSummary,
  groupMemberImportRowErrorMessage,
  type GroupMemberImportResultView,
} from './AdminGroupMemberImportResult'
import { adminGroupsDictionary } from './AdminGroupsPage.i18n'

// テンプレートは export と同じ import 互換列を、同じ並びで持つ。membership_state は
// export が常に present を書くため、テンプレートでも present を置く。
const MEMBER_IMPORT_CSV_TEMPLATE =
  'group_id,group_name,user_id,preferred_username,membership_state,source,created_at\n'
const MEMBER_IMPORT_POLL_INTERVAL_MS = 1000
const MEMBER_IMPORT_POLL_MAX_ATTEMPTS = 30

class MemberImportTimeoutError extends Error {}
class MemberImportJobFailedError extends Error {}

const EMPTY_MEMBER_IMPORT_RESULT: GroupMembershipImportResult = {
  group_id: '',
  total_rows: 0,
  added_rows: 0,
  removed_rows: 0,
  unchanged_rows: 0,
  rejected_rows: 0,
  error_total: 0,
}

async function pollMemberImportJob(
  groupId: string,
  jobId: string,
): Promise<GroupMemberImportResultView> {
  for (let attempt = 0; attempt < MEMBER_IMPORT_POLL_MAX_ATTEMPTS; attempt++) {
    const page = await getAdminGroupMemberImport(groupId, jobId)
    const job = page.body
    if (job.status === 'succeeded') {
      return {
        ...(job.result ?? EMPTY_MEMBER_IMPORT_RESULT),
        errors: job.errors,
        jobId,
        previousCursor: page.previousCursor,
        nextCursor: page.nextCursor,
        currentPage: page.currentPage,
        totalPages: page.totalPages,
      }
    }
    if (job.status === 'failed' || job.status === 'canceled') {
      throw new MemberImportJobFailedError()
    }
    await new Promise((resolve) => setTimeout(resolve, MEMBER_IMPORT_POLL_INTERVAL_MS))
  }
  throw new MemberImportTimeoutError()
}

function memberImportSubmitErrorMessage(
  t: typeof adminGroupsDictionary.ja,
  cause: unknown,
): string {
  if (cause instanceof MemberImportTimeoutError) return t.importTimeoutError
  if (cause instanceof MemberImportJobFailedError) return t.importJobFailedError
  if (cause instanceof AuthenticationAPIError) {
    if (cause.code) {
      const mapped = groupMemberImportRowErrorMessage(t, cause.code)
      if (mapped !== cause.code) return mapped
    }
    return cause.message
  }
  return t.genericActionError
}

type MemberImportStep =
  | 'select'
  | 'preview_running'
  | 'preview_result'
  | 'apply_running'
  | 'apply_result'

// CSV は preview へ一度だけ送信し、apply は成功済み preview ID のみを参照する。
// 対象グループは URL が決め、この画面はそれをそのまま運ぶ。
export function AdminGroupMemberImportPage({
  csrfToken,
  actorUsername,
  groupId,
}: {
  csrfToken: string
  actorUsername?: string
  groupId: string
}) {
  const detailPath = tenantURL(`/admin/groups/${encodeURIComponent(groupId)}`)
  const t = useDictionary(adminGroupsDictionary)
  const [step, setStep] = useState<MemberImportStep>('select')
  const [fileName, setFileName] = useState('')
  const [selectedFile, setSelectedFile] = useState<File | null>(null)
  const [previewJobId, setPreviewJobId] = useState('')
  const [previewResult, setPreviewResult] = useState<GroupMemberImportResultView | null>(null)
  const [applyResult, setApplyResult] = useState<GroupMemberImportResultView | null>(null)
  const [showApplyConfirm, setShowApplyConfirm] = useState(false)
  const [releaseAcknowledged, setReleaseAcknowledged] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  function downloadTemplate() {
    const blob = new Blob([MEMBER_IMPORT_CSV_TEMPLATE], { type: 'text/csv;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const anchor = document.createElement('a')
    anchor.href = url
    anchor.download = 'group-member-import-template.csv'
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
      const job = await previewAdminGroupMembers(csrfToken, groupId, selectedFile)
      setPreviewJobId(job.id)
      setPreviewResult(await pollMemberImportJob(groupId, job.id))
      setStep('preview_result')
    } catch (cause) {
      setError(memberImportSubmitErrorMessage(t, cause))
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
      const job = await applyAdminGroupMemberImport(csrfToken, groupId, previewJobId)
      setApplyResult(await pollMemberImportJob(groupId, job.id))
      setStep('apply_result')
    } catch (cause) {
      setError(memberImportSubmitErrorMessage(t, cause))
      setStep('preview_result')
    } finally {
      setBusy(false)
      setReleaseAcknowledged(false)
    }
  }

  function reset() {
    setStep('select')
    setFileName('')
    setSelectedFile(null)
    setPreviewJobId('')
    setPreviewResult(null)
    setApplyResult(null)
    setReleaseAcknowledged(false)
    setError('')
  }

  async function loadErrorPage(result: GroupMemberImportResultView, cursor: string) {
    setBusy(true)
    setError('')
    try {
      const page = await getAdminGroupMemberImport(groupId, result.jobId, cursor)
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
      setError(memberImportSubmitErrorMessage(t, cause))
    } finally {
      setBusy(false)
    }
  }

  const canRunPreview = Boolean(selectedFile) && step === 'select' && !busy
  const canApply =
    (previewResult?.added_rows ?? 0) +
      (previewResult?.removed_rows ?? 0) +
      (previewResult?.unchanged_rows ?? 0) >
    0

  return (
    <AdminShell
      active="groups"
      actorUsername={actorUsername}
      title={t.importMembersTitle}
      description={t.importMembersDescription}
    >
      <div className="flex items-center gap-3">
        <a
          href={detailPath}
          className="inline-flex size-9 items-center justify-center rounded-lg border border-slate-200 bg-white text-slate-700 transition hover:bg-slate-50 hover:text-slate-900"
          aria-label={t.backToGroupDetailAria}
        >
          <IconArrowLeft size={18} aria-hidden="true" />
        </a>
        <h1 className="text-2xl font-bold tracking-tight text-slate-900">{t.importMembersTitle}</h1>
      </div>

      <div className="mt-6 max-w-3xl">
        {error && <Alert className="mb-4">{error}</Alert>}

        <Card className="shadow-[0_1px_2px_rgb(15_23_42/4%)]">
          <div className="grid gap-6 p-6">
            <div className="rounded-xl border border-slate-200 bg-slate-50 p-4 text-sm leading-6 text-slate-700">
              <p>{t.membershipImportInstructions}</p>
              <p className="mt-1">{t.membershipImportStateNotice}</p>
              <p className="mt-1 font-semibold text-amber-800">{t.membershipImportScopeNotice}</p>
              <p className="mt-2 text-xs text-slate-600">
                {t.membershipImportTransferPolicyNotice}
              </p>
              <p className="text-xs text-slate-600">{t.membershipImportSplitNotice}</p>
              <Button type="button" variant="outline" className="mt-3" onClick={downloadTemplate}>
                <IconDownload size={16} aria-hidden="true" />
                {t.downloadTemplate}
              </Button>
            </div>

            <div className="grid gap-2">
              <Label htmlFor="group-member-import-csv-file">{t.selectCsvFile}</Label>
              <input
                id="group-member-import-csv-file"
                type="file"
                accept=".csv,text/csv"
                onChange={handleFileChange}
                disabled={busy}
                aria-label={t.selectCsvFile}
                className="sr-only"
              />
              <label
                htmlFor="group-member-import-csv-file"
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
              <GroupMemberImportResultSummary
                t={t}
                title={t.previewResultTitle}
                result={previewResult}
                busy={busy}
                onErrorPage={(cursor) => void loadErrorPage(previewResult, cursor)}
              />
            )}

            {applyResult && step === 'apply_result' && (
              <GroupMemberImportResultSummary
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
              href={detailPath}
              className="inline-flex h-9 items-center justify-center rounded-lg border border-slate-200 bg-white px-4 text-sm font-medium text-slate-700 shadow-sm transition hover:bg-slate-50 hover:text-slate-900"
            >
              {step === 'apply_result' ? t.backToGroupDetail : t.cancel}
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
        <ApplyGroupMemberImportConfirmDialog
          result={previewResult}
          busy={busy}
          acknowledged={releaseAcknowledged}
          onAcknowledgedChange={setReleaseAcknowledged}
          onClose={() => {
            setShowApplyConfirm(false)
            setReleaseAcknowledged(false)
          }}
          onConfirm={() => void runApply()}
        />
      )}
    </AdminShell>
  )
}
