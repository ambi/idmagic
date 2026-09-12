import { createFileRoute } from '@tanstack/react-router'
import { type AdminJob, AuthenticationAPIError, listSystemJobs } from '../../api'
import { SystemJobsPage } from '../../features/admin-jobs/SystemJobsPage'
import { jobKindsOf } from '../admin/jobs'
import { requireSystemAccount } from '../-guards'
import { PageMarker } from '../-page'

export const Route = createFileRoute('/system/jobs')({
  loader: async ({ location }) => {
    const account = await requireSystemAccount(location.pathname, location.searchStr)
    // 一覧の取得失敗はページ全体を壊さず、ページ内のエラー表示に留める。
    // 認証そのものの失敗は requireSystemAccount 側が扱う。
    let jobs: AdminJob[] = []
    let nextCursor: string | undefined
    let initialError = ''
    try {
      const page = await listSystemJobs()
      jobs = page.jobs
      nextCursor = page.next_cursor
    } catch (cause) {
      initialError = cause instanceof AuthenticationAPIError ? cause.message : String(cause)
    }
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
      jobs,
      nextCursor,
      initialError,
    }
  },
  component: SystemJobsRoute,
})

function SystemJobsRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="system-jobs">
      <SystemJobsPage
        actorUsername={data.actorUsername}
        jobs={data.jobs}
        nextCursor={data.nextCursor}
        kinds={jobKindsOf(data.jobs)}
        csrfToken={data.csrfToken}
        initialError={data.initialError}
      />
    </PageMarker>
  )
}
