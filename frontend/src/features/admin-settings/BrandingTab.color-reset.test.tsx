import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { LocaleProvider } from '../../lib/i18n'
import { BrandingTab } from './BrandingTab'
import { brandingTabDictionary } from './BrandingTab.i18n'

const t = brandingTabDictionary.en

const response = (status: number, body: unknown = {}) => ({
  ok: status >= 200 && status < 300,
  status,
  json: vi.fn().mockResolvedValue(body),
})

describe('BrandingTab color reset', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('resets each configured color to unset and saves empty color values', async () => {
    const fetch = vi
      .fn()
      .mockResolvedValueOnce(
        response(200, {
          primary_color: '#123456',
          accent_color: '#abcdef',
        }),
      )
      .mockResolvedValueOnce(response(200, {}))
    vi.stubGlobal('fetch', fetch)

    render(
      <LocaleProvider initialLocale="en">
        <BrandingTab csrfToken="csrf-token" />
      </LocaleProvider>,
    )

    expect(await screen.findByText(`${t.currentValuePrefix}#123456`)).toBeInTheDocument()
    expect(screen.getByText(`${t.currentValuePrefix}#abcdef`)).toBeInTheDocument()

    fireEvent.click(screen.getAllByRole('button', { name: t.resetToDefault })[0])
    fireEvent.click(screen.getAllByRole('button', { name: t.resetToDefault })[1])
    expect(screen.getAllByText(t.colorUnsetNotice)).toHaveLength(2)

    fireEvent.click(screen.getByRole('button', { name: t.save }))

    await waitFor(() => expect(fetch).toHaveBeenCalledTimes(2))
    expect(fetch.mock.calls[1]).toEqual([
      '/api/admin/tenant/branding',
      expect.objectContaining({
        method: 'PUT',
        body: JSON.stringify({
          product_name: '',
          primary_color: '',
          accent_color: '',
          footer_link_1: { label: '', url: '' },
          footer_link_2: { label: '', url: '' },
          footer_text: '',
        }),
      }),
    ])
    expect(screen.getAllByText(t.colorUnsetNotice)).toHaveLength(2)
  })
})
