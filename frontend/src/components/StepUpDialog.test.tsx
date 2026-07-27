import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, mock } from 'bun:test'
import { restoreGlobals, stubGlobal } from '../test/globals'
import { AuthenticationAPIError } from '../api'
import { StepUpCancelledError, useStepUpGuard } from './StepUpDialog'
import { commonDictionary } from '../lib/i18n/common.i18n'

const t = commonDictionary.en

const response = (status: number, body: unknown = {}) => ({
  ok: status >= 200 && status < 300,
  status,
  json: mock().mockResolvedValue(body),
})

function stubStepUpFetch(methods: string[], completeStatus?: number, completeBody?: unknown) {
  stubGlobal(
    'fetch',
    mock((url: string) => {
      if (url.includes('/step_up/start')) return Promise.resolve(response(200, { methods }))
      if (url.includes('/step_up/complete') && completeStatus !== undefined) {
        return Promise.resolve(response(completeStatus, completeBody))
      }
      throw new Error(`unexpected fetch ${url}`)
    }),
  )
}

function stepUpRequiredError() {
  return new AuthenticationAPIError('Reauthentication is required.', 'step_up_required')
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
        Run
      </button>
      <p id="result" />
      {dialog}
    </div>
  )
}

describe('useStepUpGuard', () => {
  afterEach(() => restoreGlobals())

  it('retries the guarded action once re-authentication succeeds', async () => {
    stubStepUpFetch(['password'], 204)
    const action = mock().mockRejectedValueOnce(stepUpRequiredError()).mockResolvedValueOnce('done')

    render(<StepUpHarness action={action} />)
    fireEvent.click(screen.getByRole('button', { name: 'Run' }))

    expect(await screen.findByRole('dialog')).toBeInTheDocument()
    fireEvent.change(screen.getByLabelText(t.currentPassword), { target: { value: 'secret' } })
    fireEvent.click(screen.getByRole('button', { name: t.reauthenticateAndContinue }))

    await waitFor(() => expect(screen.getByText('resolved:done')).toBeInTheDocument())
    expect(action).toHaveBeenCalledTimes(2)
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('shows an error and keeps the dialog open when re-authentication fails', async () => {
    stubStepUpFetch(['password'], 400, { message: 'The password is incorrect.' })
    const action = mock().mockRejectedValueOnce(stepUpRequiredError())

    render(<StepUpHarness action={action} />)
    fireEvent.click(screen.getByRole('button', { name: 'Run' }))

    expect(await screen.findByRole('dialog')).toBeInTheDocument()
    fireEvent.change(screen.getByLabelText(t.currentPassword), { target: { value: 'wrong' } })
    fireEvent.click(screen.getByRole('button', { name: t.reauthenticateAndContinue }))

    expect(await screen.findByText('The password is incorrect.')).toBeInTheDocument()
    expect(screen.getByRole('dialog')).toBeInTheDocument()
    expect(action).toHaveBeenCalledTimes(1)
  })

  it('rejects with StepUpCancelledError when the user cancels', async () => {
    stubStepUpFetch(['password'])
    const action = mock().mockRejectedValueOnce(stepUpRequiredError())

    render(<StepUpHarness action={action} />)
    fireEvent.click(screen.getByRole('button', { name: 'Run' }))

    expect(await screen.findByRole('dialog')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: t.cancel }))

    await waitFor(() => expect(screen.getByText('cancelled')).toBeInTheDocument())
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    expect(action).toHaveBeenCalledTimes(1)
  })

  it('cancels when the backdrop is clicked', async () => {
    stubStepUpFetch(['password'])
    const action = mock().mockRejectedValueOnce(stepUpRequiredError())

    render(<StepUpHarness action={action} />)
    fireEvent.click(screen.getByRole('button', { name: 'Run' }))

    const dialog = await screen.findByRole('dialog')
    fireEvent.click(dialog)

    await waitFor(() => expect(screen.getByText('cancelled')).toBeInTheDocument())
  })

  it('cancels on Escape', async () => {
    stubStepUpFetch(['password'])
    const action = mock().mockRejectedValueOnce(stepUpRequiredError())

    render(<StepUpHarness action={action} />)
    fireEvent.click(screen.getByRole('button', { name: 'Run' }))

    const dialog = await screen.findByRole('dialog')
    fireEvent.keyDown(dialog, { key: 'Escape' })

    await waitFor(() => expect(screen.getByText('cancelled')).toBeInTheDocument())
  })

  it('switches methods and clears the credential field', async () => {
    stubStepUpFetch(['password', 'totp'])
    const action = mock().mockRejectedValueOnce(stepUpRequiredError())

    render(<StepUpHarness action={action} />)
    fireEvent.click(screen.getByRole('button', { name: 'Run' }))

    await screen.findByRole('dialog')
    fireEvent.change(screen.getByLabelText(t.currentPassword), { target: { value: 'secret' } })
    fireEvent.click(screen.getByRole('button', { name: t.authenticatorApp }))

    expect(screen.getByLabelText(t.totpCode)).toHaveValue('')
  })

  it('renders the passkey prompt instead of a credential field for webauthn', async () => {
    stubStepUpFetch(['webauthn'])
    const action = mock().mockRejectedValueOnce(stepUpRequiredError())

    render(<StepUpHarness action={action} />)
    fireEvent.click(screen.getByRole('button', { name: 'Run' }))

    await screen.findByRole('dialog')
    expect(screen.getByText(t.passkeyStepUpDescription)).toBeInTheDocument()
    expect(screen.queryByLabelText(t.currentPassword)).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: t.authenticateWithPasskey })).toBeEnabled()
  })
})
