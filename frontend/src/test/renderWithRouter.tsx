import {
  createMemoryHistory,
  createRootRoute,
  createRouter,
  RouterProvider,
} from '@tanstack/react-router'
import { act, render } from '@testing-library/react'
import type { ReactElement } from 'react'
import { LocaleProvider, type Locale } from '../lib/i18n'

// AccountShell / AdminShell / AuthShell などは内部で <Link> (TanStack Router) を使うため、
// RouterProvider の外で render すると "useRouter must be used inside a <RouterProvider>" で落ちる。
// テスト対象コンポーネントだけを描画するダミーの root route を都度組み立てて満たす。
// ルーターの match 解決は複数 tick にまたがるため、呼び出し側は await して結果を待つこと。
export async function renderWithRouter(ui: ReactElement, options: { locale?: Locale } = {}) {
  const rootRoute = createRootRoute({
    component: () => <LocaleProvider initialLocale={options.locale ?? 'en'}>{ui}</LocaleProvider>,
  })
  const router = createRouter({
    routeTree: rootRoute,
    history: createMemoryHistory({ initialEntries: ['/'] }),
  })
  await router.load()
  let result!: ReturnType<typeof render>
  await act(async () => {
    result = render(<RouterProvider router={router} />)
  })
  return result
}
