import { afterEach, describe, it, expect, mock, spyOn } from 'bun:test'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import { screen, fireEvent, waitFor } from '@testing-library/react'
import { renderWithRouter as renderWithRouterBase } from '../../test/renderWithRouter'
import { AccountDataPage, AccountDataPresentation } from './AccountDataPage'

const renderWithRouter = (ui: Parameters<typeof renderWithRouterBase>[0]) =>
  renderWithRouterBase(ui, { locale: 'ja' })

const response = (status: number, body: unknown = {}) => ({
  ok: status >= 200 && status < 300,
  status,
  json: mock().mockResolvedValue(body),
})

describe('AccountDataPresentation', () => {
  const baseProps = {
    username: 'taro',
    isAdmin: false,
    downloading: false,
    error: '',
    onExport: mock(),
  }

  it('calls onExport when the download button is clicked', async () => {
    const onExport = mock()
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

describe('AccountDataPage', () => {
  afterEach(() => restoreGlobals())

  it('downloads the export and clears the downloading state', async () => {
    stubGlobal('fetch', mock().mockResolvedValue(response(200, { profile: {} })))
    const createObjectURL = mock().mockReturnValue('blob:mock')
    const revokeObjectURL = mock()
    URL.createObjectURL = createObjectURL
    URL.revokeObjectURL = revokeObjectURL
    const anchorClick = spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})

    await renderWithRouter(<AccountDataPage username="taro" isAdmin={false} />)
    fireEvent.click(screen.getByRole('button', { name: /データをダウンロード/ }))

    await waitFor(() => expect(createObjectURL).toHaveBeenCalledTimes(1))
    expect(anchorClick).toHaveBeenCalledTimes(1)
    expect(revokeObjectURL).toHaveBeenCalledWith('blob:mock')
    expect(screen.getByRole('button', { name: /データをダウンロード/ })).toBeEnabled()
    anchorClick.mockRestore()
  })

  it('shows an error message when the export fails', async () => {
    stubGlobal(
      'fetch',
      mock().mockResolvedValue(response(500, { message: '一時的に利用できません' })),
    )
    await renderWithRouter(<AccountDataPage username="taro" isAdmin={false} />)
    fireEvent.click(screen.getByRole('button', { name: /データをダウンロード/ }))

    expect(await screen.findByText('一時的に利用できません')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /データをダウンロード/ })).toBeEnabled()
  })
})
