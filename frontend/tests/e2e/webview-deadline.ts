// Bun.WebView の呼び出しは期限を持たない。応答が返らなければ、待ちのループは繰り返しの間で
// しか期限を見られないので、そこで止まってテストの持ち時間を丸ごと使う (wi-624)。
// そのとき残るのは bun の「60 秒経った」だけで、どの呼び出しが返らなかったかは分からない。
//
// ここが足すのは観測である。**再試行はしない。** 期限を超えた呼び出しは、メソッド名、引数、
// そのときの URL を述べて落ちる。原因を名指しさせることが目的であり、症状を隠すことではない。
//
// 期限は待ちを短くするためのものではないので、正常な呼び出しを落とさない値を渡す。

// WebViewCallExpired は「応答が返らなかった」ことだけを表す。呼び出しが返した失敗とは別の
// 型にする。遷移中の失敗を飲んで再試行する待ちが、返らなかったことまで飲まないようにする。
export class WebViewCallExpired extends Error {
  constructor(method: string, budgetMs: number, detail: string, url: string, notes: string[]) {
    const target = detail === '' ? '' : `: ${detail}`
    const where = url === '' ? '' : ` (url=${url})`
    const tail = notes.length === 0 ? '' : `\n${notes.map((note) => `  ${note}`).join('\n')}`
    super(`Bun.WebView.${method} did not answer within ${budgetMs} ms${target}${where}${tail}`)
    this.name = 'WebViewCallExpired'
  }
}

// KNOWN_WEBKIT_DEFECT は、この失敗が既知の upstream の欠陥であることを次に読む人へ渡す。
// webkit バックエンドでは、飛んでいる evaluate の最中に文書が入れ替わると完了ハンドラーが
// 呼ばれずに捨てられ、「1 view に 1 つ」の枠も解放されない。ページ自身が遷移を始めるので
// 呼び出し側では避けられない。追試が枠の占有を報告したときだけ添える。
const KNOWN_WEBKIT_DEFECT =
  'this is the signature of oven-sh/bun#43412: on the webkit backend an evaluate() overtaken by a navigation never settles and wedges the view. The chrome backend rejects it instead.'

const PENDING_GUARD = 'already pending'

const EXPIRED = Symbol('expired')

// 失敗メッセージに載せる引数は 1 行へ潰して切る。evaluate の式は数十行になりうる。
const DETAIL_LIMIT = 80

// 期限切れの直後に投げる小さな評価の猶予。生きている view なら数ミリ秒で返る。
const LIVENESS_BUDGET_MS = 1_000

function detailOf(args: readonly unknown[]): string {
  const first = args[0]
  if (typeof first !== 'string') return ''
  const oneLine = first.replace(/\s+/g, ' ').trim()
  return oneLine.length <= DETAIL_LIMIT ? oneLine : `${oneLine.slice(0, DETAIL_LIMIT)}…`
}

// urlOf は止まった時点の所在を読む。応答を待たない同期の読み取りなので、返らない呼び出しの
// 後でも得られる。読めない包み対象もありうるので、読めなければ黙って省く。
function urlOf(view: object): string {
  try {
    const url = Reflect.get(view, 'url')
    return typeof url === 'string' ? url : ''
  } catch {
    return ''
  }
}

function failureText(error: unknown): string {
  return error instanceof Error ? error.message : String(error)
}

// probeLiveness は期限切れの直後に、同じ view へ最小の評価を 1 つ投げる。**取りこぼした 1 回
// だけの問題か、その view がもう応答しないのかを、ここで分ける。** 分けないと、対策として
// 再試行を選ぶべきかどうかが決まらない。
//
// evaluate は 1 view に 1 回しか同時に飛ばせないので、諦めた呼び出しが枠を握ったままなら
// この評価は ERR_INVALID_STATE で失敗する。それも答えの 1 つとして本文へ載せる。
async function probeLiveness(view: object): Promise<string> {
  const evaluate = Reflect.get(view, 'evaluate')
  if (typeof evaluate !== 'function') return ''
  const started = Date.now()
  try {
    const answer = await Promise.race([
      Promise.resolve(evaluate.call(view, '1')).then(() => 'answered'),
      Bun.sleep(LIVENESS_BUDGET_MS).then(() => 'did not answer either'),
    ])
    return `follow-up evaluate ${answer} after ${Date.now() - started} ms`
  } catch (error) {
    return `follow-up evaluate failed: ${failureText(error)}`
  }
}

async function callWithin<T>(
  view: object,
  method: string,
  budgetMs: number,
  start: () => unknown,
  detail: string,
  context: CallContext | undefined,
): Promise<T> {
  const call = Promise.resolve(start()) as Promise<T>
  let timer: ReturnType<typeof setTimeout> | undefined
  const expiry = new Promise<typeof EXPIRED>((resolve) => {
    timer = setTimeout(() => resolve(EXPIRED), budgetMs)
  })
  try {
    const outcome = await Promise.race([call.then((value) => ({ value })), expiry])
    if (outcome !== EXPIRED) return outcome.value
    // 諦めた呼び出しは捨てない。後から失敗しても未処理の拒否にしない。
    call.catch(() => {})
    const liveness = await probeLiveness(view)
    const notes = [
      liveness,
      liveness.includes(PENDING_GUARD) ? KNOWN_WEBKIT_DEFECT : '',
      context?.() ?? '',
    ].filter((note) => note !== '')
    throw new WebViewCallExpired(method, budgetMs, detail, urlOf(view), notes)
  } finally {
    clearTimeout(timer)
  }
}

// CallContext は期限切れのときだけ呼ばれ、失敗の本文へ足す手掛かりを返す。呼び出し側が
// 持っている記録 (ページ側の console、サーバーのログの場所) をここから載せる。
export type CallContext = () => string

// withCallDeadlines は budgets が名前を挙げたメソッドだけを期限付きにする。挙げていない
// メンバー (url、close など) はそのまま通す。
//
// 包むのは WebView 自体であって呼び出し側ではない。呼び出し側に書かせると、素の
// view.evaluate を書いた 1 箇所が再び名前のないタイムアウトへ戻す。
export function withCallDeadlines<T extends object>(
  view: T,
  budgets: Readonly<Record<string, number>>,
  context?: CallContext,
): T {
  return new Proxy(view, {
    get(target, property) {
      const value = Reflect.get(target, property, target)
      if (typeof value !== 'function') return value
      const method = String(property)
      const budget = budgets[method]
      if (budget === undefined) return value.bind(target)
      return (...args: unknown[]) =>
        callWithin(target, method, budget, () => value.apply(target, args), detailOf(args), context)
    },
  })
}
