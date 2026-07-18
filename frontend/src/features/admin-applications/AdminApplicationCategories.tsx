import { IconTrash } from '@tabler/icons-react'
import { useEffect, useState } from 'react'
import {
  createApplicationCategory,
  deleteApplicationCategory,
  listApplicationCategories,
  setApplicationCategories,
} from '../../api'
import { Button } from '../../components/ui/button'
import { Input } from '../../components/ui/input'
import { useDictionary } from '../../lib/i18n'
import { adminApplicationsDictionary } from './AdminApplicationsPage.i18n'
import { messageOf, SectionTitle } from './AdminApplicationsShared'
import type { AdminApplication, ApplicationCategory } from '../../types'

// カテゴリ (定義の管理 + アプリへの付与) — wi-70 / ADR-069
export function CategoryManager({
  app,
  csrfToken,
  onError,
}: {
  app: AdminApplication
  csrfToken: string
  onError: (msg: string) => void
}) {
  const [categories, setCategories] = useState<ApplicationCategory[]>([])
  const [assigned, setAssigned] = useState<Set<string>>(new Set(app.category_ids))
  const [newName, setNewName] = useState('')
  const [busy, setBusy] = useState(false)
  const [loaded, setLoaded] = useState(false)
  const t = useDictionary(adminApplicationsDictionary)

  useEffect(() => {
    let cancelled = false
    void listApplicationCategories()
      .then((list) => {
        if (cancelled) return
        setCategories(list)
        setLoaded(true)
      })
      .catch((cause) => onError(messageOf(cause, t.categoryFetchFailedError)))
    return () => {
      cancelled = true
    }
  }, [onError, t.categoryFetchFailedError])

  async function run(action: () => Promise<void>) {
    setBusy(true)
    try {
      await action()
    } catch (cause) {
      onError(messageOf(cause, t.categoryUpdateFailedError))
    } finally {
      setBusy(false)
    }
  }

  function toggle(categoryID: string) {
    const next = new Set(assigned)
    if (next.has(categoryID)) next.delete(categoryID)
    else next.add(categoryID)
    setAssigned(next)
    void run(async () => {
      const updated = await setApplicationCategories(csrfToken, app.application_id, [...next])
      setAssigned(new Set(updated.category_ids))
    })
  }

  function addCategory() {
    const name = newName.trim()
    if (name === '') return
    void run(async () => {
      const created = await createApplicationCategory(csrfToken, { name })
      setCategories((current) => [...current, created])
      setNewName('')
    })
  }

  function removeCategory(categoryID: string) {
    void run(async () => {
      await deleteApplicationCategory(csrfToken, categoryID)
      setCategories((current) => current.filter((c) => c.category_id !== categoryID))
      setAssigned((current) => {
        const next = new Set(current)
        next.delete(categoryID)
        return next
      })
    })
  }

  return (
    <div className="flex flex-col gap-4">
      <SectionTitle>{t.categoriesHeading}</SectionTitle>
      <p className="text-xs text-slate-500">{t.categoriesHelp}</p>
      {loaded && categories.length === 0 ? (
        <p className="text-sm text-slate-500">{t.noCategoriesNotice}</p>
      ) : (
        <ul className="flex flex-col gap-2">
          {categories.map((category) => (
            <li key={category.category_id} className="flex items-center gap-3">
              <label className="flex flex-1 items-center gap-2 text-sm text-slate-800">
                <input
                  type="checkbox"
                  checked={assigned.has(category.category_id)}
                  onChange={() => toggle(category.category_id)}
                  disabled={busy}
                  className="size-4 rounded border-slate-300"
                />
                {category.name}
              </label>
              <Button
                type="button"
                variant="ghost"
                size="default"
                className="size-9 px-0 text-slate-400 hover:text-red-600"
                disabled={busy}
                onClick={() => removeCategory(category.category_id)}
                aria-label={t.deleteCategoryAria.replace('{name}', category.name)}
              >
                <IconTrash size={16} aria-hidden="true" />
              </Button>
            </li>
          ))}
        </ul>
      )}
      <div className="flex items-center gap-2">
        <Input
          value={newName}
          onChange={(e) => setNewName(e.target.value)}
          placeholder={t.newCategoryPlaceholder}
          disabled={busy}
          onKeyDown={(e) => {
            if (e.key === 'Enter') {
              e.preventDefault()
              addCategory()
            }
          }}
        />
        <Button type="button" variant="secondary" onClick={addCategory} disabled={busy}>
          {t.add}
        </Button>
      </div>
    </div>
  )
}
