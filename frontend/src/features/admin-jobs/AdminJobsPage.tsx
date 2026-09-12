import { type AdminJob, cancelAdminJob, listAdminJobs } from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { useDictionary } from '../../lib/i18n'
import { adminJobsDictionary } from './AdminJobsPage.i18n'
import { JobsBrowser } from './JobsBrowser'

// AdminJobsPage はテナント管理コンソールのジョブ画面。範囲は常に要求先テナントであり、
// 操作者が `system_admin` であっても横断しない。横断はシステムコンソールが持つ
// (REQ-SYSTEM-020)。そのため、このページはロールも realm も見ない。
export function AdminJobsPage({
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
    <AdminShell
      active="jobs"
      actorUsername={actorUsername}
      title={t.pageTitle}
      description={t.pageDescription}
    >
      <JobsBrowser
        jobs={jobs}
        nextCursor={nextCursor}
        kinds={kinds}
        t={t}
        listJobs={listAdminJobs}
        cancelJob={(job) => cancelAdminJob(csrfToken, job.id)}
        confirmCancel={() => window.confirm(t.cancelConfirm)}
        initialError={initialError}
      />
    </AdminShell>
  )
}
