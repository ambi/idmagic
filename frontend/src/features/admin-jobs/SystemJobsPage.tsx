import { type AdminJob, cancelSystemJob, listSystemJobs } from '../../api'
import { SystemShell } from '../../components/SystemShell'
import { useDictionary } from '../../lib/i18n'
import { adminJobsDictionary } from './AdminJobsPage.i18n'
import { JobsBrowser } from './JobsBrowser'

// SystemJobsPage はシステムコンソールのジョブ画面。範囲は全テナントに固定され、切り替える
// UI を持たない。呼ぶのは専用のシステム API だけである (REQ-JOBS-015 / REQ-SYSTEM-020)。
//
// 取り消しは他テナントの実行へ影響するので、確認に対象の tenant_id、ジョブ種別、ジョブ ID を
// 再掲する。一覧では対象テナントを列として常時見せる。
export function SystemJobsPage({
  actorUsername,
  jobs,
  nextCursor,
  kinds,
  csrfToken,
  initialError,
}: {
  actorUsername?: string
  jobs: AdminJob[]
  nextCursor?: string
  kinds: string[]
  csrfToken: string
  initialError?: string
}) {
  const t = useDictionary(adminJobsDictionary)
  return (
    <SystemShell
      active="jobs"
      actorUsername={actorUsername}
      title={t.systemPageTitle}
      description={t.systemPageDescription}
    >
      <JobsBrowser
        jobs={jobs}
        nextCursor={nextCursor}
        kinds={kinds}
        t={t}
        tenantColumn
        listJobs={listSystemJobs}
        cancelJob={(job) => cancelSystemJob(csrfToken, job.id)}
        confirmCancel={(job) =>
          window.confirm(
            `${t.systemCancelConfirm}\n\n${t.tableHeaderTenant}: ${job.tenant_id}\n${t.tableHeaderKind}: ${job.kind}\n${t.idLabel}: ${job.id}`,
          )
        }
        initialError={initialError}
      />
    </SystemShell>
  )
}
