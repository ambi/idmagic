import { describe, it, expect, vi } from 'vitest'
import { screen, fireEvent } from '@testing-library/react'
import { renderWithRouter } from '../../test/renderWithRouter'
import { AccountDataPresentation } from './AccountDataPage'

describe('AccountDataPresentation', () => {
  const baseProps = {
    username: 'taro',
    isAdmin: false,
    downloading: false,
    error: '',
    onExport: vi.fn(),
  }

  it('calls onExport when the download button is clicked', async () => {
    const onExport = vi.fn()
    await renderWithRouter(<AccountDataPresentation {...baseProps} onExport={onExport} />)
    fireEvent.click(screen.getByRole('button', { name: /データをダウンロード/ }))
    expect(onExport).toHaveBeenCalledTimes(1)
  })

  it('shows a generating label and disables the button while downloading', async () => {
    await renderWithRouter(<AccountDataPresentation {...baseProps} downloading />)
    expect(screen.getByRole('button', { name: /生成中/ })).toBeDisabled()
  })

  it('shows an error message when present', async () => {
    await renderWithRouter(
      <AccountDataPresentation {...baseProps} error="データをエクスポートできませんでした。" />,
    )
    expect(screen.getByText('データをエクスポートできませんでした。')).toBeInTheDocument()
  })
})
