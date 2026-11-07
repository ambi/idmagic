import { type FormEvent, useEffect, useMemo, useState } from 'react'
import { createApiToken, listApiTokens, revokeApiToken } from '../../api'
import { request, adminRequest, tenantURL } from '../../api/core'
import { ConfirmActionDialog } from '../../components/ConfirmActionDialog'
import { useStepUpGuard } from '../../components/StepUpDialog'
import { ResourceListToolbar } from '../../components/ResourceListToolbar'
import { FieldError } from '../../components/FieldError'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { Toast } from '../../components/ui/toast'
import {
  formSchema,
  integerInRange,
  positiveInteger,
  requiredText,
  useFormValidation,
} from '../../lib/form'
import { useDictionary, useLocale } from '../../lib/i18n'
import type { AdminIntegrationEndpointCatalog, ApiToken, ApiTokenScope } from '../../types'
import { adminSettingsDictionary } from './AdminSettingsPage.i18n'
import { ApiTokenScopePicker } from './ApiTokenScopePicker'

// 個人用のトークンは 1 から 365 日（IssuePersonal）、管理者のトークンは 1 日以上（Issue）である。
const PERSONAL_MAX_EXPIRY_DAYS = 365

function tokenSchema(personal: boolean) {
  return formSchema({
    description: requiredText(),
    expiryDays: personal
      ? integerInRange(1, PERSONAL_MAX_EXPIRY_DAYS, { required: true })
      : positiveInteger(),
  })
}

export function ApiTokensTab({
  csrfToken,
  integrationEndpoints,
  actorUsername,
  personal = false,
  screen,
}: {
  csrfToken: string
  integrationEndpoints?: AdminIntegrationEndpointCatalog
  actorUsername?: string
  personal?: boolean
  screen?: 'list' | 'create'
}) {
  const [tokens, setTokens] = useState<ApiToken[]>([])
  const [loading, setLoading] = useState(screen !== 'create')
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const [tokenDesc, setTokenDesc] = useState('')
  const [tokenExpiry, setTokenExpiry] = useState('7')
  const [selectedScopes, setSelectedScopes] = useState<ApiTokenScope[]>([])
  const [generatedToken, setGeneratedToken] = useState('')
  const [creating, setCreating] = useState(screen === 'create')
  const [busy, setBusy] = useState(false)
  const [pendingRevoke, setPendingRevoke] = useState<ApiToken | null>(null)
  const t = useDictionary(adminSettingsDictionary)
  const schema = useMemo(() => tokenSchema(personal), [personal])
  const validation = useFormValidation(schema, { description: tokenDesc, expiryDays: tokenExpiry })
  const { locale } = useLocale()
  const { guard, dialog } = useStepUpGuard(csrfToken)
  const basePath = personal ? '/account/api-tokens' : '/admin/api-tokens'
  const personalAPI = '/api/account/v1/api-tokens'
  const listTokens = async () =>
    personal ? (await request<{ tokens: ApiToken[] }>(personalAPI)).tokens : listApiTokens()
  async function refresh() {
    setTokens(await listTokens())
  }

  // biome-ignore lint/correctness/useExhaustiveDependencies: 初回マウント時のみ取得する
  useEffect(() => {
    if (screen === 'create') return
    async function loadData() {
      try {
        const tList = await listTokens()
        setTokens(tList)
      } catch {
        setError(t.apiTokensFetchFailedError)
      } finally {
        setLoading(false)
      }
    }
    loadData()
  }, [])

  async function handleCreateToken(e: FormEvent) {
    e.preventDefault()
    if (busy) return
    setError('')
    setNotice('')
    setGeneratedToken('')
    const data = validation.data
    if (!data) {
      validation.revealAll()
      return
    }
    try {
      setBusy(true)
      const input = {
        description: data.description,
        scopes: selectedScopes,
        expiry_days: data.expiryDays ?? 0,
      }
      const res = personal
        ? await guard(() =>
            request<{ token: string; meta: ApiToken }>(
              personalAPI,
              adminRequest(csrfToken, 'POST', input),
            ),
          )
        : await createApiToken(csrfToken, input)
      setGeneratedToken(res.token)
      setTokenDesc('')
      setSelectedScopes([])
      setCreating(false)
      if (screen !== 'create') await refresh()
      setNotice(t.apiTokenIssuedNotice)
    } catch {
      setError(t.tokenIssueFailedError)
    } finally {
      setBusy(false)
    }
  }

  async function handleRevokeToken(id: string) {
    setError('')
    setNotice('')
    try {
      setBusy(true)
      if (personal)
        await request(`${personalAPI}/${encodeURIComponent(id)}`, adminRequest(csrfToken, 'DELETE'))
      else await revokeApiToken(csrfToken, id)
      setTokens(
        tokens.map((token) =>
          token.id === id ? { ...token, revoked_at: new Date().toISOString() } : token,
        ),
      )
      setPendingRevoke(null)
      setNotice(t.tokenRevokedNotice)
    } catch {
      setError(t.tokenRevokeFailedError)
    } finally {
      setBusy(false)
    }
  }

  if (loading) {
    return <div className="text-sm text-slate-500">{t.loadingNotice}</div>
  }

  const managementApiBaseUrl = integrationEndpoints?.apis.management_api_base_url ?? ''
  const scimBaseUrl = integrationEndpoints?.apis.scim_base_url ?? ''
  const accountApiBaseUrl =
    integrationEndpoints?.apis.account_api_base_url ??
    new URL(tenantURL('/api/account/v1'), window.location.origin).href

  return (
    <Card className="p-6">
      {dialog}
      <header>
        <h2 className="text-base font-semibold text-slate-900">{t.apiTokensHeading}</h2>
        <p className="mt-1 text-sm text-slate-600">
          {personal ? t.personalTokenDescription : t.tenantTokenDescription}
        </p>
      </header>

      <div className="mt-6 grid gap-6">
        {error ? <Alert variant="destructive">{error}</Alert> : null}
        <Toast message={notice} onDismiss={() => setNotice('')} />
        {creating && !personal ? (
          <section
            aria-label={t.tokenIdentityHeading}
            className="rounded-lg border border-blue-200 bg-blue-50 p-4"
          >
            <dl className="grid gap-4 sm:grid-cols-2">
              <div>
                <dt className="text-xs font-medium text-slate-600">{t.tokenIssuerLabel}</dt>
                <dd className="mt-1 text-sm font-semibold text-slate-950">
                  {actorUsername ?? t.currentUserLabel}
                </dd>
              </div>
              <div>
                <dt className="text-xs font-medium text-slate-600">{t.tokenTenantLabel}</dt>
                <dd className="mt-1 break-all font-mono text-xs text-slate-900">
                  {integrationEndpoints?.issuer}
                </dd>
              </div>
            </dl>
            <p className="mt-3 text-sm text-slate-700">{t.tokenIdentityDescription}</p>
          </section>
        ) : null}
        {screen !== 'create' ? (
          <ResourceListToolbar
            total={tokens.length}
            displayed={tokens.length}
            busy={busy}
            onRefresh={refresh}
          />
        ) : null}

        <div className="rounded-lg border border-slate-200 bg-slate-50 p-4">
          <h3 className="text-sm font-semibold text-slate-900">{t.connectionInfoHeading}</h3>
          <div className="mt-3 grid gap-3">
            {!personal ? (
              <div>
                <Label htmlFor="management-api-base-url">{t.managementApiBaseUrlLabel}</Label>
                <div className="mt-1 flex items-center gap-2">
                  <input
                    id="management-api-base-url"
                    readOnly
                    value={managementApiBaseUrl}
                    className="flex-1 rounded-md border border-slate-300 bg-white px-3 py-1.5 font-mono text-sm"
                  />
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => {
                      navigator.clipboard.writeText(managementApiBaseUrl)
                      setNotice(t.urlCopiedNotice)
                    }}
                  >
                    {t.copy}
                  </Button>
                </div>
                <p className="mt-1 text-xs text-slate-500">{t.managementApiHelp}</p>
              </div>
            ) : null}
            {!personal ? (
              <div>
                <Label htmlFor="scim-api-base-url">{t.scimBaseUrlLabel}</Label>
                <div className="mt-1 flex items-center gap-2">
                  <input
                    id="scim-api-base-url"
                    readOnly
                    value={scimBaseUrl}
                    className="flex-1 rounded-md border border-slate-300 bg-white px-3 py-1.5 font-mono text-sm"
                  />
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => {
                      navigator.clipboard.writeText(scimBaseUrl)
                      setNotice(t.urlCopiedNotice)
                    }}
                  >
                    {t.copy}
                  </Button>
                </div>
                <p className="mt-1 text-xs text-slate-500">{t.scimConnectorHelp}</p>
              </div>
            ) : null}
            <div>
              <Label htmlFor="account-api-base-url">{t.accountApiBaseUrlLabel}</Label>
              <div className="mt-1 flex items-center gap-2">
                <input
                  id="account-api-base-url"
                  readOnly
                  value={accountApiBaseUrl}
                  className="flex-1 rounded-md border border-slate-300 bg-white px-3 py-1.5 font-mono text-sm"
                />
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => {
                    navigator.clipboard.writeText(accountApiBaseUrl)
                    setNotice(t.urlCopiedNotice)
                  }}
                >
                  {t.copy}
                </Button>
              </div>
              <p className="mt-1 text-xs text-slate-500">{t.accountApiHelp}</p>
            </div>
          </div>
        </div>

        <div className="grid gap-4">
          <div className="flex flex-wrap items-center justify-between gap-2">
            {screen !== 'create' ? (
              <h3 className="text-sm font-semibold text-slate-900">
                {personal ? t.personalTokensHeading : t.apiTokensListHeading}
              </h3>
            ) : null}
            {!creating && screen === 'list' ? (
              <Button nativeButton={false} render={<a href={tenantURL(`${basePath}/new`)} />}>
                {t.issueToken}
              </Button>
            ) : !creating && screen === undefined ? (
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

          {screen === 'create' ? null : tokens.length === 0 ? (
            <p className="text-sm text-slate-500">{t.noTokensNotice}</p>
          ) : (
            <div className="overflow-x-auto rounded-lg border border-slate-200">
              <table className="min-w-full divide-y divide-slate-200 text-left text-sm text-slate-700">
                <thead className="bg-slate-50 font-semibold text-slate-900">
                  <tr>
                    <th className="px-4 py-2">{t.tableHeaderDescription}</th>
                    <th className="px-4 py-2">{t.tableHeaderScopes}</th>
                    <th className="px-4 py-2">{t.tokenOwnerLabel}</th>
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
                        <div className="flex max-w-md flex-wrap gap-1">
                          {tok.scopes.map((scope) => (
                            <code
                              key={scope}
                              className="rounded bg-slate-100 px-1.5 py-0.5 text-xs"
                            >
                              {scope}
                            </code>
                          ))}
                        </div>
                      </td>
                      <td className="px-4 py-3 font-mono text-xs">{tok.user_id}</td>
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
                          variant="destructive"
                          disabled={busy || !!tok.revoked_at}
                          onClick={() => setPendingRevoke(tok)}
                        >
                          {tok.revoked_at ? t.revokedLabel : t.revoke}
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
                  <Label htmlFor="token-desc" required>
                    {t.tokenDescriptionLabel}
                  </Label>
                  <Input
                    id="token-desc"
                    required
                    placeholder={t.tokenDescriptionPlaceholder}
                    value={tokenDesc}
                    onChange={(e) => setTokenDesc(e.target.value)}
                    {...validation.fieldProps('description')}
                  />
                  <FieldError form={validation} name="description" />
                </div>
                <div className="grid gap-1.5">
                  <Label htmlFor="token-expiry" required>
                    {t.tokenExpiryLabel}
                  </Label>
                  <Input
                    id="token-expiry"
                    type="number"
                    required
                    min={1}
                    max={personal ? PERSONAL_MAX_EXPIRY_DAYS : undefined}
                    value={tokenExpiry}
                    onChange={(e) => setTokenExpiry(e.target.value)}
                    {...validation.fieldProps('expiryDays')}
                  />
                  <FieldError form={validation} name="expiryDays" />
                </div>
              </div>
              <ApiTokenScopePicker
                t={t}
                selectedScopes={selectedScopes}
                personal={personal}
                onChange={setSelectedScopes}
              />
              <div className="mt-4 flex items-center gap-2">
                <Button
                  type="submit"
                  disabled={busy || !validation.data || selectedScopes.length === 0}
                >
                  {t.issueToken}
                </Button>
                {screen === 'create' ? (
                  <Button
                    variant="outline"
                    nativeButton={false}
                    render={<a href={tenantURL(basePath)} />}
                  >
                    {t.cancel}
                  </Button>
                ) : (
                  <Button
                    type="button"
                    variant="ghost"
                    onClick={() => {
                      setTokenDesc('')
                      setSelectedScopes([])
                      setError('')
                      setCreating(false)
                    }}
                  >
                    {t.cancel}
                  </Button>
                )}
              </div>
            </form>
          ) : null}
          {screen === 'create' && generatedToken ? (
            <Button nativeButton={false} render={<a href={tenantURL(basePath)} />}>
              {t.backToTokens}
            </Button>
          ) : null}
        </div>
      </div>
      {pendingRevoke ? (
        <ConfirmActionDialog
          title={t.revoke}
          target={pendingRevoke.description || pendingRevoke.id}
          description={t.tokenRevokeConfirm}
          confirmLabel={t.revoke}
          busy={busy}
          onClose={() => setPendingRevoke(null)}
          onConfirm={() => void handleRevokeToken(pendingRevoke.id)}
        />
      ) : null}
    </Card>
  )
}
