import { useCallback, useState } from 'react'

export type ListPage<T> = { items: T[]; nextCursor: string | null }

// usePaginatedList は admin 一覧画面の「さらに読み込む」操作 (Link ヘッダ由来の
// nextCursor) に共通する蓄積・二重読み込み防止のみを持つ。エラーの捕捉・表示文言は既存の
// 各画面の慣習 (AuthenticationAPIError 判定 + 画面固有メッセージ) に揃えるため、呼び出し側で
// loadMore() を try/catch する想定であえて持たない。
export function usePaginatedList<T>(initial: ListPage<T>) {
  const [items, setItems] = useState<T[]>(initial.items)
  const [nextCursor, setNextCursor] = useState<string | null>(initial.nextCursor)
  const [loadingMore, setLoadingMore] = useState(false)

  const loadMore = useCallback(
    async (fetchPage: (cursor: string) => Promise<ListPage<T>>) => {
      if (!nextCursor || loadingMore) return
      setLoadingMore(true)
      try {
        const page = await fetchPage(nextCursor)
        setItems((current) => [...current, ...page.items])
        setNextCursor(page.nextCursor)
      } finally {
        setLoadingMore(false)
      }
    },
    [nextCursor, loadingMore],
  )

  const reset = useCallback((page: ListPage<T>) => {
    setItems(page.items)
    setNextCursor(page.nextCursor)
  }, [])

  return {
    items,
    setItems,
    nextCursor,
    hasMore: nextCursor !== null,
    loadingMore,
    loadMore,
    reset,
  }
}
