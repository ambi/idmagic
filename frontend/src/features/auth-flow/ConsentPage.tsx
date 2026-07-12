import {
  IconArrowRight,
  IconCheck,
  IconClock,
  IconId,
  IconMail,
  IconRefresh,
  IconShieldCheck,
  IconUser,
  IconX,
} from '@tabler/icons-react'
import { useState } from 'react'
import { AuthenticationAPIError, continueBrowserFlow, submitConsent } from '../../api'
import { AuthShell } from '../../components/AuthShell'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { useDictionary } from '../../lib/i18n'
import type { ConsentDetailView } from '../../types'
import { consentPageDictionary } from './ConsentPage.i18n'

export function ConsentPage({
  csrfToken,
  clientName,
  scopes,
  authorizationDetails = [],
}: {
  csrfToken: string
  clientName: string
  scopes: string[]
  authorizationDetails?: ConsentDetailView[]
}) {
  const t = useDictionary(consentPageDictionary)
  const scopeDetails: Record<string, { label: string; description: string; icon: typeof IconId }> =
    {
      openid: { label: t.scopeOpenidLabel, description: t.scopeOpenidDescription, icon: IconId },
      profile: {
        label: t.scopeProfileLabel,
        description: t.scopeProfileDescription,
        icon: IconUser,
      },
      email: {
        label: t.scopeEmailLabel,
        description: t.scopeEmailDescription,
        icon: IconMail,
      },
      offline_access: {
        label: t.scopeOfflineAccessLabel,
        description: t.scopeOfflineAccessDescription,
        icon: IconRefresh,
      },
    }
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)

  async function handleConsent(action: 'allow' | 'deny') {
    setSubmitting(true)
    setError('')
    try {
      continueBrowserFlow(await submitConsent(csrfToken, action))
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.consentError)
      setSubmitting(false)
    }
  }

  return (
    <AuthShell asideTitle={t.asideTitle} asideText={t.asideText}>
      <div className="flex flex-col gap-6">
        <header className="flex flex-col gap-2">
          <p className="eyebrow">{t.eyebrow}</p>
          <h2 className="page-title">{t.title}</h2>
          <p className="page-description">{t.description}</p>
        </header>

        <Card className="overflow-hidden">
          <div className="flex items-center gap-4 border-b border-slate-200 bg-slate-50/70 p-4">
            <div className="flex size-12 shrink-0 items-center justify-center rounded-xl border border-blue-100 bg-blue-50 text-sm font-bold text-blue-700">
              {clientName.slice(0, 2).toUpperCase()}
            </div>
            <div className="min-w-0">
              <p className="text-xs font-medium text-slate-500">{t.requestingApplication}</p>
              <p className="truncate font-semibold text-slate-950">{clientName}</p>
            </div>
            <span className="ml-auto flex shrink-0 items-center gap-1.5 rounded-full bg-emerald-50 px-2.5 py-1 text-[0.68rem] font-bold text-emerald-700">
              <IconShieldCheck size={13} aria-hidden="true" />
              {t.registered}
            </span>
          </div>

          <div className="p-4">
            <p className="mb-3 text-xs font-bold uppercase tracking-[0.09em] text-slate-500">
              {t.sharedInformation}
            </p>
            <div className="divide-y divide-slate-100">
              {scopes.map((scopeName) => {
                const detail = scopeDetails[scopeName] ?? {
                  label: scopeName,
                  description: t.scopeUnknownDescription,
                  icon: IconCheck,
                }
                const ScopeIcon = detail.icon
                return (
                  <div key={scopeName} className="flex items-start gap-3 py-3 first:pt-1 last:pb-1">
                    <div className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-slate-100 text-slate-600">
                      <ScopeIcon size={18} aria-hidden="true" />
                    </div>
                    <div className="pt-0.5">
                      <p className="text-sm font-semibold text-slate-900">{detail.label}</p>
                      <p className="mt-0.5 text-xs leading-5 text-slate-500">
                        {detail.description}
                      </p>
                    </div>
                  </div>
                )
              })}
            </div>
          </div>
        </Card>

        {authorizationDetails.length > 0 ? (
          <Card className="overflow-hidden border-blue-200/70">
            <div className="border-b border-slate-200 bg-blue-50/50 p-4">
              <p className="text-xs font-bold uppercase tracking-[0.09em] text-blue-700">
                {t.fineGrainedPermissions}
              </p>
              <p className="mt-0.5 text-xs leading-5 text-slate-500">
                {t.fineGrainedPermissionsDescription}
              </p>
            </div>
            <div className="divide-y divide-slate-100 p-4">
              {authorizationDetails.map((detail) => (
                <div
                  key={`${detail.type}-${detail.summary}`}
                  className="flex items-start gap-3 py-3 first:pt-1 last:pb-1"
                >
                  <div className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-blue-100 text-blue-700">
                    <IconShieldCheck size={18} aria-hidden="true" />
                  </div>
                  <div className="min-w-0 pt-0.5">
                    <p className="text-sm font-semibold text-slate-900">
                      {detail.description || detail.type}
                    </p>
                    <p className="mt-0.5 text-xs leading-5 text-slate-700">{detail.summary}</p>
                    {detail.lines && detail.lines.length > 0 ? (
                      <ul className="mt-1.5 flex flex-col gap-0.5 text-[0.7rem] leading-5 text-slate-500">
                        {detail.lines.map((line) => (
                          <li key={line} className="font-mono">
                            {line}
                          </li>
                        ))}
                      </ul>
                    ) : null}
                  </div>
                </div>
              ))}
            </div>
          </Card>
        ) : null}

        <div className="flex gap-3 rounded-xl border border-amber-200/80 bg-amber-50/70 p-3.5 text-xs leading-5 text-amber-950">
          <IconClock className="mt-0.5 shrink-0 text-amber-700" size={17} aria-hidden="true" />
          <p>{t.retentionNote}</p>
        </div>

        <ConsentActionsPresentation
          error={error}
          submitting={submitting}
          onConsent={handleConsent}
        />
      </div>
    </AuthShell>
  )
}

export function ConsentActionsPresentation({
  error,
  submitting,
  onConsent,
}: {
  error: string
  submitting: boolean
  onConsent: (action: 'allow' | 'deny') => void
}) {
  const t = useDictionary(consentPageDictionary)
  return (
    <>
      {error ? (
        <p role="alert" className="text-sm font-medium text-red-700">
          {error}
        </p>
      ) : null}
      <div className="flex flex-col gap-2.5">
        <Button
          type="button"
          size="lg"
          className="tenant-primary-cta"
          disabled={submitting}
          onClick={() => onConsent('allow')}
        >
          {submitting ? t.processing : t.allow}
          <IconArrowRight size={18} aria-hidden="true" />
        </Button>
        <Button
          type="button"
          size="lg"
          variant="ghost"
          disabled={submitting}
          onClick={() => onConsent('deny')}
        >
          <IconX size={17} aria-hidden="true" />
          {t.deny}
        </Button>
      </div>
    </>
  )
}
