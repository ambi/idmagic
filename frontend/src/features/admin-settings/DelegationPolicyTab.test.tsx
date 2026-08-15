import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, mock } from 'bun:test'
import { LocaleProvider } from '../../lib/i18n'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import type { AdminSettings } from '../../types'
import { adminSettingsDictionary } from './AdminSettingsPage.i18n'
import { DelegationPolicyTab } from './DelegationPolicyTab'

const t = adminSettingsDictionary.en

const settings: AdminSettings = {
  tenant_id: 'tenant-1',
  realm: 'acme',
  display_name: 'Acme',
  password_policy_defaults: { min_length: 8, max_length: 64, history_depth: 5 },
  max_delegation_depth_default: 3,
  supported_locales: ['ja', 'en'],
}

const response = (status: number, body: unknown = {}) => ({
  ok: status >= 200 && status < 300,
  status,
  json: mock().mockResolvedValue(body),
})

describe('DelegationPolicyTab', () => {
  afterEach(() => restoreGlobals())

  // REQ-TENANCY-021: 空欄はシステム既定の継承を表し、厳しい上書きだけを保存できる。
  it('saves a tighter delegation depth and can clear it back to the system default', async () => {
    const fetch = mock()
      .mockResolvedValueOnce(response(200, { ...settings, max_delegation_depth: 1 }))
      .mockResolvedValueOnce(response(200, settings))
    stubGlobal('fetch', fetch)

    const onSaved = mock()
    render(
      <LocaleProvider initialLocale="en">
        <DelegationPolicyTab csrfToken="csrf-token" settings={settings} onSaved={onSaved} />
      </LocaleProvider>,
    )

    expect(screen.getByText(t.delegationDepthInheritedValue)).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: t.edit }))
    fireEvent.change(screen.getByLabelText(t.maxDelegationDepthFieldLabel), {
      target: { value: '1' },
    })
    fireEvent.click(screen.getByRole('button', { name: t.save }))

    await waitFor(() => expect(fetch).toHaveBeenCalledTimes(1))
    expect(JSON.parse((fetch as any).mock.calls[0][1].body)).toEqual({ max_delegation_depth: 1 })
    expect(await screen.findByText(t.delegationPolicyUpdatedNotice)).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: t.edit }))
    fireEvent.change(screen.getByLabelText(t.maxDelegationDepthFieldLabel), {
      target: { value: '' },
    })
    fireEvent.click(screen.getByRole('button', { name: t.save }))

    await waitFor(() => expect(fetch).toHaveBeenCalledTimes(2))
    expect(JSON.parse((fetch as any).mock.calls[1][1].body)).toEqual({ max_delegation_depth: 0 })
  })

  // REQ-TENANCY-021: システム上限を超える値は往復前に presentation logic で拒否する。
  it('rejects a value above the system ceiling without sending it', () => {
    const fetch = mock()
    stubGlobal('fetch', fetch)

    render(
      <LocaleProvider initialLocale="en">
        <DelegationPolicyTab csrfToken="csrf-token" settings={settings} onSaved={mock()} />
      </LocaleProvider>,
    )

    fireEvent.click(screen.getByRole('button', { name: t.edit }))
    fireEvent.change(screen.getByLabelText(t.maxDelegationDepthFieldLabel), {
      target: { value: '4' },
    })
    fireEvent.click(screen.getByRole('button', { name: t.save }))

    expect(screen.getByText(t.maxDelegationDepthRangeError)).toBeInTheDocument()
    expect(fetch).not.toHaveBeenCalled()
  })
})
