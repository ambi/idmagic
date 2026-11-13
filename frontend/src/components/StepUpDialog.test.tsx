import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, mock } from 'bun:test'
import { restoreGlobals, stubGlobal } from '../test/globals'
import { AuthenticationAPIError, currentBearer, ensureLoggedIn, StepUpRequiredError } from '../api'
import { StepUpCancelledError, useStepUpGuard } from './StepUpDialog'
import { commonDictionary } from '../lib/i18n/common.i18n'

const t = commonDictionary.en
const origin = 'https://idp.test'

type FakePopup = { closed: boolean; location: { href: string } }

function stepUpRequired(acrValues?: string) {
  return new StepUpRequiredError(
    'A more recent authentication is required.',
    'insufficient_user_authentication',
    {
      maxAge: 300,
      ...(acrValues ? { acrValues } : {}),
    },
  )
}

// useStepUpGuard() の dialog は確認の部品を export していないため、実際の呼び出し側と同じ形で
// hook を使う小さな harness を通して検証する。
function StepUpHarness({ action }: { action: () => Promise<string> }) {
  const { guard, dialog } = useStepUpGuard()
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

function result() {
  return document.getElementById('result')?.textContent
}

async function openPrompt(action: () => Promise<string>) {
  render(<StepUpHarness action={action} />)
  fireEvent.click(screen.getByRole('button', { name: 'Run' }))
  await screen.findByText(t.stepUpTitle)
}

describe('useStepUpGuard', () => {
  let popup: FakePopup

  beforeEach(async () => {
    sessionStorage.clear()
    stubGlobal('location', {
      origin,
      href: `${origin}/realms/acme/account/security`,
      pathname: '/realms/acme/account/security',
      search: '',
      assign: mock(),
      replace: mock(),
    })
    popup = { closed: false, location: { href: '' } }
    stubGlobal(
      'open',
      mock(() => popup),
    )
    stubGlobal(
      'fetch',
      mock().mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({ access_token: 'stepped-up', expires_in: 600 }),
      }),
    )
    sessionStorage.setItem(
      'ra_oidc_token_account',
      JSON.stringify({ accessToken: 'before-step-up', expiresAt: Date.now() + 120_000 }),
    )
    await ensureLoggedIn('account', '/account/security')
  })

  afterEach(() => restoreGlobals())

  //spec:covers REQ-AUTHENTICATION-039: 再認証の要求で確認を出し、応答の max_age と acr_values を付けた再認可をポップアップで開き、得たトークンで元の操作を一回だけ再送すること
  it('reauthorizes in a popup and retries the action once', async () => {
    const action = mock()
      .mockRejectedValueOnce(stepUpRequired('urn:idmagic:acr:mfa'))
      .mockResolvedValueOnce('done')
    await openPrompt(action)
    fireEvent.click(screen.getByRole('button', { name: t.stepUpContinue }))

    await waitFor(() => expect(popup.location.href).toContain('/authorize'))
    const authorize = new URL(popup.location.href, origin)
    expect(authorize.searchParams.get('max_age')).toBe('300')
    expect(authorize.searchParams.get('acr_values')).toBe('urn:idmagic:acr:mfa')
    const state = authorize.searchParams.get('state') ?? ''
    expect(state.startsWith('stepup.')).toBe(true)

    // 別のオリジンと別の state のメッセージは再認可の結果として受け取らない。
    window.dispatchEvent(
      new MessageEvent('message', {
        origin: 'https://evil.test',
        data: { type: 'idmagic:step-up-callback', state, code: 'forged' },
      }),
    )
    window.dispatchEvent(
      new MessageEvent('message', {
        origin,
        data: { type: 'idmagic:step-up-callback', state: 'stepup.other', code: 'forged' },
      }),
    )
    window.dispatchEvent(
      new MessageEvent('message', {
        origin,
        data: { type: 'idmagic:step-up-callback', state, code: 'code-1' },
      }),
    )

    await waitFor(() => expect(result()).toBe('resolved:done'))
    expect(action).toHaveBeenCalledTimes(2)
    expect(currentBearer()).toBe('stepped-up')
    expect(fetch).toHaveBeenCalledTimes(1)
    const exchange = (fetch as unknown as ReturnType<typeof mock>).mock.calls[0]
    expect(String(exchange[0])).toContain('/token')
    expect(String((exchange[1] as RequestInit).body)).toContain('code=code-1')
    expect(String((exchange[1] as RequestInit).body)).not.toContain('forged')
  })

  it('does not retry when the popup closes before reauthorization completes', async () => {
    const action = mock().mockRejectedValue(stepUpRequired())
    await openPrompt(action)
    fireEvent.click(screen.getByRole('button', { name: t.stepUpContinue }))
    await waitFor(() => expect(popup.location.href).toContain('/authorize'))
    popup.closed = true

    await screen.findByText(t.stepUpNotCompleted, {}, { timeout: 2000 })
    expect(action).toHaveBeenCalledTimes(1)
    fireEvent.click(screen.getByRole('button', { name: t.cancel }))
    await waitFor(() => expect(result()).toBe('cancelled'))
  })

  it('reauthorizes the whole page when the popup is blocked', async () => {
    stubGlobal(
      'open',
      mock(() => null),
    )
    await openPrompt(mock().mockRejectedValue(stepUpRequired()))
    fireEvent.click(screen.getByRole('button', { name: t.stepUpContinue }))

    await waitFor(() => expect(window.location.replace).toHaveBeenCalled())
    const target = new URL(
      String((window.location.replace as ReturnType<typeof mock>).mock.calls[0][0]),
      origin,
    )
    expect(target.pathname).toBe('/realms/acme/authorize')
    expect(target.searchParams.get('max_age')).toBe('300')
  })

  it('passes other failures through without prompting', async () => {
    render(
      <StepUpHarness
        action={mock().mockRejectedValue(new AuthenticationAPIError('nope', 'access_denied'))}
      />,
    )
    fireEvent.click(screen.getByRole('button', { name: 'Run' }))
    await waitFor(() => expect(result()).toBe('rejected'))
    expect(screen.queryByText(t.stepUpTitle)).toBeNull()
  })
})
