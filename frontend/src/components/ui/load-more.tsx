import { commonDictionary } from '../../lib/i18n/common.i18n'
import { useDictionary } from '../../lib/i18n'
import { Button } from './button'

// LoadMoreButton は admin 一覧画面の keyset pagination 共通の「さらに読み込む」
// 操作。次ページが無ければ何も描画しない (Link ヘッダ不在 = 最終ページ)。
export function LoadMoreButton({
  hasMore,
  loading,
  onClick,
}: {
  hasMore: boolean
  loading: boolean
  onClick: () => void
}) {
  const t = useDictionary(commonDictionary)
  if (!hasMore) return null
  return (
    <div className="flex justify-center border-t border-slate-100 p-3">
      <Button type="button" variant="outline" disabled={loading} onClick={onClick}>
        {loading ? t.loadingMore : t.loadMore}
      </Button>
    </div>
  )
}
