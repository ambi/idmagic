import { createBrowserHistory, createRouter } from '@tanstack/react-router'
import { AuthenticationAPIError, tenantBasePath, tenantLocalPath, tenantURL } from './api/core'
import { type PortalAudience, restartPortalLogin } from './api/oidc'
import { routeTree } from './routeTree.gen'
import { markErrorPage } from './routes/-page'

export function preloadPageChunks() {
  // File-based routes with autoCodeSplitting let TanStack Router/Vite own route chunk loading.
}

// portalReauthTarget は現在のパスが first-party portal (admin/account) 配下なら、
// 再ログイン後に戻す audience と同一オリジンの return_to を返す。行き止まり画面から
// 元の画面へ復帰する再ログイン導線を出すために使う (open redirect を避けるため
// return_to は現在の相対パスに固定する)。
function portalReauthTarget(): { audience: PortalAudience; returnTo: string } | null {
  const local = tenantLocalPath()
  const returnTo = `${window.location.pathname}${window.location.search}`
  if (local === '/admin' || local.startsWith('/admin/')) return { audience: 'admin', returnTo }
  if (local === '/account' || local.startsWith('/account/'))
    return { audience: 'account', returnTo }
  return null
}

function ErrorScreen({ error }: { error: unknown }) {
  markErrorPage()
  const rawMessage =
    error instanceof AuthenticationAPIError ? error.message : '認証画面を読み込めませんでした。'
  const expiredLogin =
    rawMessage.includes('認可トランザクション') || rawMessage.toLowerCase().includes('transaction')
  const reauth = expiredLogin ? null : portalReauthTarget()
  const title = expiredLogin ? 'ログイン要求が終了しています' : '認証を続行できません'
  const message = expiredLogin
    ? '前のログイン要求は完了または期限切れになっています。利用したい画面をもう一度開いてログインしてください。'
    : reauth
      ? 'セッションの有効期限が切れました。もう一度ログインすると、この画面に戻ります。'
      : rawMessage || '認証画面を読み込めませんでした。もう一度お試しください。'
  return (
    <main className="flex min-h-screen items-center justify-center bg-slate-100 p-6">
      <div className="w-full max-w-md rounded-2xl border bg-white p-8 text-center shadow-lg">
        <h1 className="text-xl font-semibold text-slate-950">{title}</h1>
        <p className="mt-3 text-sm leading-6 text-slate-600">{message}</p>
        {expiredLogin ? (
          <div className="mt-6 grid gap-2">
            <a
              href={tenantURL('/account')}
              className="inline-flex h-10 items-center justify-center rounded-lg bg-slate-950 px-4 text-sm font-semibold text-white hover:bg-slate-800"
            >
              マイページを開く
            </a>
            <a
              href={tenantURL('/admin')}
              className="inline-flex h-10 items-center justify-center rounded-lg border border-slate-300 bg-white px-4 text-sm font-semibold text-slate-800 hover:bg-slate-50"
            >
              管理コンソールを開く
            </a>
          </div>
        ) : reauth ? (
          <div className="mt-6 grid gap-2">
            <button
              type="button"
              onClick={() => {
                void restartPortalLogin(reauth.audience, reauth.returnTo)
              }}
              className="inline-flex h-10 items-center justify-center rounded-lg bg-slate-950 px-4 text-sm font-semibold text-white hover:bg-slate-800"
            >
              再ログイン
            </button>
            <a
              href={tenantURL(reauth.audience === 'account' ? '/account' : '/admin')}
              className="inline-flex h-10 items-center justify-center rounded-lg border border-slate-300 bg-white px-4 text-sm font-semibold text-slate-800 hover:bg-slate-50"
            >
              {reauth.audience === 'account' ? 'マイページを開く' : '管理コンソールを開く'}
            </a>
          </div>
        ) : null}
      </div>
    </main>
  )
}

export function createAppRouter() {
  return createRouter({
    routeTree,
    history: createBrowserHistory(),
    basepath: tenantBasePath() || '/',
    defaultPreload: 'intent',
    // pendingComponent を設定しないことで、loader 解決中は前ページを表示したままにする。
    defaultErrorComponent: ({ error }) => <ErrorScreen error={error} />,
  })
}
