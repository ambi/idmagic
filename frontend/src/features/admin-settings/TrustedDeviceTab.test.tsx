import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, mock } from 'bun:test'
import { LocaleProvider } from '../../lib/i18n'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import type { AdminSettings } from '../../types'
import { adminSettingsDictionary } from './AdminSettingsPage.i18n'
import { TrustedDeviceTab, daysFromSeconds, formatLifetime } from './TrustedDeviceTab'

const t = adminSettingsDictionary.en

const settings: AdminSettings = {
  tenant_id: 'tenant-1',
  realm: 'acme',
  display_name: 'Acme',
  password_policy_defaults: { min_length: 8, max_length: 64, history_depth: 5 },
  max_delegation_depth_default: 3,
  trusted_device_max_age_seconds_ceiling: 7776000,
  supported_locales: ['ja', 'en'],
}

const response = (status: number, body: unknown = {}) => ({
  ok: status >= 200 && status < 300,
  status,
  json: mock().mockResolvedValue(body),
})

const renderTab = (props: Partial<Parameters<typeof TrustedDeviceTab>[0]> = {}) =>
  render(
    <LocaleProvider initialLocale="en">
      <TrustedDeviceTab csrfToken="csrf-token" settings={settings} onSaved={mock()} {...props} />
    </LocaleProvider>,
  )

describe('daysFromSeconds', () => {
  it('treats an unset or non-positive lifetime as disabled', () => {
    expect(daysFromSeconds(undefined)).toBe(0)
    expect(daysFromSeconds(0)).toBe(0)
    expect(daysFromSeconds(-1)).toBe(0)
  })

  it('converts whole days', () => {
    expect(daysFromSeconds(2592000)).toBe(30)
  })
})

describe('formatLifetime', () => {
  it('shows the disabled label when the feature is off', () => {
    expect(formatLifetime(0, ' days', ' seconds', 'Disabled')).toBe('Disabled')
  })

  // API から直接設定された端数は丸めず、実際に効いている値をそのまま示す。
  it('falls back to seconds when the value is not a whole number of days', () => {
    expect(formatLifetime(2592000, ' days', ' seconds', 'Disabled')).toBe('30 days')
    expect(formatLifetime(90061, ' days', ' seconds', 'Disabled')).toBe('90061 seconds')
  })
})

describe('TrustedDeviceTab', () => {
  afterEach(() => restoreGlobals())

  // wi-91: 未設定は「無効」として提示する (委譲深さのような「継承」ではない)。
  it('presents an unset lifetime as disabled', () => {
    stubGlobal('fetch', mock())
    renderTab()

    // 状態と有効期間の両方が「無効」と読める。
    expect(screen.getAllByText(t.trustedDeviceStatusDisabled)).toHaveLength(2)
    expect(screen.getByText(`90${t.trustedDeviceDaySuffix}`)).toBeInTheDocument()
  })

  it('saves a lifetime in days and converts it to seconds', async () => {
    const fetch = mock().mockResolvedValue(
      response(200, { ...settings, trusted_device_max_age_seconds: 2592000 }),
    )
    stubGlobal('fetch', fetch)
    const onSaved = mock()
    renderTab({ onSaved })

    fireEvent.click(screen.getByRole('button', { name: t.edit }))
    fireEvent.change(screen.getByLabelText(t.trustedDeviceFieldLabel), { target: { value: '30' } })
    fireEvent.click(screen.getByRole('button', { name: t.save }))

    await waitFor(() => expect(fetch).toHaveBeenCalledTimes(1))
    expect(JSON.parse((fetch as any).mock.calls[0][1].body)).toEqual({
      trusted_device_max_age_seconds: 2592000,
    })
    expect(await screen.findByText(t.trustedDeviceUpdatedNotice)).toBeInTheDocument()
    expect(onSaved).toHaveBeenCalledTimes(1)
  })

  // wi-91: 0 は上書きの解除ではなく機能無効。
  it('sends 0 to disable the feature again', async () => {
    const enabled: AdminSettings = { ...settings, trusted_device_max_age_seconds: 2592000 }
    const fetch = mock().mockResolvedValue(response(200, settings))
    stubGlobal('fetch', fetch)
    renderTab({ settings: enabled })

    expect(screen.getByText(t.trustedDeviceStatusEnabled)).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: t.edit }))
    fireEvent.change(screen.getByLabelText(t.trustedDeviceFieldLabel), { target: { value: '0' } })
    fireEvent.click(screen.getByRole('button', { name: t.save }))

    await waitFor(() => expect(fetch).toHaveBeenCalledTimes(1))
    expect(JSON.parse((fetch as any).mock.calls[0][1].body)).toEqual({
      trusted_device_max_age_seconds: 0,
    })
  })

  // wi-91: システム上限 (90 日) 超過は往復前に拒否する。
  it('rejects a lifetime above the system ceiling without sending it', () => {
    const fetch = mock()
    stubGlobal('fetch', fetch)
    renderTab()

    fireEvent.click(screen.getByRole('button', { name: t.edit }))
    fireEvent.change(screen.getByLabelText(t.trustedDeviceFieldLabel), { target: { value: '91' } })
    fireEvent.click(screen.getByRole('button', { name: t.save }))

    expect(screen.getByText(t.trustedDeviceRangeError.replace('{max}', '90'))).toBeInTheDocument()
    expect(fetch).not.toHaveBeenCalled()
  })

  it('rejects a fractional or negative lifetime without sending it', () => {
    const fetch = mock()
    stubGlobal('fetch', fetch)
    renderTab()

    fireEvent.click(screen.getByRole('button', { name: t.edit }))
    for (const value of ['1.5', '-1']) {
      fireEvent.change(screen.getByLabelText(t.trustedDeviceFieldLabel), { target: { value } })
      fireEvent.click(screen.getByRole('button', { name: t.save }))
    }

    expect(screen.getByText(t.trustedDeviceRangeError.replace('{max}', '90'))).toBeInTheDocument()
    expect(fetch).not.toHaveBeenCalled()
  })
})
