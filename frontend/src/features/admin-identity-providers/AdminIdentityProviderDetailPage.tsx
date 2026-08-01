import { IconArrowLeft, IconPencil } from '@tabler/icons-react'
import { useState } from 'react'
import {
  AuthenticationAPIError,
  deleteIdentityProviderConnection,
  runIdentityProviderAction,
  tenantURL,
  testIdentityProviderConnection,
  type IdentityProviderConnection,
  type IdentityProviderConnectionTestResult,
} from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Toast } from '../../components/ui/toast'
import { useDictionary, useLocale } from '../../lib/i18n'
import { identityProvidersDictionary } from './AdminIdentityProvidersPage.i18n'
import {
  DeleteConnectionDialog,
  StatusBadge,
  TestResultBanner,
} from './AdminIdentityProvidersShared'

function formatDateTime(value: string | undefined, locale: string): string {
  if (!value) return ''
  return new Date(value).toLocaleString(locale === 'ja' ? 'ja-JP' : 'en-US')
}

export function AdminIdentityProviderDetailPage({
  csrfToken,
  actorUsername,
  connection: initial,
}: {
  csrfToken: string
  actorUsername?: string
  connection: IdentityProviderConnection
}) {
  const [connection, setConnection] = useState(initial)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const [testResult, setTestResult] = useState<IdentityProviderConnectionTestResult | null>(null)
  const [confirmDelete, setConfirmDelete] = useState(false)
  const t = useDictionary(identityProvidersDictionary)
  const { locale } = useLocale()

  async function run(action: () => Promise<void>) {
    setBusy(true)
    setError('')
    try {
      await action()
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.failed)
    } finally {
      setBusy(false)
    }
  }

  async function toggleStatus() {
    const activate = connection.status === 'disabled'
    await run(async () => {
      await runIdentityProviderAction(csrfToken, connection.id, activate ? 'activate' : 'disable')
      setConnection((current) => ({ ...current, status: activate ? 'active' : 'disabled' }))
      setNotice(activate ? t.activated : t.disabled)
    })
  }

  async function test() {
    await run(async () => {
      const result = await testIdentityProviderConnection(csrfToken, connection.id)
      setTestResult(result)
    })
  }

  async function remove() {
    await run(async () => {
      await deleteIdentityProviderConnection(csrfToken, connection.id)
      window.location.href = tenantURL('/admin/identity-providers')
    })
  }

  function requestDelete() {
    if (connection.status === 'active') {
      setConfirmDelete(true)
    } else {
      void remove()
    }
  }

  return (
    <>
      <AdminShell
        active="identity-providers"
        actorUsername={actorUsername}
        title={connection.display_name}
        description={connection.issuer}
        actions={
          <div className="flex flex-wrap items-center gap-2">
            <a
              href={tenantURL('/admin/identity-providers')}
              className="inline-flex items-center gap-1.5 rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm font-semibold text-slate-700 transition-colors hover:bg-slate-50"
            >
              <IconArrowLeft size={16} aria-hidden="true" />
              {t.backToList}
            </a>
            <Button
              variant="outline"
              nativeButton={false}
              render={
                <a
                  href={tenantURL(
                    `/admin/identity-providers/${encodeURIComponent(connection.id)}/edit`,
                  )}
                />
              }
            >
              <IconPencil size={16} aria-hidden="true" />
              {t.edit}
            </Button>
            <Button variant="outline" disabled={busy} onClick={() => void test()}>
              {t.test}
            </Button>
            <Button variant="outline" disabled={busy} onClick={() => void toggleStatus()}>
              {connection.status === 'active' ? t.disable : t.activate}
            </Button>
            <Button variant="destructive" disabled={busy} onClick={requestDelete}>
              {t.delete}
            </Button>
          </div>
        }
      >
        {error ? <Alert variant="destructive">{error}</Alert> : null}
        <Toast message={notice} onDismiss={() => setNotice('')} />
        {testResult ? (
          <TestResultBanner result={testResult} onDismiss={() => setTestResult(null)} />
        ) : null}

        <Card className="overflow-hidden">
          <div className="border-b border-slate-200 p-5">
            <div className="flex items-center gap-3">
              <h2 className="text-lg font-semibold text-slate-950">{connection.display_name}</h2>
              <StatusBadge status={connection.status} labels={t} />
            </div>
          </div>
          <dl className="grid gap-x-8 gap-y-4 p-5 sm:grid-cols-2">
            <DetailRow label={t.displayName} value={connection.display_name} />
            <DetailRow label={t.protocol} value={connection.protocol.toUpperCase()} />
            <DetailRow label={t.issuer} value={connection.issuer} mono />
            {connection.protocol === 'oidc' ? (
              <>
                <DetailRow label={t.clientId} value={connection.client_id ?? ''} mono />
                <DetailRow
                  label={t.secretValue}
                  value={
                    connection.client_secret_configured ? t.secretConfigured : t.secretNotConfigured
                  }
                />
                <DetailRow
                  label={t.authorizationEndpoint}
                  value={connection.authorization_endpoint ?? ''}
                  mono
                />
                <DetailRow label={t.tokenEndpoint} value={connection.token_endpoint ?? ''} mono />
                <DetailRow label={t.jwksUri} value={connection.jwks_uri ?? ''} mono />
              </>
            ) : (
              <>
                <DetailRow label={t.samlEntityId} value={connection.saml_entity_id ?? ''} mono />
                <DetailRow label={t.samlSsoUrl} value={connection.saml_sso_url ?? ''} mono />
                <DetailRow
                  label={t.samlCertificates}
                  value={String((connection.saml_signing_certificates ?? []).length)}
                />
              </>
            )}
            <DetailRow
              label={t.linkingPolicy}
              value={connection.linking_policy === 'none' ? t.linkingNone : t.linkingVerifiedEmail}
            />
            <DetailRow
              label={t.jitProvisioning}
              value={connection.jit_provisioning ? t.secretConfigured : t.secretNotConfigured}
            />
          </dl>

          <div className="border-t border-slate-200 p-5">
            <h3 className="text-xs font-bold uppercase tracking-[0.1em] text-slate-400">
              Claim mapping
            </h3>
            <dl className="mt-3 grid gap-x-8 gap-y-3 sm:grid-cols-2">
              <DetailRow label={t.subjectClaim} value={connection.claim_mapping.subject} mono />
              <DetailRow label={t.usernameClaim} value={connection.claim_mapping.username} mono />
              <DetailRow label={t.emailClaim} value={connection.claim_mapping.email ?? ''} mono />
              <DetailRow label={t.nameClaim} value={connection.claim_mapping.name ?? ''} mono />
            </dl>
          </div>

          <div className="flex flex-wrap gap-x-8 gap-y-2 border-t border-slate-200 bg-slate-50/60 px-5 py-3 text-xs text-slate-500">
            <span>{formatDateTime(connection.created_at, locale)}</span>
            <span>{formatDateTime(connection.updated_at, locale)}</span>
          </div>
        </Card>
      </AdminShell>

      {confirmDelete ? (
        <DeleteConnectionDialog
          connection={connection}
          busy={busy}
          onClose={() => setConfirmDelete(false)}
          onConfirm={() => void remove()}
        />
      ) : null}
    </>
  )
}

function DetailRow({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div>
      <dt className="text-xs font-semibold uppercase tracking-wide text-slate-400">{label}</dt>
      <dd
        className={
          mono ? 'mt-1 break-all font-mono text-sm text-slate-800' : 'mt-1 text-sm text-slate-800'
        }
      >
        {value || '—'}
      </dd>
    </div>
  )
}
