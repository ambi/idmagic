import { type FormEvent, useState } from 'react'
import { AuthenticationAPIError, tenantURL, updateAccountProfile } from '../../api'
import { useDictionary } from '../../lib/i18n'
import { AccountShell } from '../../components/AccountShell'
import { Alert } from '../../components/ui/alert'
import { Toast } from '../../components/ui/toast'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import type { AccountProfile } from '../../types'
import {
  type AttributeDraft,
  draftFromProfile,
  EditableAttributeGroups,
  ProfileAttributeGroups,
  ProfileReadField,
  textToValue,
} from './AccountProfileAttributes'
import { accountProfileDictionary } from './AccountProfilePage.i18n'

export { draftFromProfile, textToValue, valueToText } from './AccountProfileAttributes'

export function AccountProfilePage({
  profile,
  isAdmin,
}: {
  csrfToken: string
  profile: AccountProfile
  isAdmin: boolean
}) {
  const t = useDictionary(accountProfileDictionary)
  const [notice, setNotice] = useState(() => {
    return new URLSearchParams(window.location.search).get('notice') === 'success' ? t.updated : ''
  })

  return (
    <AccountProfilePresentation
      profile={profile}
      isAdmin={isAdmin}
      notice={notice}
      onDismissNotice={() => setNotice('')}
    />
  )
}

export function AccountProfilePresentation({
  profile,
  isAdmin,
  notice,
  onDismissNotice,
}: {
  profile: AccountProfile
  isAdmin: boolean
  notice: string
  onDismissNotice: () => void
}) {
  const t = useDictionary(accountProfileDictionary)
  return (
    <AccountShell
      active="profile"
      username={profile.preferred_username}
      isAdmin={isAdmin}
      title={t.title}
      description={t.description}
    >
      <div className="grid gap-6">
        <Toast message={notice} onDismiss={onDismissNotice} />

        <Card className="p-5">
          <div className="flex flex-wrap items-start justify-between gap-4">
            <div>
              <h2 className="text-base font-semibold text-slate-900">{t.profile}</h2>
              <p className="mt-1 text-sm text-slate-600">{t.profileDescription}</p>
            </div>
            <Button
              variant="outline"
              nativeButton={false}
              render={<a href={tenantURL('/account/profile/edit')} />}
            >
              {t.edit}
            </Button>
          </div>

          <dl className="mt-5 grid gap-3 sm:grid-cols-2">
            <ProfileReadField label={t.displayName} value={profile.name ?? t.notSet} />
            <ProfileReadField label={t.givenName} value={profile.given_name ?? t.notSet} />
            <ProfileReadField label={t.familyName} value={profile.family_name ?? t.notSet} />
            <ProfileReadField
              label={t.email}
              value={profile.email ?? t.notSet}
              action={
                <a
                  href={tenantURL('/account/emails')}
                  className="text-xs font-semibold text-blue-600 hover:text-blue-700 hover:underline"
                >
                  {t.change}
                </a>
              }
            />
            <ProfileReadField
              label={t.emailVerification}
              value={profile.email_verified ? t.verified : t.unverified}
            />
            <ProfileReadField
              label={t.mfa}
              value={profile.mfa_enrolled ? t.enrolled : t.notEnrolled}
            />
            <ProfileReadField label={t.status} value={profile.status} />
          </dl>
          <div className="mt-5 grid gap-4">
            <ProfileAttributeGroups
              defs={profile.readable_attributes}
              values={profile.attributes}
            />
          </div>
        </Card>
      </div>
    </AccountShell>
  )
}

export function AccountProfileEditPage({
  csrfToken,
  profile,
  isAdmin,
}: {
  csrfToken: string
  profile: AccountProfile
  isAdmin: boolean
}) {
  const t = useDictionary(accountProfileDictionary)
  const [name, setName] = useState(profile.name ?? '')
  const [givenName, setGivenName] = useState(profile.given_name ?? '')
  const [familyName, setFamilyName] = useState(profile.family_name ?? '')
  const [attributes, setAttributes] = useState<AttributeDraft>(draftFromProfile(profile))
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  async function handleSave(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setSaving(true)
    setError('')
    try {
      const nextAttributes: AccountProfile['attributes'] = {}
      for (const def of profile.editable_attributes) {
        const value = textToValue(def, attributes[def.key] ?? '')
        if (value) {
          nextAttributes[def.key] = value
        }
      }
      await updateAccountProfile(csrfToken, {
        name: name.trim() || undefined,
        given_name: givenName.trim() || undefined,
        family_name: familyName.trim() || undefined,
        attributes: nextAttributes,
      })
      window.location.assign(`${tenantURL('/account/profile')}?notice=success`)
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.updateFailed)
      setSaving(false)
    }
  }

  return (
    <AccountProfileEditPresentation
      profile={profile}
      isAdmin={isAdmin}
      name={name}
      givenName={givenName}
      familyName={familyName}
      attributes={attributes}
      saving={saving}
      error={error}
      onNameChange={setName}
      onGivenNameChange={setGivenName}
      onFamilyNameChange={setFamilyName}
      onAttributeChange={(key, next) => setAttributes((current) => ({ ...current, [key]: next }))}
      onSubmit={handleSave}
    />
  )
}

export function AccountProfileEditPresentation({
  profile,
  isAdmin,
  name,
  givenName,
  familyName,
  attributes,
  saving,
  error,
  onNameChange,
  onGivenNameChange,
  onFamilyNameChange,
  onAttributeChange,
  onSubmit,
}: {
  profile: AccountProfile
  isAdmin: boolean
  name: string
  givenName: string
  familyName: string
  attributes: AttributeDraft
  saving: boolean
  error: string
  onNameChange: (value: string) => void
  onGivenNameChange: (value: string) => void
  onFamilyNameChange: (value: string) => void
  onAttributeChange: (key: string, value: string) => void
  onSubmit: (event: FormEvent<HTMLFormElement>) => void
}) {
  const t = useDictionary(accountProfileDictionary)
  return (
    <AccountShell
      active="profile"
      username={profile.preferred_username}
      isAdmin={isAdmin}
      title={t.editTitle}
      description={t.editDescription}
    >
      <div className="grid gap-6">
        {error ? <Alert variant="destructive">{error}</Alert> : null}

        <Card className="p-5">
          <div className="flex items-center gap-3">
            <a
              href={tenantURL('/account/profile')}
              className="inline-flex size-9 items-center justify-center rounded-lg border border-slate-200 bg-white text-slate-700 transition hover:bg-slate-50 hover:text-slate-900"
              aria-label={t.back}
            >
              <svg
                xmlns="http://www.w3.org/2000/svg"
                className="h-5 w-5"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <title>{t.backIcon}</title>
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M10 19l-7-7m0 0l7-7m-7 7h18"
                />
              </svg>
            </a>
            <h2 className="text-base font-semibold text-slate-900">{t.editHeading}</h2>
          </div>

          <form onSubmit={onSubmit} className="mt-5 grid gap-4">
            <div className="grid gap-1.5">
              <Label htmlFor="name">{t.displayName}</Label>
              <Input
                id="name"
                value={name}
                onChange={(event) => onNameChange(event.target.value)}
              />
            </div>
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="grid gap-1.5">
                <Label htmlFor="given-name">{t.givenName} (given_name)</Label>
                <Input
                  id="given-name"
                  value={givenName}
                  onChange={(event) => onGivenNameChange(event.target.value)}
                />
              </div>
              <div className="grid gap-1.5">
                <Label htmlFor="family-name">{t.familyName} (family_name)</Label>
                <Input
                  id="family-name"
                  value={familyName}
                  onChange={(event) => onFamilyNameChange(event.target.value)}
                />
              </div>
            </div>

            <EditableAttributeGroups
              defs={profile.editable_attributes}
              values={attributes}
              onChange={onAttributeChange}
            />

            <div className="flex items-center gap-2">
              <Button type="submit" disabled={saving}>
                {saving ? t.saving : t.save}
              </Button>
              <Button
                type="button"
                variant="ghost"
                disabled={saving}
                nativeButton={false}
                render={<a href={tenantURL('/account/profile')} />}
              >
                {t.cancel}
              </Button>
            </div>
          </form>
        </Card>
      </div>
    </AccountShell>
  )
}
