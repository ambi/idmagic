import { createFileRoute } from '@tanstack/react-router'
import { type AdminJob, AuthenticationAPIError, listAdminJobs } from '../../api'
import { AdminJobsPage } from '../../features/admin-jobs/AdminJobsPage'
import { requirePortalAccount } from '../-guards'
import { PageMarker } from '../-page'

// 登録済みの JobKind は API が別途返さないので、読み込んだページに現れた種別から
// 絞り込みの選択肢を作る。手書きの一覧を UI に持つと Go 側の登録と食い違う。
export function jobKindsOf(jobs: AdminJob[]): string[] {
  return [...new Set(jobs.map((job) => job.kind))].sort()
}

export const Route = createFileRoute('/admin/jobs')({
  loader: async ({ location }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    // 一覧の取得失敗はページ全体を壊さず、ページ内のエラー表示に留める。
    // 認証そのものの失敗は requirePortalAccount 側が扱う。
    let jobs: AdminJob[] = []
    let nextCursor: string | undefined
    let initialError = ''
    try {
      const page = await listAdminJobs()
      jobs = page.jobs
      nextCursor = page.next_cursor
    } catch (cause) {
      initialError = cause instanceof AuthenticationAPIError ? cause.message : String(cause)
    }
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
      actorRoles: account.roles ?? [],
      actorRealm: account.realm ?? '',
      jobs,
      nextCursor,
      initialError,
    }
  },
  component: AdminJobsRoute,
})

function AdminJobsRoute() {
  const data = Route.useLoaderData()
  return (
    <PageMarker kind="admin-jobs">
      <AdminJobsPage
        actorUsername={data.actorUsername}
        actorRoles={data.actorRoles}
        actorRealm={data.actorRealm}
        jobs={data.jobs}
        nextCursor={data.nextCursor}
        kinds={jobKindsOf(data.jobs)}
        csrfToken={data.csrfToken}
        initialError={data.initialError}
      />
    </PageMarker>
  )
}
