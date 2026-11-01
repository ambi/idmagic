import { act, cleanup } from '@testing-library/react'
import { expect, it } from 'bun:test'
import { adminNavigationPages, clickAdminNavigation } from '../../tests/e2e/admin-navigation'
import { renderWithRouter } from '../test/renderWithRouter'
import { AdminNavigation } from './AdminNavigation'

// E2E と同じスクリプトを実コンポーネントの DOM に実行する。
// 実ブラウザーの代わりに評価先だけを変え、操作の実装は置き換えない。
const view = {
  async evaluate(script: string) {
    let result: unknown
    await act(async () => {
      result = new Function('document', `return ${script}`)(document)
    })
    return result
  },
} as Pick<Bun.WebView, 'evaluate'>

it.each(['ja', 'en'] as const)(
  '%s の E2E の全管理導線が閉じた領域と現在の領域から使える',
  async (locale) => {
    for (const page of adminNavigationPages) {
      for (const active of ['dashboard', page.key] as const) {
        await renderWithRouter(<AdminNavigation active={active} />, { locale })
        let destination = ''
        const capture = (event: MouseEvent) => {
          const anchor = (event.target as Element).closest('a')
          if (anchor) {
            event.preventDefault()
            destination = anchor.getAttribute('href') ?? ''
          }
        }
        document.addEventListener('click', capture, true)
        try {
          await clickAdminNavigation(view, page)
          expect(destination, `${locale}: ${active} -> ${page.path}`).toBe(page.path)
        } finally {
          document.removeEventListener('click', capture, true)
          cleanup()
        }
      }
    }
  },
)
