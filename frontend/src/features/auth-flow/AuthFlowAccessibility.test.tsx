import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, mock } from 'bun:test'
import { restoreGlobals, stubGlobal } from '../../test/globals'
import { commonDictionary } from '../../lib/i18n/common.i18n'
import { ConsentPage } from './ConsentPage'
import { DevicePage } from './DevicePage'
import { EmailVerifyPage } from './EmailVerifyPage'
import { ForgotPasswordPage } from './ForgotPasswordPage'
import { LoginPage } from './LoginPage'
import { MfaEnrollmentPage } from './MfaEnrollmentPage'
import { ResetPasswordPage } from './ResetPasswordPage'
import { TotpPage } from './TotpPage'
import { loginPageDictionary } from './LoginPage.i18n'
import { passwordRecoveryDictionary } from './PasswordRecoveryPages.i18n'

// docs/standards.md の WCAG22-LABELS-ERRORS と WCAG22-STATUS を、認証画面を入口として
// 観測する。行は「すべての認証操作」を対象にしているので、入口は個々の部品ではなく画面で
// あり、8 つある認証画面すべてを通す。
//
// 残る 2 行 (WCAG22-KEYBOARD / WCAG22-FOCUS) は
// frontend/tests/e2e/authentication-accessibility.spec.ts が持つ。実キーの走査と算出後の
// スタイルは実ブラウザーにしか無く、ここで書けるのは「クラス名が付いている」までである。

const loginT = loginPageDictionary.en
const recoveryT = passwordRecoveryDictionary.en
const commonT = commonDictionary.en

const response = (status: number, body: unknown = {}) => ({
  ok: status >= 200 && status < 300,
  status,
  json: mock().mockResolvedValue(body),
})

// どの画面も AuthShell がブランディングを、画面によっては自分の初期取得を行う。1 つの本文で
// すべてを賄えるので、既定の取得はまとめて成功させる。
const readyBody = {
  providers: [],
  secret: 'JBSWY3DPEHPK3PXP',
  account_name: 'alice@example.com',
  issuer: 'IdMagic',
}

/**
 * 支援技術が読む名前。label 要素、aria-label、aria-labelledby、title の順に見る。
 * 空文字を返したら、その入力には名前が無い。
 */
function accessibleName(element: Element): string {
  const ariaLabel = element.getAttribute('aria-label')
  if (ariaLabel?.trim()) return ariaLabel.trim()

  const labelledBy = element.getAttribute('aria-labelledby')
  if (labelledBy) {
    const named = labelledBy
      .split(/\s+/)
      .map((id) => document.getElementById(id)?.textContent ?? '')
      .join(' ')
      .trim()
    if (named) return named
  }

  if (element.id) {
    const label = document.querySelector(`label[for="${element.id}"]`)
    if (label?.textContent?.trim()) return label.textContent.trim()
  }

  const wrapping = element.closest('label')
  if (wrapping?.textContent?.trim()) return wrapping.textContent.trim()

  return element.getAttribute('title')?.trim() ?? ''
}

/**
 * 支援技術へ通知される入れ物。`role="alert"` と `role="status"` は暗黙のライブリージョンで、
 * `aria-live` は明示のそれである。どれでもなければ、内容が変わっても読み上げは起きない。
 */
function enclosingLiveRegion(element: Element): Element | null {
  return element.closest('[aria-live], [role="alert"], [role="status"]')
}

/** 画面に出ている入力の識別子。id か name のどちらかで呼ぶ。 */
function renderedInputs(): string[] {
  return [...document.querySelectorAll('input')]
    .filter((input) => input.type !== 'hidden')
    .map((input) => input.id || input.name || `(${input.type})`)
    .sort()
}

function unnamedInputs(): string[] {
  return [...document.querySelectorAll('input')]
    .filter((input) => input.type !== 'hidden')
    .filter((input) => accessibleName(input) === '')
    .map((input) => input.id || input.name || `(${input.type})`)
    .sort()
}

/**
 * 名前付きの認証画面と、そこに出るはずの入力。`inputs` を書いておかないと、画面が何も
 * 描画できていないときにも「名前の無い入力は 0 件」で通ってしまう。
 */
const authenticationScreens: { name: string; inputs: string[]; render: () => void }[] = [
  {
    name: 'login',
    inputs: ['password', 'username'],
    render: () => render(<LoginPage csrfToken="csrf" returnTo="/return" />),
  },
  {
    name: 'totp',
    inputs: ['code', 'remember_device'],
    render: () =>
      render(
        <TotpPage
          csrfToken="csrf"
          secondFactorMethods={['totp', 'webauthn', 'recovery_code']}
          canRememberDevice
        />,
      ),
  },
  {
    name: 'mfa-enrollment',
    inputs: ['enrollment-code'],
    render: () => render(<MfaEnrollmentPage csrfToken="csrf" />),
  },
  {
    name: 'device',
    inputs: ['user-code'],
    render: () => render(<DevicePage csrfToken="csrf" userCode="ABCD-EFGH" />),
  },
  {
    name: 'forgot-password',
    inputs: ['email'],
    render: () => render(<ForgotPasswordPage csrfToken="csrf" />),
  },
  {
    name: 'reset-password',
    inputs: ['new_password'],
    render: () => render(<ResetPasswordPage csrfToken="csrf" token="reset-token" />),
  },
  {
    name: 'email-verify',
    inputs: [],
    render: () => render(<EmailVerifyPage csrfToken="csrf" token="tok" />),
  },
  {
    name: 'consent',
    inputs: [],
    render: () => render(<ConsentPage csrfToken="csrf" clientName="Portal" scopes={['openid']} />),
  },
]

describe('authentication screen accessibility', () => {
  const originalLocation = window.location

  beforeEach(() => {
    stubGlobal('location', { ...originalLocation, assign: mock() })
    stubGlobal('fetch', mock().mockResolvedValue(response(200, readyBody)))
  })

  afterEach(() => restoreGlobals())

  //spec:covers WCAG22-LABELS-ERRORS: どの認証画面でも、すべての入力が支援技術から読める名前を持つ。
  it('gives every input on every authentication screen an accessible name', async () => {
    for (const authenticationScreen of authenticationScreens) {
      document.body.innerHTML = ''
      authenticationScreen.render()
      // 取得のあとに現れる入力があるので、期待する入力が揃うまで待つ。ここが揃わないまま
      // 名前を数えると、描画できていない画面が「名前の無い入力は 0 件」で通ってしまう。
      await waitFor(() =>
        expect({ screen: authenticationScreen.name, inputs: renderedInputs() }).toEqual({
          screen: authenticationScreen.name,
          inputs: [...authenticationScreen.inputs].sort(),
        }),
      )
      expect({ screen: authenticationScreen.name, unnamed: unnamedInputs() }).toEqual({
        screen: authenticationScreen.name,
        unnamed: [],
      })
    }
  })

  //spec:covers WCAG22-LABELS-ERRORS: 送信の失敗はテキストで識別され、直せる失敗はその直し方を述べる。
  // 表示するのは backend の生の本文ではなく、対応付けた文言である。生の本文をそのまま出す
  // 実装は、識別はできても修正方法を示せない。
  it('identifies a failed submission in text and states how to correct it', async () => {
    stubGlobal(
      'fetch',
      mock()
        .mockResolvedValueOnce(response(200, readyBody))
        .mockResolvedValueOnce(response(200, readyBody))
        .mockResolvedValueOnce(
          response(403, { error: 'csrf_token', message: 'csrf token mismatch' }),
        ),
    )
    render(<LoginPage csrfToken="csrf" returnTo="/return" />)
    fireEvent.change(screen.getByLabelText(loginT.usernameLabel), { target: { value: 'alice' } })
    fireEvent.change(screen.getByLabelText(loginT.passwordLabel), { target: { value: 'secret' } })
    fireEvent.click(screen.getByRole('button', { name: loginT.submit }))

    const alert = await screen.findByRole('alert')
    // 「ページを再読み込みして、もう一度お試しください」— 次にとる行動を述べている。
    expect(alert).toHaveTextContent(commonT.csrfToken)
    expect(alert).not.toHaveTextContent('csrf token mismatch')

    // 入力の内容そのものが不正な失敗でも、直し方が本文に出る。
    document.body.innerHTML = ''
    stubGlobal('fetch', mock().mockResolvedValue(response(400, { error: 'password_policy' })))
    render(<ResetPasswordPage csrfToken="csrf" token="reset-token" />)
    fireEvent.change(screen.getByLabelText(recoveryT.newPassword), {
      target: { value: 'a long new password' },
    })
    fireEvent.click(screen.getByRole('button', { name: recoveryT.updatePassword }))
    expect(await screen.findByRole('alert')).toHaveTextContent(recoveryT.passwordPolicy)
  })

  //spec:covers WCAG22-STATUS: 結果と送信エラーの入れ物はライブリージョンであり、状態が変わっても
  // フォーカスは動かない。フォーカスを結果へ移す実装は、読み上げは起きても操作の文脈を
  // 奪うので、この行が禁じている側である。
  it('announces a submission failure through a live region without moving focus', async () => {
    stubGlobal(
      'fetch',
      mock()
        .mockResolvedValueOnce(response(200, readyBody))
        .mockResolvedValueOnce(response(200, readyBody))
        .mockResolvedValueOnce(response(401, { error: 'invalid_credentials' })),
    )
    render(<LoginPage csrfToken="csrf" returnTo="/return" />)
    const password = screen.getByLabelText(loginT.passwordLabel)
    fireEvent.change(screen.getByLabelText(loginT.usernameLabel), { target: { value: 'alice' } })
    fireEvent.change(password, { target: { value: 'wrong' } })
    ;(password as HTMLInputElement).focus()
    expect(document.activeElement).toBe(password)

    fireEvent.click(screen.getByRole('button', { name: loginT.submit }))

    const alert = await screen.findByRole('alert')
    expect(alert).toHaveTextContent(commonT.invalidCredentials)
    // 送信の結果はフォーカスを奪わずに知らされる。
    expect(document.activeElement).toBe(password)
  })

  //spec:covers WCAG22-STATUS: 成功の通知も同じ扱いを受ける。失敗だけをライブリージョンに入れる実装は、
  // 「認証結果や送信エラーを」と 2 つ挙げているこの行の片方しか満たさない。
  it('announces a successful submission through a live region without moving focus', async () => {
    stubGlobal('fetch', mock().mockResolvedValue(response(204)))
    render(<ForgotPasswordPage csrfToken="csrf" />)
    const email = screen.getByLabelText(recoveryT.emailAddress)
    fireEvent.change(email, { target: { value: 'alice@example.com' } })
    ;(email as HTMLInputElement).focus()

    fireEvent.click(screen.getByRole('button', { name: recoveryT.sendResetLink }))

    const sent = await screen.findByText(recoveryT.resetSent)
    expect(enclosingLiveRegion(sent)).not.toBeNull()
    expect(document.activeElement).toBe(email)
  })
})
