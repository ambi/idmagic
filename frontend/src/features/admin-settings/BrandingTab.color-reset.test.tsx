import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { BrandingTab } from './BrandingTab'

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

    render(<BrandingTab csrfToken="csrf-token" />)

    expect(await screen.findByText('現在値: #123456')).toBeInTheDocument()
    expect(screen.getByText('現在値: #abcdef')).toBeInTheDocument()

    fireEvent.click(screen.getAllByRole('button', { name: '既定に戻す' })[0])
    fireEvent.click(screen.getAllByRole('button', { name: '既定に戻す' })[1])
    expect(screen.getAllByText('未設定（IdMagic の既定色を使用）')).toHaveLength(2)

    fireEvent.click(screen.getByRole('button', { name: '保存' }))

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
    expect(screen.getAllByText('未設定（IdMagic の既定色を使用）')).toHaveLength(2)
  })
})
