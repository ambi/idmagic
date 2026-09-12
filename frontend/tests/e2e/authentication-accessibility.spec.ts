// docs/standards.md の WCAG22-KEYBOARD と WCAG22-FOCUS を、実ブラウザーの認証画面で観測する。
//
// この 2 行は算出後の状態でしか区別できない。前者は実キーの走査と活性化がフォーカスをどう
// 動かすかであり、後者は Tailwind を算出したあとのスタイルとヒットテストである。Happy DOM の
// 単体テストで書けるのは「クラス名が付いている」までなので、この 2 行は E2E が持つ。
// 残る 2 行 (WCAG22-LABELS-ERRORS / WCAG22-STATUS) は
// frontend/src/features/auth-flow/AuthFlowAccessibility.test.tsx が持つ。
//
// キーの送り方には条件がある。`press("Tab")` と `press("Enter")` は WebKit の編集コマンドへ
// 写像されるので、ボタンの上では何も起きず、Tab ではフォーカスが 1 つも動かない。生の文字
// (`"\t"` と `" "`) は raw keyDown/keyUp へ落ちるため、実際の走査と活性化になる。
import { expect, test } from 'bun:test'
import { authorizePath, demo, uiOrigin, waitForPage, waitForUrl } from './fixtures'

const TAB = '\t'
const ACTIVATE = ' '

type FocusReport = {
  label: string
  focusable: boolean
  obscured: boolean
  indicator: string[]
}

/** いま焦点のある要素を、失敗したときに読める文字列にする。 */
const ACTIVE_LABEL = `(() => {
  const el = document.activeElement
  if (!el) return 'none'
  return [el.tagName, el.getAttribute('type') ?? '', el.getAttribute('name') ?? '',
    (el.getAttribute('aria-label') ?? el.textContent ?? '').trim().slice(0, 24)].join('|')
})()`

async function activeLabel(view: Bun.WebView): Promise<string> {
  return String(await view.evaluate(ACTIVE_LABEL))
}

/**
 * 走査だけで、`match` を満たす要素へフォーカスを移す。`match` は `el` を受ける JavaScript の
 * 式である。上限まで押しても届かないなら、その要素はキーボードから到達できない。
 */
async function tabUntil(view: Bun.WebView, match: string, limit = 24): Promise<void> {
  const test = `(() => { const el = document.activeElement; return el ? Boolean(${match}) : false })()`
  for (let pressed = 0; pressed <= limit; pressed++) {
    if ((await view.evaluate(test)) === true) return
    await view.press(TAB)
    // 走査の反映は次のフレームなので、読む前に 1 拍おく。
    await Bun.sleep(40)
  }
  throw new Error(
    `キーボードの走査が ${limit} 回で ${match} に届かなかった。最後の焦点=${await activeLabel(view)}`,
  )
}

/**
 * いま出ている画面のフォーカス表示を測る。`focus()` の直後は遷移の途中なので、値が落ち着く
 * まで待ってから読む。待たずに読むと、透明なリング (遷移の初期フレーム) を「表示が無い」と
 * 読み違える。
 */
const MEASURE_FOCUS = `(async () => {
  const settle = () => new Promise((resolve) => setTimeout(resolve, 350))
  const paint = (el) => {
    const style = getComputedStyle(el)
    return {
      outlineStyle: style.outlineStyle, outlineWidth: style.outlineWidth,
      boxShadow: style.boxShadow, borderColor: style.borderColor,
      borderWidth: style.borderWidth, backgroundColor: style.backgroundColor,
    }
  }
  const transparent = (color) =>
    color === 'transparent' ||
    /rgba\\([^)]*,\\s*0(?:\\.0+)?\\s*\\)/.test(color) ||
    /\\/\\s*0(?:\\.0+)?\\s*\\)/.test(color)
  const shadows = (value) =>
    value === 'none' ? [] : value.split(/,(?![^(]*\\))/).map((part) => part.trim())
  const drawnShadow = (shadow) => {
    const color = shadow.match(/^[a-z]+\\([^)]*\\)|^#[0-9a-f]+|^[a-z]+/i)?.[0] ?? ''
    const lengths = shadow.replace(color, '').match(/-?\\d*\\.?\\d+px/g) ?? []
    return !transparent(color) && lengths.some((length) => parseFloat(length) !== 0)
  }
  const label = (el) => [el.tagName, el.getAttribute('type') ?? '', el.getAttribute('name') ?? '',
    (el.getAttribute('aria-label') ?? el.textContent ?? '').trim().slice(0, 24)].join('|')

  const elements = [...document.querySelectorAll('a[href], button, input, select, textarea, [tabindex]')]
  document.activeElement?.blur?.()
  await settle()
  const base = elements.map(paint)

  const report = []
  for (let index = 0; index < elements.length; index++) {
    const el = elements[index]
    el.focus()
    await settle()
    const focusable = document.activeElement === el
    const focused = paint(el)
    const before = base[index]
    const indicator = []
    if (focused.outlineStyle !== 'none' && parseFloat(focused.outlineWidth) > 0) indicator.push('outline')
    if (shadows(focused.boxShadow).some(drawnShadow) &&
        focused.boxShadow !== before.boxShadow) indicator.push('shadow')
    if (focused.borderColor !== before.borderColor && !transparent(focused.borderColor) &&
        parseFloat(focused.borderWidth) > 0) indicator.push('border')
    if (focused.backgroundColor !== before.backgroundColor &&
        !transparent(focused.backgroundColor)) indicator.push('background')
    const box = el.getBoundingClientRect()
    const topmost = document.elementFromPoint(box.left + box.width / 2, box.top + box.height / 2)
    el.blur()
    await settle()
    report.push({
      label: label(el), focusable, indicator,
      obscured: !(topmost === el || el.contains(topmost)),
    })
  }
  return JSON.stringify(report)
})()`

async function measureFocus(view: Bun.WebView): Promise<FocusReport[]> {
  return JSON.parse(String(await view.evaluate(MEASURE_FOCUS))) as FocusReport[]
}

/** `tabUntil` へ渡す、表示文字列でボタンを選ぶ式。画面の言語が切り替わっても効くよう両方見る。 */
function buttonWithText(texts: string[]): string {
  return `el.tagName === 'BUTTON' && ${JSON.stringify(texts)}.some((text) => (el.textContent ?? '').includes(text))`
}

//spec:covers WCAG22-KEYBOARD: サインインから同意までの認証操作は、ポインターを 1 度も使わずに
// キーボードだけで完了する。走査で届き、走査で活性化し、判断がクライアントへ返るところまで
// 到達する。
//
// この試験は view.click を 1 度も呼ばない。呼べば「マウスで押せた」ことしか言えず、行を
// 区別できない。
//
// 同意は「許可」ではなく「拒否」で終える。許可すると同意レコードが残り、同じサーバーを
// 共有するあとの spec から同意の段が消えてしまう。拒否は AuthorizationRequest を Rejected に
// するだけでレコードを残さないので、共有の状態に触れずに済む。許可の側は「走査で届く」ところ
// まで観測する。
test('every authentication step completes with the keyboard alone', async () => {
  const view = new Bun.WebView({ width: 1280, height: 2000 })
  try {
    // prompt=consent で毎回同意画面を通す。指定が無いと、先に同意を与えた spec があるかどうかで
    // この試験が測る歩数が変わる。
    await view.navigate(`${uiOrigin}${authorizePath('wcag-keyboard')}&prompt=consent`)
    await waitForPage(view, 'login')

    // 走査の順序が文書の順序から外れていないこと。正の tabindex は順序を書き換えるので、
    // 画面を読む順と操作する順がずれる。
    expect(
      await view.evaluate<number>(`[...document.querySelectorAll('[tabindex]')]
        .map((el) => el.tabIndex).filter((index) => index > 0).length`),
    ).toBe(0)

    await tabUntil(view, `el.getAttribute('name') === 'username'`)
    await view.type(demo.username)
    await tabUntil(view, `el.getAttribute('name') === 'password'`)
    await view.type(demo.password)
    await tabUntil(view, `el.tagName === 'BUTTON' && el.getAttribute('type') === 'submit'`)
    await view.press(ACTIVATE)

    // 同意画面に着いたこと自体が、サインインがキーボードだけで完了した証拠である。
    // サーバーはここへ、認証が成立した要求しか回さない。
    await waitForPage(view, 'consent')
    await tabUntil(view, buttonWithText(['許可', 'Allow']))
    await tabUntil(view, buttonWithText(['拒否', 'Deny']))
    await view.press(ACTIVATE)

    await waitForUrl(view, /localhost:3000\/callback/)
    const callback = new URL(view.url)
    expect(callback.searchParams.get('error')).toBe('access_denied')
    expect(callback.searchParams.get('state')).toBe('wcag-keyboard')
  } finally {
    view.close()
  }
}, 60_000)

//spec:covers WCAG22-FOCUS: 走査で止まる要素はどれも、視認できるフォーカス表示を持ち、他の要素に
// 完全に隠れていない。表示の有無だけを見ると、透明なリングを描く実装も通ってしまうので、
// 非フォーカス時との差と、その差が透明でないことの両方を読む。
test('every focus stop on an authentication screen is visible and unobscured', async () => {
  const view = new Bun.WebView({ width: 1280, height: 2000 })
  try {
    await view.navigate(`${uiOrigin}${authorizePath('wcag-focus')}&prompt=consent`)
    await waitForPage(view, 'login')

    const login = await measureFocus(view)
    expect(login.length).toBeGreaterThan(3)
    expect(login.filter((stop) => stop.focusable && stop.indicator.length === 0)).toEqual([])
    expect(login.filter((stop) => stop.focusable && stop.obscured)).toEqual([])

    // 同意画面はサインインの後ろにあり、要素の種類も違う。1 画面だけでは行を満たしたと
    // 言えないので、続きの画面でも同じことを測る。ここは行が言う対象ではないので、
    // サインイン自体はポインターで進めてよい。
    await view.click('input[name="username"]')
    await view.type(demo.username)
    await view.click('input[name="password"]')
    await view.type(demo.password)
    await view.click('button[type="submit"]')
    await waitForPage(view, 'consent')

    const consent = await measureFocus(view)
    expect(consent.length).toBeGreaterThan(1)
    expect(consent.filter((stop) => stop.focusable && stop.indicator.length === 0)).toEqual([])
    expect(consent.filter((stop) => stop.focusable && stop.obscured)).toEqual([])
  } finally {
    view.close()
  }
}, 60_000)
