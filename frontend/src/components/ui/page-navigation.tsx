import { useDictionary } from '../../lib/i18n'
import { commonDictionary } from '../../lib/i18n/common.i18n'
import { Button } from './button'

export function PageNavigation({
  previousCursor,
  nextCursor,
  onNavigate,
}: {
  previousCursor: string | null
  nextCursor: string | null
  onNavigate: (cursor: string) => void
}) {
  const t = useDictionary(commonDictionary)
  if (!previousCursor && !nextCursor) return null
  return (
    <div className="flex justify-between gap-3 border-t border-slate-100 p-3">
      <div>
        {previousCursor ? (
          <Button type="button" variant="outline" onClick={() => onNavigate(previousCursor)}>
            {t.previousPage}
          </Button>
        ) : null}
      </div>
      <div>
        {nextCursor ? (
          <Button type="button" variant="outline" onClick={() => onNavigate(nextCursor)}>
            {t.nextPage}
          </Button>
        ) : null}
      </div>
    </div>
  )
}
