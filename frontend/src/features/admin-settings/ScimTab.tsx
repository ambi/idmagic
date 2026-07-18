import { type FormEvent, useEffect, useState } from 'react'
import { createScimToken, listScimTokens, revokeScimToken } from '../../api'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { Toast } from '../../components/ui/toast'
import { useDictionary, useLocale } from '../../lib/i18n'
import type { ScimToken } from '../../types'
import { adminSettingsDictionary } from './AdminSettingsPage.i18n'

export function ScimTab({ csrfToken, tenantID }: { csrfToken: string; tenantID: string }) {
  const [tokens, setTokens] = useState<ScimToken[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const [tokenDesc, setTokenDesc] = useState('')
  const [tokenExpiry, setTokenExpiry] = useState('7')
  const [generatedToken, setGeneratedToken] = useState('')
  const [creating, setCreating] = useState(false)
  const t = useDictionary(adminSettingsDictionary)
  const { locale } = useLocale()

  // biome-ignore lint/correctness/useExhaustiveDependencies: 初回マウント時のみ取得する
  useEffect(() => {
    async function loadData() {
      try {
        const tList = await listScimTokens()
        setTokens(tList)
      } catch {
        setError(t.scimTokensFetchFailedError)
      } finally {
        setLoading(false)
      }
    }
    loadData()
  }, [])

  async function handleCreateToken(e: FormEvent) {
    e.preventDefault()
    setError('')
    setNotice('')
    setGeneratedToken('')
    if (!tokenDesc.trim()) {
      setError(t.tokenDescriptionRequiredError)
      return
    }
    try {
      const res = await createScimToken(csrfToken, {
        description: tokenDesc.trim(),
        expiry_days: Number.parseInt(tokenExpiry, 10),
      })
      setGeneratedToken(res.token)
      setTokenDesc('')
      setCreating(false)
      const tList = await listScimTokens()
      setTokens(tList)
      setNotice(t.scimTokenIssuedNotice)
    } catch {
      setError(t.tokenIssueFailedError)
    }
  }

  async function handleRevokeToken(id: string) {
    setError('')
    setNotice('')
    try {
      await revokeScimToken(csrfToken, id)
      setTokens(tokens.filter((token) => token.id !== id))
      setNotice(t.tokenRevokedNotice)
    } catch {
      setError(t.tokenRevokeFailedError)
    }
  }

  if (loading) {
    return <div className="text-sm text-slate-500">{t.loadingNotice}</div>
  }

  const endpointUrl = `${window.location.origin}/realms/${tenantID}/scim/v2`

  return (
    <Card className="p-6">
      <header>
        <h2 className="text-base font-semibold text-slate-900">{t.scimHeading}</h2>
        <p className="mt-1 text-sm text-slate-600">{t.scimDescription}</p>
      </header>

      <div className="mt-6 grid gap-6">
        {error ? <Alert variant="destructive">{error}</Alert> : null}
        <Toast message={notice} onDismiss={() => setNotice('')} />

        <div className="rounded-lg border border-slate-200 bg-slate-50 p-4">
          <h3 className="text-sm font-semibold text-slate-900">{t.connectionInfoHeading}</h3>
          <div className="mt-3 grid gap-3">
            <div>
              <span className="text-xs text-slate-500">{t.scimBaseUrlLabel}</span>
              <div className="mt-1 flex items-center gap-2">
                <input
                  readOnly
                  value={endpointUrl}
                  className="flex-1 rounded-md border border-slate-300 bg-white px-3 py-1.5 font-mono text-sm"
                />
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => {
                    navigator.clipboard.writeText(endpointUrl)
                    setNotice(t.urlCopiedNotice)
                  }}
                >
                  {t.copy}
                </Button>
              </div>
              <p className="mt-1 text-xs text-slate-500">{t.scimConnectorHelp}</p>
            </div>
          </div>
        </div>

        <div className="grid gap-4">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <h3 className="text-sm font-semibold text-slate-900">{t.scimTokenHeading}</h3>
            {!creating ? (
              <Button type="button" variant="outline" onClick={() => setCreating(true)}>
                {t.issueToken}
              </Button>
            ) : null}
          </div>

          {generatedToken ? (
            <div className="rounded-lg border border-emerald-200 bg-emerald-50 p-4">
              <h4 className="text-sm font-bold text-emerald-800">{t.issuedTokenHeading}</h4>
              <p className="mt-1 text-xs text-emerald-700">{t.issuedTokenWarning}</p>
              <div className="mt-3 flex items-center gap-2">
                <input
                  readOnly
                  value={generatedToken}
                  className="flex-1 rounded-md border border-emerald-300 bg-white px-3 py-1.5 font-mono text-sm text-emerald-900"
                />
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => {
                    navigator.clipboard.writeText(generatedToken)
                    setNotice(t.tokenCopiedNotice)
                  }}
                >
                  {t.copy}
                </Button>
              </div>
            </div>
          ) : null}

          {tokens.length === 0 ? (
            <p className="text-sm text-slate-500">{t.noTokensNotice}</p>
          ) : (
            <div className="overflow-x-auto rounded-lg border border-slate-200">
              <table className="min-w-full divide-y divide-slate-200 text-left text-sm text-slate-700">
                <thead className="bg-slate-50 font-semibold text-slate-900">
                  <tr>
                    <th className="px-4 py-2">{t.tableHeaderDescription}</th>
                    <th className="px-4 py-2">{t.tableHeaderCreatedAt}</th>
                    <th className="px-4 py-2">{t.tableHeaderExpiresAt}</th>
                    <th className="px-4 py-2">{t.tableHeaderAction}</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-200">
                  {tokens.map((tok) => (
                    <tr key={tok.id}>
                      <td className="px-4 py-3">{tok.description}</td>
                      <td className="px-4 py-3">
                        {new Date(tok.created_at).toLocaleString(
                          locale === 'ja' ? 'ja-JP' : 'en-US',
                        )}
                      </td>
                      <td className="px-4 py-3">
                        {tok.expires_at
                          ? new Date(tok.expires_at).toLocaleString(
                              locale === 'ja' ? 'ja-JP' : 'en-US',
                            )
                          : t.noneLabel}
                      </td>
                      <td className="px-4 py-3">
                        <Button
                          type="button"
                          variant="ghost"
                          className="text-red-600 hover:text-red-700 hover:bg-red-50"
                          onClick={() => handleRevokeToken(tok.id)}
                        >
                          {t.revoke}
                        </Button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}

          {creating ? (
            <form
              onSubmit={handleCreateToken}
              className="mt-4 rounded-lg border border-slate-200 p-4"
            >
              <h4 className="text-sm font-semibold text-slate-900">{t.newTokenHeading}</h4>
              <div className="mt-3 grid gap-4 sm:grid-cols-2">
                <div className="grid gap-1.5">
                  <Label htmlFor="token-desc">{t.tokenDescriptionLabel}</Label>
                  <Input
                    id="token-desc"
                    placeholder={t.tokenDescriptionPlaceholder}
                    value={tokenDesc}
                    onChange={(e) => setTokenDesc(e.target.value)}
                  />
                </div>
                <div className="grid gap-1.5">
                  <Label htmlFor="token-expiry">{t.tokenExpiryLabel}</Label>
                  <Input
                    id="token-expiry"
                    type="number"
                    min={1}
                    max={365}
                    value={tokenExpiry}
                    onChange={(e) => setTokenExpiry(e.target.value)}
                  />
                </div>
              </div>
              <div className="mt-4 flex items-center gap-2">
                <Button type="submit">{t.issueToken}</Button>
                <Button
                  type="button"
                  variant="ghost"
                  onClick={() => {
                    setTokenDesc('')
                    setError('')
                    setCreating(false)
                  }}
                >
                  {t.cancel}
                </Button>
              </div>
            </form>
          ) : null}
        </div>
      </div>
    </Card>
  )
}
