// 期限付きの包みは、返らない呼び出しを名指しさせる。ここでは本物の WebView を開かず、
// 返らない evaluate を作って振る舞いを固定する。本物では「返らない」状況を作れないので、
// 固定したいことが確かめられない。
import { expect, test } from 'bun:test'
import { waitForText } from './fixtures'
import { WebViewCallExpired, withCallDeadlines } from './webview-deadline'

// neverAnswers は応答を返さない呼び出し。wi-624 が観測した止まり方をそのまま作る。
const neverAnswers = () => new Promise<never>(() => {})

// fakeView は evaluate の応答を呼ばれた順に返す。足りなくなったら最後の応答を繰り返す。
function fakeView(answers: Array<() => Promise<unknown>>): {
  url: string
  evaluate: () => Promise<unknown>
  close: () => void
  closed: boolean
} {
  let index = 0
  const view = {
    url: 'http://localhost:5174/admin/users',
    evaluate: () => {
      const answer = answers[Math.min(index, answers.length - 1)]
      index += 1
      return answer === undefined ? Promise.resolve(null) : answer()
    },
    close: () => {
      view.closed = true
    },
    closed: false,
  }
  return view
}

test('a call that never answers names the method, the argument, and the url', async () => {
  const view = withCallDeadlines(fakeView([neverAnswers]), { evaluate: 50 })

  const failure = await view.evaluate().catch((error: unknown) => error)

  expect(failure).toBeInstanceOf(WebViewCallExpired)
  expect((failure as Error).message).toContain('Bun.WebView.evaluate')
  expect((failure as Error).message).toContain('50 ms')
  expect((failure as Error).message).toContain('url=http://localhost:5174/admin/users')
})

test('a call that answers within its budget answers unchanged', async () => {
  const view = withCallDeadlines(fakeView([() => Promise.resolve('answered')]), { evaluate: 50 })

  expect(await view.evaluate()).toBe('answered')
})

test('a member without a budget passes through', () => {
  const view = withCallDeadlines(fakeView([]), { evaluate: 50 })

  expect(view.url).toBe('http://localhost:5174/admin/users')
  view.close()
  expect(view.closed).toBe(true)
})

// 枠を握ったままになる形は既知の upstream の欠陥である。次に読む人が同じ調査をやり直さない
// よう、失敗がその名前を述べることを固定する。
test('a wedged view is reported as the known upstream defect', async () => {
  const view = withCallDeadlines(
    fakeView([
      neverAnswers,
      () => Promise.reject(new Error('Invalid state: an evaluate() is already pending')),
    ]),
    { evaluate: 50 },
  )

  const failure = await view.evaluate().catch((error: unknown) => error)

  expect((failure as Error).message).toContain('oven-sh/bun#43412')
  expect((failure as Error).message).toContain('already pending')
})

// 取りこぼした 1 回だけなら、欠陥の名前は添えない。別の原因をこの名前で説明しないためである。
test('a call that expires while the view still answers is not blamed on the defect', async () => {
  const view = withCallDeadlines(fakeView([neverAnswers, () => Promise.resolve(1)]), {
    evaluate: 50,
  })

  const failure = await view.evaluate().catch((error: unknown) => error)

  expect((failure as Error).message).toContain('follow-up evaluate answered')
  expect((failure as Error).message).not.toContain('oven-sh/bun#43412')
})

// 待ちが期限切れを飲むと、原因を調べずに再試行で隠したことになる。飲まないことを固定する。
test('a wait does not swallow an unanswered call', async () => {
  const view = withCallDeadlines(fakeView([neverAnswers]), { evaluate: 50 })

  const failure = await waitForText(view as unknown as Bun.WebView, 'never appears', 5_000).catch(
    (error: unknown) => error,
  )

  expect(failure).toBeInstanceOf(WebViewCallExpired)
})
