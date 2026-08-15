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

const SECONDS_PER_DAY = 86400

// API は秒で扱うが、上限が 7,776,000 秒では入力を誤りやすいので UI は日で受ける。
export function daysFromSeconds(seconds?: number): number {
  if (!seconds || seconds <= 0) return 0
  return Math.round(seconds / SECONDS_PER_DAY)
}

// formatLifetime は読み取り表示の文字列。日で割り切れない値 (API から直接設定された値) は
// 丸めずに秒のまま示し、実際に効いている値を偽らない。
export function formatLifetime(
  seconds: number | undefined,
  daySuffix: string,
  secondSuffix: string,
  disabled: string,
): string {
  if (!seconds || seconds <= 0) return disabled
  if (seconds % SECONDS_PER_DAY === 0) return `${seconds / SECONDS_PER_DAY}${daySuffix}`
  return `${seconds}${secondSuffix}`
}

// TrustedDeviceTab はテナントの trusted_device_max_age_seconds を編集する (wi-91)。
// 委譲深さと違い 0 は「上書きの解除」ではなく「機能無効」であり、それがこの設定の
// 既定かつ最も厳しい状態なので、UI も有効 / 無効として提示する。
export function TrustedDeviceTab({
  csrfToken,
  settings,
  onSaved,
}: {
  csrfToken: string
  settings: AdminSettings
  onSaved: (next: AdminSettings) => void
}) {
  const t = useDictionary(adminSettingsDictionary)
  const ceilingDays = daysFromSeconds(settings.trusted_device_max_age_seconds_ceiling)
  const [savedSeconds, setSavedSeconds] = useState(settings.trusted_device_max_age_seconds ?? 0)
  const [days, setDays] = useState(String(daysFromSeconds(settings.trusted_device_max_age_seconds)))
  const [editing, setEditing] = useState(false)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')

  async function handleSave(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setError('')
    setNotice('')

    const trimmed = days.trim()
    const requestedDays = trimmed === '' ? 0 : Number(trimmed)
    if (!Number.isInteger(requestedDays) || requestedDays < 0 || requestedDays > ceilingDays) {
      setError(t.trustedDeviceRangeError.replace('{max}', String(ceilingDays)))
      return
    }
    const requested = requestedDays * SECONDS_PER_DAY
    if (requested === savedSeconds) {
      setNotice(t.noChangesNotice)
      return
    }

    setSaving(true)
    try {
      const next = await updateAdminSettings(csrfToken, {
        trusted_device_max_age_seconds: requested,
      })
      onSaved(next)
      setSavedSeconds(next.trusted_device_max_age_seconds ?? 0)
      setDays(String(daysFromSeconds(next.trusted_device_max_age_seconds)))
      setEditing(false)
      setNotice(t.trustedDeviceUpdatedNotice)
    } catch (cause) {
      setError(
        cause instanceof AuthenticationAPIError ? cause.message : t.trustedDeviceUpdateFailedError,
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
            <h2 className="text-base font-semibold text-slate-900">{t.trustedDeviceHeading}</h2>
            <p className="mt-1 text-sm text-slate-600">{t.trustedDeviceSubheading}</p>
          </div>
          {!editing ? (
            <Button type="button" variant="outline" onClick={() => setEditing(true)}>
              <IconPencil size={16} aria-hidden="true" />
              {t.edit}
            </Button>
          ) : null}
        </div>
        <p className="mt-3 text-xs text-slate-500">{t.trustedDeviceSecurityWarning}</p>
      </header>
      <div className="mt-5 grid gap-4">
        {error ? <Alert variant="destructive">{error}</Alert> : null}
        <Toast message={notice} onDismiss={() => setNotice('')} />
        {!editing ? (
          <dl className="grid gap-3 sm:grid-cols-3">
            <ReadSetting
              label={t.trustedDeviceStatusLabel}
              value={
                savedSeconds > 0 ? t.trustedDeviceStatusEnabled : t.trustedDeviceStatusDisabled
              }
            />
            <ReadSetting
              label={t.trustedDeviceEffectiveLabel}
              value={formatLifetime(
                savedSeconds,
                t.trustedDeviceDaySuffix,
                t.trustedDeviceSecondSuffix,
                t.trustedDeviceStatusDisabled,
              )}
            />
            <ReadSetting
              label={t.trustedDeviceCeilingLabel}
              value={`${ceilingDays}${t.trustedDeviceDaySuffix}`}
            />
          </dl>
        ) : (
          <form onSubmit={handleSave} className="grid gap-4" noValidate>
            <div className="grid max-w-sm gap-1.5">
              <Label htmlFor="trusted-device-max-age">{t.trustedDeviceFieldLabel}</Label>
              <Input
                id="trusted-device-max-age"
                type="number"
                min={0}
                max={ceilingDays}
                value={days}
                placeholder="0"
                onChange={(event) => setDays(event.target.value)}
              />
              <p className="text-xs text-slate-500">
                {t.trustedDeviceHint.replace('{max}', String(ceilingDays))}
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
                  setDays(String(daysFromSeconds(savedSeconds)))
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
