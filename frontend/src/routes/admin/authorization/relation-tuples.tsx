import { createFileRoute, useRouter } from '@tanstack/react-router'
import { listRelationTuples } from '../../../api/admin'
import { AdminAuthorizationTuplesPage } from '../../../features/admin-authorization/AdminAuthorizationTuplesPage'
import type { RelationTupleFilters } from '../../../types'
import { requirePortalAccount } from '../../-guards'
import { PageMarker } from '../../-page'
import { useAuthorizationNotice } from './-notice'

const FILTER_KEYS = [
  'resource_type',
  'resource_id',
  'relation',
  'subject_type',
  'subject_id',
] as const

export function validateTupleSearch(search: Record<string, unknown>): RelationTupleFilters {
  const filters: RelationTupleFilters = {}
  for (const key of FILTER_KEYS) {
    const value = search[key]
    // 数字だけの識別子は検索パラメーターの解析で数値になるので、文字列へ戻す。
    if (typeof value === 'string' && value) filters[key] = value
    else if (typeof value === 'number') filters[key] = String(value)
  }
  return filters
}

export const Route = createFileRoute('/admin/authorization/relation-tuples')({
  validateSearch: validateTupleSearch,
  loaderDeps: ({ search }) => search,
  loader: async ({ deps, location }) => {
    const account = await requirePortalAccount('admin', location.pathname, location.searchStr)
    const listed = await listRelationTuples(deps)
    return {
      csrfToken: account.csrf_token,
      actorUsername: account.preferred_username,
      tuples: listed.tuples,
      consistency: listed.consistency,
      filters: deps,
    }
  },
  component: AdminAuthorizationTuplesRoute,
})

function AdminAuthorizationTuplesRoute() {
  const data = Route.useLoaderData()
  const notice = useAuthorizationNotice()
  const router = useRouter()
  return (
    <PageMarker kind="admin-authorization-tuples">
      <AdminAuthorizationTuplesPage
        // 取得結果をコンテナの状態へ写す。ルーターはキャッシュした結果を先に描画し、裏で読み直すので、
        // 条件か整合トークンが変わったら作り直す。
        key={`${JSON.stringify(data.filters)}:${data.consistency}`}
        {...data}
        justAdded={notice === 'tuple-added'}
        onFiltersChange={(search) =>
          void router.navigate({ to: '/admin/authorization/relation-tuples', search })
        }
      />
    </PageMarker>
  )
}
