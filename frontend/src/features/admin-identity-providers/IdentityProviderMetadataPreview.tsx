import { useState } from 'react'
import {
  previewIdentityProviderMetadata,
  type IdentityProviderMetadataPreview as Metadata,
} from '../../api'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { useDictionary } from '../../lib/i18n'
import { identityProvidersDictionary } from './AdminIdentityProvidersPage.i18n'

export function IdentityProviderMetadataPreview({
  csrfToken,
  issuer,
  onApply,
}: {
  csrfToken: string
  issuer: string
  onApply: (metadata: Metadata) => void
}) {
  const t = useDictionary(identityProvidersDictionary)
  const [busy, setBusy] = useState(false)
  const [candidate, setCandidate] = useState<Metadata | null>(null)
  const [error, setError] = useState('')

  async function fetchMetadata() {
    setBusy(true)
    setError('')
    setCandidate(null)
    try {
      setCandidate(await previewIdentityProviderMetadata(csrfToken, issuer))
    } catch {
      setError(t.metadataFailed)
    } finally {
      setBusy(false)
    }
  }

  return (
    <section
      className="grid gap-3 rounded-lg border border-slate-200 p-4 md:col-span-2"
      aria-label={t.metadataCandidates}
    >
      <p className="text-sm text-slate-600">{t.metadataHelp}</p>
      <div>
        <Button
          type="button"
          variant="outline"
          disabled={busy || !issuer}
          onClick={() => void fetchMetadata()}
        >
          {t.fetchMetadata}
        </Button>
      </div>
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      {candidate ? (
        <>
          <dl className="grid gap-2 break-all text-sm">
            {(
              [
                [t.issuer, candidate.issuer],
                [t.authorizationEndpoint, candidate.authorization_endpoint],
                [t.tokenEndpoint, candidate.token_endpoint],
                [t.jwksUri, candidate.jwks_uri],
              ] as const
            ).map(([label, value]) => (
              <div key={label}>
                <dt className="font-medium">{label}</dt>
                <dd>{value}</dd>
              </div>
            ))}
          </dl>
          <div>
            <Button
              type="button"
              onClick={() => {
                onApply(candidate)
                setCandidate(null)
              }}
            >
              {t.applyCandidates}
            </Button>
          </div>
        </>
      ) : null}
    </section>
  )
}
