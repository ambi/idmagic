import { afterEach, describe, expect, it, mock } from 'bun:test'
import { act, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { restoreGlobals, stubGlobal } from '../test/globals'
import { AdminTheme, useAdminBranding, useUpdateAdminBranding } from './AdminTheme'

function ProductName() {
  return <span>{useAdminBranding().product_name ?? 'IdMagic'}</span>
}

function ApplySavedBranding() {
  const update = useUpdateAdminBranding()
  return (
    <button
      type="button"
      onClick={() => update?.({ product_name: 'Saved', primary_color: '#ff0000' })}
    >
      Apply saved branding
    </button>
  )
}

describe('AdminTheme', () => {
  afterEach(() => {
    restoreGlobals()
  })

  //spec:covers REQ-FRONTEND-005: 保存結果を再読み込みなしで表示し、遅れて完了した保存前の取得結果を捨てる。
  it('keeps saved branding when an older read completes later', async () => {
    let finish!: (value: Response) => void
    stubGlobal(
      'fetch',
      mock(
        () =>
          new Promise<Response>((resolve) => {
            finish = resolve
          }),
      ),
    )
    const { container } = render(
      <AdminTheme>
        <ProductName />
        <ApplySavedBranding />
      </AdminTheme>,
    )
    fireEvent.click(screen.getByRole('button', { name: 'Apply saved branding' }))
    expect(screen.getByText('Saved')).toBeInTheDocument()
    await act(async () => finish(Response.json({ product_name: 'Before saving' })))
    expect(screen.getByText('Saved')).toBeInTheDocument()
    expect((container.firstChild as HTMLElement).style.getPropertyValue('--primary')).toBe(
      '#ff0000',
    )
  })

  //spec:covers REQ-FRONTEND-005: 取得失敗と未設定でも子画面と既定表示を保つ。
  it.each([{}, null, 'failure'])('keeps the console usable for %j', async (response) => {
    const fetch = mock(async () => {
      if (response === 'failure') throw new Error('offline')
      return Response.json(response)
    })
    stubGlobal('fetch', fetch)
    const { container } = render(
      <AdminTheme>
        <ProductName />
        <button type="button">Manage</button>
      </AdminTheme>,
    )
    expect(screen.getByText('IdMagic')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Manage' })).toBeEnabled()
    await waitFor(() => expect(fetch).toHaveBeenCalledTimes(1))
    expect((container.firstChild as HTMLElement).style.getPropertyValue('--primary')).toBe(
      '#020617',
    )
  })

  //spec:covers REQ-FRONTEND-005: realm 切替直後に前の外観を消し、遅れて完了した旧要求も破棄する。
  it('isolates realms and ignores late responses from the previous realm', async () => {
    let finishOld!: (value: Response) => void
    let finishNew!: (value: Response) => void
    const fetch = mock((url: string) => {
      if (url.startsWith('/realms/alpha/'))
        return Promise.resolve(Response.json({ product_name: 'Alpha', primary_color: '#123456' }))
      if (url.startsWith('/realms/old/'))
        return new Promise<Response>((resolve) => {
          finishOld = resolve
        })
      return new Promise<Response>((resolve) => {
        finishNew = resolve
      })
    })
    stubGlobal('fetch', fetch)
    stubGlobal('location', { ...window.location, pathname: '/realms/alpha/admin' })
    const content = (
      <AdminTheme>
        <ProductName />
      </AdminTheme>
    )
    const { container, rerender } = render(content)
    expect(await screen.findByText('Alpha')).toBeInTheDocument()
    stubGlobal('location', { ...window.location, pathname: '/realms/old/admin' })
    rerender(
      <AdminTheme>
        <ProductName />
      </AdminTheme>,
    )
    expect(screen.getByText('IdMagic')).toBeInTheDocument()
    stubGlobal('location', { ...window.location, pathname: '/realms/new/admin' })
    rerender(
      <AdminTheme>
        <ProductName />
      </AdminTheme>,
    )
    await act(async () =>
      finishNew(Response.json({ product_name: 'New', primary_color: '#654321' })),
    )
    expect(screen.getByText('New')).toBeInTheDocument()
    await act(async () =>
      finishOld(Response.json({ product_name: 'Old', primary_color: '#ff0000' })),
    )
    expect(screen.queryByText('Old')).toBeNull()
    expect((container.firstChild as HTMLElement).style.getPropertyValue('--primary')).toBe(
      '#654321',
    )
    expect(document.documentElement.style.getPropertyValue('--primary')).toBe('')
  })
})
