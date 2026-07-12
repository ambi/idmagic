import { createBrowserHistory, createRouter } from '@tanstack/react-router'
import { AuthenticationAPIError, tenantBasePath, tenantLocalPath, tenantURL } from './api/core'
import { type PortalAudience, restartPortalLogin } from './api/oidc'
import { routeTree } from './routeTree.gen'
import { markErrorPage } from './routes/-page'
import { commonDictionary } from './lib/i18n/common.i18n'
import { useDictionary } from './lib/i18n'

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
  const t = useDictionary(commonDictionary)
  markErrorPage()
  const rawMessage =
    error instanceof AuthenticationAPIError ? error.message : t.authenticationUnavailable
  const expiredLogin = rawMessage.toLowerCase().includes('transaction')
  const reauth = expiredLogin ? null : portalReauthTarget()
  const title = expiredLogin ? t.loginRequestExpired : t.cannotContinueAuthentication
  const message = expiredLogin
    ? t.expiredLoginMessage
    : reauth
      ? t.sessionExpiredMessage
      : rawMessage || t.retryAuthenticationMessage
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
              {t.openAccount}
            </a>
            <a
              href={tenantURL('/admin')}
              className="inline-flex h-10 items-center justify-center rounded-lg border border-slate-300 bg-white px-4 text-sm font-semibold text-slate-800 hover:bg-slate-50"
            >
              {t.openAdmin}
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
              {t.signInAgain}
            </button>
            <a
              href={tenantURL(reauth.audience === 'account' ? '/account' : '/admin')}
              className="inline-flex h-10 items-center justify-center rounded-lg border border-slate-300 bg-white px-4 text-sm font-semibold text-slate-800 hover:bg-slate-50"
            >
              {reauth.audience === 'account' ? t.openAccount : t.openAdmin}
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
