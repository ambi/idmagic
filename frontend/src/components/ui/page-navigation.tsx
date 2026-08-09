import { useDictionary } from '../../lib/i18n'
import { commonDictionary } from '../../lib/i18n/common.i18n'
import { Button } from './button'

export type PageNavigationData = {
  hasFirst: boolean
  previousCursor: string | null
  nextCursor: string | null
  lastCursor: string | null
  totalItems: number
  totalPages: number
  currentPage: number
}

export function PageNavigation({
  hasFirst = false,
  previousCursor = null,
  nextCursor,
  lastCursor = null,
  totalItems = 0,
  totalPages = 0,
  currentPage = 0,
  onNavigate,
}: Partial<Omit<PageNavigationData, 'nextCursor'>> &
  Pick<PageNavigationData, 'nextCursor'> & {
    onNavigate: (cursor: string | null) => void
  }) {
  const t = useDictionary(commonDictionary)
  return (
    <div className="flex flex-wrap items-center justify-between gap-3 border-t border-slate-100 p-3">
      <div className="text-sm text-slate-600">
        <span>{`${totalItems} ${t.paginationItems}`}</span>
        <span className="px-2" aria-hidden="true">
          ·
        </span>
        <span>{`${currentPage} / ${totalPages}`}</span>
      </div>
      <div className="flex flex-wrap gap-2">
        <Button
          type="button"
          variant="outline"
          disabled={!hasFirst}
          onClick={() => onNavigate(null)}
        >
          {t.firstPage}
        </Button>
        <Button
          type="button"
          variant="outline"
          disabled={!previousCursor}
          onClick={() => previousCursor && onNavigate(previousCursor)}
        >
          {t.previousPage}
        </Button>
        <Button
          type="button"
          variant="outline"
          disabled={!nextCursor}
          onClick={() => nextCursor && onNavigate(nextCursor)}
        >
          {t.nextPage}
        </Button>
        <Button
          type="button"
          variant="outline"
          disabled={!lastCursor}
          onClick={() => lastCursor && onNavigate(lastCursor)}
        >
          {t.lastPage}
        </Button>
      </div>
    </div>
  )
}
