import { useState } from 'react'
import { discoverWorkloadIssuerMetadata } from '../../api'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { useDictionary } from '../../lib/i18n'
import { adminWorkloadIdentityDictionary } from './AdminWorkloadIdentityPage.i18n'

export function IssuerMetadataAssistance({
  csrfToken,
  issuer,
  onApply,
  disabled,
}: {
  csrfToken: string
  issuer: string
  onApply: (jwksURI: string) => void
  disabled: boolean
}) {
  const t = useDictionary(adminWorkloadIdentityDictionary)
  const [busy, setBusy] = useState(false)
  const [candidate, setCandidate] = useState<{ issuer: string; jwks_uri: string } | null>(null)
  const [error, setError] = useState<{ issuer: string; message: string } | null>(null)
  async function fetchMetadata() {
    setBusy(true)
    setError(null)
    setCandidate(null)
    try {
      setCandidate(await discoverWorkloadIssuerMetadata(csrfToken, issuer.trim()))
    } catch {
      setError({ issuer: issuer.trim(), message: t.issuerMetadataFailed })
    } finally {
      setBusy(false)
    }
  }
  return (
    <div className="space-y-2 md:col-span-2">
      <Button
        type="button"
        variant="outline"
        disabled={busy || disabled || !issuer.trim()}
        onClick={() => void fetchMetadata()}
      >
        {busy ? t.fetchingIssuerMetadata : t.fetchIssuerMetadata}
      </Button>
      {error?.issuer === issuer.trim() ? (
        <Alert variant="destructive">{error.message}</Alert>
      ) : null}
      {candidate?.issuer === issuer.trim() ? (
        <div className="space-y-2">
          <p className="break-all font-mono text-xs">{candidate.jwks_uri}</p>
          <Button
            type="button"
            variant="outline"
            disabled={disabled}
            onClick={() => {
              onApply(candidate.jwks_uri)
              setCandidate(null)
            }}
          >
            {t.applyIssuerMetadata}
          </Button>
        </div>
      ) : null}
    </div>
  )
}
