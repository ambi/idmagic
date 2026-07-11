import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { AuthenticationAPIError } from '../api'
import { StepUpCancelledError, useStepUpGuard } from './StepUpDialog'

const response = (status: number, body: unknown = {}) => ({
  ok: status >= 200 && status < 300,
  status,
  json: vi.fn().mockResolvedValue(body),
})

function stubStepUpFetch(methods: string[], completeStatus?: number, completeBody?: unknown) {
  vi.stubGlobal(
    'fetch',
    vi.fn((url: string) => {
      if (url.includes('/step_up/start')) return Promise.resolve(response(200, { methods }))
      if (url.includes('/step_up/complete') && completeStatus !== undefined) {
        return Promise.resolve(response(completeStatus, completeBody))
      }
      throw new Error(`unexpected fetch ${url}`)
    }),
  )
}

function stepUpRequiredError() {
  return new AuthenticationAPIError('再認証が必要です', 'step_up_required')
}

// useStepUpGuard() の dialog は StepUpDialog 自体を export していないため、実際の呼び出し側
// (AccountEmailsPage 等) と同じ形で hook を使う小さな harness を通して検証する。
function StepUpHarness({ action }: { action: () => Promise<string> }) {
  const { guard, dialog } = useStepUpGuard('csrf-token')
  return (
    <div>
      <button
        type="button"
        onClick={() => {
          guard(action)
            .then((value) => {
              document.getElementById('result')!.textContent = `resolved:${value}`
            })
            .catch((cause) => {
              document.getElementById('result')!.textContent =
                cause instanceof StepUpCancelledError ? 'cancelled' : 'rejected'
            })
        }}
      >
        実行
      </button>
      <p id="result" />
      {dialog}
    </div>
  )
}

describe('useStepUpGuard', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('retries the guarded action once re-authentication succeeds', async () => {
    stubStepUpFetch(['password'], 204)
    const action = vi
      .fn()
      .mockRejectedValueOnce(stepUpRequiredError())
      .mockResolvedValueOnce('done')

    render(<StepUpHarness action={action} />)
    fireEvent.click(screen.getByRole('button', { name: '実行' }))

    expect(await screen.findByRole('dialog')).toBeInTheDocument()
    fireEvent.change(screen.getByLabelText('現在のパスワード'), { target: { value: 'secret' } })
    fireEvent.click(screen.getByRole('button', { name: '再認証して続行' }))

    await waitFor(() => expect(screen.getByText('resolved:done')).toBeInTheDocument())
    expect(action).toHaveBeenCalledTimes(2)
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('shows an error and keeps the dialog open when re-authentication fails', async () => {
    stubStepUpFetch(['password'], 400, { message: 'パスワードが違います' })
    const action = vi.fn().mockRejectedValueOnce(stepUpRequiredError())

    render(<StepUpHarness action={action} />)
    fireEvent.click(screen.getByRole('button', { name: '実行' }))

    expect(await screen.findByRole('dialog')).toBeInTheDocument()
    fireEvent.change(screen.getByLabelText('現在のパスワード'), { target: { value: 'wrong' } })
    fireEvent.click(screen.getByRole('button', { name: '再認証して続行' }))

    expect(await screen.findByText('パスワードが違います')).toBeInTheDocument()
    expect(screen.getByRole('dialog')).toBeInTheDocument()
    expect(action).toHaveBeenCalledTimes(1)
  })

  it('rejects with StepUpCancelledError when the user cancels', async () => {
    stubStepUpFetch(['password'])
    const action = vi.fn().mockRejectedValueOnce(stepUpRequiredError())

    render(<StepUpHarness action={action} />)
    fireEvent.click(screen.getByRole('button', { name: '実行' }))

    expect(await screen.findByRole('dialog')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'キャンセル' }))

    await waitFor(() => expect(screen.getByText('cancelled')).toBeInTheDocument())
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    expect(action).toHaveBeenCalledTimes(1)
  })

  it('cancels when the backdrop is clicked', async () => {
    stubStepUpFetch(['password'])
    const action = vi.fn().mockRejectedValueOnce(stepUpRequiredError())

    render(<StepUpHarness action={action} />)
    fireEvent.click(screen.getByRole('button', { name: '実行' }))

    const dialog = await screen.findByRole('dialog')
    fireEvent.click(dialog)

    await waitFor(() => expect(screen.getByText('cancelled')).toBeInTheDocument())
  })

  it('cancels on Escape', async () => {
    stubStepUpFetch(['password'])
    const action = vi.fn().mockRejectedValueOnce(stepUpRequiredError())

    render(<StepUpHarness action={action} />)
    fireEvent.click(screen.getByRole('button', { name: '実行' }))

    const dialog = await screen.findByRole('dialog')
    fireEvent.keyDown(dialog, { key: 'Escape' })

    await waitFor(() => expect(screen.getByText('cancelled')).toBeInTheDocument())
  })

  it('switches methods and clears the credential field', async () => {
    stubStepUpFetch(['password', 'totp'])
    const action = vi.fn().mockRejectedValueOnce(stepUpRequiredError())

    render(<StepUpHarness action={action} />)
    fireEvent.click(screen.getByRole('button', { name: '実行' }))

    await screen.findByRole('dialog')
    fireEvent.change(screen.getByLabelText('現在のパスワード'), { target: { value: 'secret' } })
    fireEvent.click(screen.getByRole('button', { name: '認証アプリ' }))

    expect(screen.getByLabelText('認証アプリの 6 桁コード')).toHaveValue('')
  })

  it('renders the passkey prompt instead of a credential field for webauthn', async () => {
    stubStepUpFetch(['webauthn'])
    const action = vi.fn().mockRejectedValueOnce(stepUpRequiredError())

    render(<StepUpHarness action={action} />)
    fireEvent.click(screen.getByRole('button', { name: '実行' }))

    await screen.findByRole('dialog')
    expect(
      screen.getByText('登録済みのパスキー (指紋・顔認証・セキュリティキー) で本人確認します。'),
    ).toBeInTheDocument()
    expect(screen.queryByLabelText('現在のパスワード')).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'パスキーで認証' })).toBeEnabled()
  })
})
