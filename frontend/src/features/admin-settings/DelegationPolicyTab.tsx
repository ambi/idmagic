import { IconPencil } from '@tabler/icons-react'
import { type FormEvent, useState } from 'react'
import { AuthenticationAPIError, updateAdminSettings } from '../../api'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { Toast } from '../../components/ui/toast'
import { useDictionary } from '../../lib/i18n'
import type { AdminSettings } from '../../types'
import { adminSettingsDictionary } from './AdminSettingsPage.i18n'
import { ReadSetting } from './AdminSettingsShared'

export function DelegationPolicyTab({
  csrfToken,
  settings,
  onSaved,
}: {
  csrfToken: string
  settings: AdminSettings
  onSaved: (next: AdminSettings) => void
}) {
  const ceiling = settings.max_delegation_depth_default
  const [savedDepth, setSavedDepth] = useState(settings.max_delegation_depth)
  const [depth, setDepth] = useState(settings.max_delegation_depth?.toString() ?? '')
  const [editing, setEditing] = useState(false)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const t = useDictionary(adminSettingsDictionary)

  async function handleSave(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setError('')
    setNotice('')

    const trimmed = depth.trim()
    const requested = trimmed === '' ? 0 : Number(trimmed)
    if (
      !Number.isInteger(requested) ||
      requested < 0 ||
      requested > ceiling ||
      (requested !== 0 && requested < 1)
    ) {
      setError(t.maxDelegationDepthRangeError)
      return
    }
    if (requested === (savedDepth ?? 0)) {
      setNotice(t.noChangesNotice)
      return
    }

    setSaving(true)
    try {
      const next = await updateAdminSettings(csrfToken, { max_delegation_depth: requested })
      onSaved(next)
      setSavedDepth(next.max_delegation_depth)
      setDepth(next.max_delegation_depth?.toString() ?? '')
      setEditing(false)
      setNotice(t.delegationPolicyUpdatedNotice)
    } catch (cause) {
      setError(
        cause instanceof AuthenticationAPIError
          ? cause.message
          : t.delegationPolicyUpdateFailedError,
      )
    } finally {
      setSaving(false)
    }
  }

  return (
    <Card className="p-6">
      <header>
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div>
            <h2 className="text-base font-semibold text-slate-900">{t.delegationPolicyHeading}</h2>
            <p className="mt-1 text-sm text-slate-600">{t.delegationPolicySubheading}</p>
          </div>
          {!editing ? (
            <Button type="button" variant="outline" onClick={() => setEditing(true)}>
              <IconPencil size={16} aria-hidden="true" />
              {t.edit}
            </Button>
          ) : null}
        </div>
        <p className="mt-3 text-xs text-slate-500">{t.delegationPolicyTighteningWarning}</p>
      </header>
      <div className="mt-5 grid gap-4">
        {error ? <Alert variant="destructive">{error}</Alert> : null}
        <Toast message={notice} onDismiss={() => setNotice('')} />
        {!editing ? (
          <dl className="grid gap-3 sm:grid-cols-3">
            <ReadSetting
              label={t.systemDelegationDepthLabel}
              value={`${ceiling}${t.delegationDepthSuffix}`}
            />
            <ReadSetting
              label={t.effectiveDelegationDepthLabel}
              value={`${savedDepth ?? ceiling}${t.delegationDepthSuffix}`}
            />
            <ReadSetting
              label={t.delegationDepthSourceLabel}
              value={
                savedDepth === undefined
                  ? t.delegationDepthInheritedValue
                  : t.delegationDepthOverrideValue
              }
            />
          </dl>
        ) : (
          <form onSubmit={handleSave} className="grid gap-4" noValidate>
            <div className="grid max-w-sm gap-1.5">
              <Label htmlFor="max-delegation-depth">{t.maxDelegationDepthFieldLabel}</Label>
              <Input
                id="max-delegation-depth"
                type="number"
                min={1}
                max={ceiling}
                value={depth}
                placeholder={ceiling.toString()}
                onChange={(event) => setDepth(event.target.value)}
              />
              <p className="text-xs text-slate-500">
                {t.maxDelegationDepthHint.replace('{max}', ceiling.toString())}
              </p>
            </div>
            <div className="flex items-center gap-2">
              <Button type="submit" disabled={saving}>
                {saving ? t.saving : t.save}
              </Button>
              <Button
                type="button"
                variant="ghost"
                disabled={saving}
                onClick={() => {
                  setDepth(savedDepth?.toString() ?? '')
                  setEditing(false)
                  setError('')
                }}
              >
                {t.cancel}
              </Button>
            </div>
          </form>
        )}
      </div>
    </Card>
  )
}
