import { IconCloudUpload } from '@tabler/icons-react'
import { Button } from '../../components/ui/button'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { Select } from '../../components/ui/select'
import { useDictionary } from '../../lib/i18n'
import { provisioningURL } from './AdminApplicationsShared'
import {
  provisioningDictionary,
  type ProvisioningDictionary,
} from './AdminApplicationProvisioning.i18n'
import type { AdminApplication, ProvisioningAuthMethod } from '../../types'
import type { ProvisioningCredentialInput } from '../../api'

// service (M2M) アプリはログイン画面・利用者を持たないため、対象範囲 (scope) が
// 常に空集合になり出す意味を持たない。Application 詳細ページの導線ボタンとして
// 提供し、ページ側の行数予算 (ui-page-lines) を圧迫しないよう単独 export にする。
export function ProvisioningNavButton({ app }: { app: AdminApplication }) {
  const t = useDictionary(provisioningDictionary)
  if (app.kind === 'service') return null
  return (
    <Button variant="outline" asChild>
      <a href={provisioningURL(app.application_id)}>
        <IconCloudUpload size={16} aria-hidden="true" />
        {t.provisioningNavLabel}
      </a>
    </Button>
  )
}

export function formatDate(
  value: string | null | undefined,
  locale: string,
  unknown: string,
): string {
  if (!value) return unknown
  return new Date(value).toLocaleString(locale === 'ja' ? 'ja-JP' : 'en-US')
}

function authMethodOptions(t: ProvisioningDictionary) {
  return [
    { value: 'bearer_token', label: t.authMethodBearerToken },
    { value: 'oauth2_client_credentials', label: t.authMethodOAuth2ClientCredentials },
  ]
}

export type CredentialFields = {
  bearerToken: string
  oauthTokenURL: string
  oauthClientID: string
  oauthClientSecret: string
  oauthScope: string
}

export function emptyCredentialFields(): CredentialFields {
  return {
    bearerToken: '',
    oauthTokenURL: '',
    oauthClientID: '',
    oauthClientSecret: '',
    oauthScope: '',
  }
}

export function credentialInputFrom(
  authMethod: ProvisioningAuthMethod,
  f: CredentialFields,
): ProvisioningCredentialInput {
  return {
    auth_method: authMethod,
    bearer_token: f.bearerToken || undefined,
    oauth2_token_url: f.oauthTokenURL || undefined,
    oauth2_client_id: f.oauthClientID || undefined,
    oauth2_client_secret: f.oauthClientSecret || undefined,
    oauth2_scope: f.oauthScope || undefined,
  }
}

export function CredentialFieldsEditor({
  authMethod,
  setAuthMethod,
  fields,
  setFields,
}: {
  authMethod: ProvisioningAuthMethod
  setAuthMethod: (m: ProvisioningAuthMethod) => void
  fields: CredentialFields
  setFields: (f: CredentialFields) => void
}) {
  const t = useDictionary(provisioningDictionary)
  return (
    <div className="grid gap-4">
      <div className="grid gap-1.5">
        <Label>{t.authMethodFieldLabel}</Label>
        <Select
          value={authMethod}
          onValueChange={(v) => setAuthMethod(v as ProvisioningAuthMethod)}
          options={authMethodOptions(t)}
        />
      </div>
      {authMethod === 'bearer_token' ? (
        <div className="grid gap-1.5">
          <Label>{t.bearerTokenFieldLabel}</Label>
          <Input
            type="password"
            value={fields.bearerToken}
            onChange={(e) => setFields({ ...fields, bearerToken: e.target.value })}
          />
        </div>
      ) : (
        <>
          <p className="text-xs text-amber-700">{t.oauth2GapNotice}</p>
          <div className="grid gap-4 sm:grid-cols-2">
            <div className="grid gap-1.5">
              <Label>{t.oauth2TokenUrlFieldLabel}</Label>
              <Input
                value={fields.oauthTokenURL}
                onChange={(e) => setFields({ ...fields, oauthTokenURL: e.target.value })}
              />
            </div>
            <div className="grid gap-1.5">
              <Label>{t.oauth2ClientIdFieldLabel}</Label>
              <Input
                value={fields.oauthClientID}
                onChange={(e) => setFields({ ...fields, oauthClientID: e.target.value })}
              />
            </div>
            <div className="grid gap-1.5">
              <Label>{t.oauth2ClientSecretFieldLabel}</Label>
              <Input
                type="password"
                value={fields.oauthClientSecret}
                onChange={(e) => setFields({ ...fields, oauthClientSecret: e.target.value })}
              />
            </div>
            <div className="grid gap-1.5">
              <Label>{t.oauth2ScopeFieldLabel}</Label>
              <Input
                value={fields.oauthScope}
                onChange={(e) => setFields({ ...fields, oauthScope: e.target.value })}
              />
            </div>
          </div>
        </>
      )}
    </div>
  )
}
