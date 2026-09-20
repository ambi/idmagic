---
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-19
priority: p2
depends_on: []
change_kind: tooling
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: "テストの待ち方だけを変える。利用者が観測できる差分が無く、リリースの読者へ知らせるものが無い。"
  references: []
spec_impact: { kind: none, reason: "ブラウザー E2E の安定性の問題であり、製品の観測可能な振る舞いも公開契約も変えない。" }
initial_context:
  source:
    - frontend/tests/e2e/README.md
    - frontend/tests/e2e/fixtures.ts
    - frontend/tests/e2e/webview-deadline.ts
  tests: [frontend/tests/e2e/webview-deadline.test.ts]
  stop_before_reading: [frontend/src, backend, spec]
---

# ブラウザー E2E が実行ごとに結果を変えるのをやめさせる

## Motivation

`mise run test-ui-e2e` は、同じコードに対して成功と失敗を行き来する。
失敗するときは 1 件だけが 60 秒のタイムアウトで落ち、全体が 40 秒から 100 秒へ伸びる。

wi-465 の作業中に 10 回実行し、2 回観測した。

| 実行 | 対象 | 結果 | 所要 |
| --- | --- | --- | --- |
| 1 回目 | 変更あり | 29 pass 1 fail | 99.78 秒 |
| 2〜5 回目 | 変更あり | 30 pass | 39.51〜41.73 秒 |
| 6 回目 | 変更なし | 29 pass 1 fail | 99.75 秒 |
| 7〜10 回目 | 変更なし | 30 pass | 39.15〜43.24 秒 |

**変更の有無に関係なく同じ頻度で起きる。**
したがって wi-465 の変更が持ち込んだものではなく、以前から存在する。

落ちたのは `frontend/tests/e2e/ui-scenario-actions.spec.ts` の
`admin API access token lifecycle works with selected SCIM scopes` である。

2026-09-20 に wi-634 の検証で 3 件目を観測した。
同じ `ui-scenario-actions.spec.ts` の、しかし別のテストである。

| 日付 | 落ちたテスト | 結果 | 所要 |
| --- | --- | --- | --- |
| wi-465 の作業中 | `admin API access token lifecycle works with selected SCIM scopes` | 29 pass 1 fail | 99.78 秒 |
| 2026-09-20 | `password reset succeeds through the local SMTP sink without external mail` | 29 pass 1 fail | 98.23 秒 |
| 2026-09-20（直後の再実行） | なし | 30 pass | 41.34 秒 |

**落ちるテストは 1 本に定まらない。** 特定のテストの論理の問題ではない。

**そして 3 件とも、落ちたテストの所要はそのテスト自身の上限 (60 秒) を丸ごと使い切っている。**
全体の所要も、正常な 41 秒に 60 秒を足した値に一致する。
すなわち残りの 29 本は普通に走り、1 本だけが 60 秒 1 回分止まっている。

ここに観測の手掛かりがある。
`fixtures.ts` の待ちヘルパーはすべて 10〜15 秒の期限を持ち、超えれば `timeout waiting for page kind=...` のように何を待っていたかを述べて投げる。
**報告されたのは bun の `this test timed out after 60000ms` だけで、ヘルパーのメッセージが 1 つも出ていない。**
つまり止まったのはヘルパーの待ちではなく、**期限を持たない 1 回の呼び出し、すなわち `view.navigate` か `view.evaluate` が応答を返さなかった**と読める。

`waitForPage` などのポーリングは `evaluate` の**失敗**を捕まえて再試行するが、**応答が返らない**場合には対処できない。
期限は繰り返しの間でしか見られないので、1 回の呼び出しが返らなければループはそこで止まり、テストの持ち時間を全部使う。
これは「冷えている」説明を要しない。
初回に偏って見えるのは、1 回の取りこぼしが起きる確率の問題である可能性が高く、偏りそのものはまだ確かめていない。

**不安定なテストは、失敗の意味を壊す。** 落ちたときに「直すべき回帰がある」と読めなくなり、再実行して通ればよいという扱いが定着する。そうなると、本物の回帰も同じ扱いで流される。

## Scope

- 止まった呼び出しを名指しする。どのメソッドが、どの引数で、どの URL で、何ミリ秒待って返らなかったかを失敗メッセージへ載せる。
- その観測の下で再現を捕まえ、原因を確定する。
- 確定した原因が upstream のものであれば、そう名乗らせる。次に読む人が同じ調査をやり直さないようにする。
- サーバー側の記録を残す。Go と Vite の出力を捨てている状態をやめる。

## Design

### 採用する形

期限を持たない呼び出しは `navigate` と `evaluate` だけではない。
`loginFromCurrentPage` の `click` と `type` も素で待つので、同じ止まり方をしうる。

| 呼び出し | 素で呼ぶ箇所 | 1 回の期限 |
| --- | --- | --- |
| `evaluate` | `fixtures.ts` に 24、spec に散在 | 2 秒 |
| `navigate` | spec に 24 | 15 秒 |
| `click` | `fixtures.ts` の入力ヘルパー | 10 秒 |
| `type` | 同上 | 10 秒 |

そこで呼び出しごとに期限を書くのではなく、**`openWebView` が返す WebView 自体を期限付きのものにする。**
`openWebView` は既に唯一の入口であり、そこで包めば spec の呼び出しも含めて全部が期限を得る。
新しく書いた spec が包み忘れる余地も無い。

**この段の変更は観測だけを足す。再試行は足さない。**
期限を超えた呼び出しは、メソッド名、引数、そのときの `view.url` を述べて投げる。
テスト自身の上限 60 秒より手前で落ちるので、失敗は「60 秒経った」ではなく「どの呼び出しが返らなかったか」として残る。

期限は待ちを短くするためではないので、正常な呼び出しを落とさない値にする。
`evaluate` の実測は 1 回あたり数ミリ秒であり、`click` と `type` は Bun 側で操作可能になるまで待つ。

| 呼び出し | 1 回の期限 |
| --- | --- |
| `evaluate` | 10 秒 |
| `navigate` | 20 秒 |
| `click`、`type`、`press` | 20 秒 |

期限切れは専用の型で投げる。
`waitForPage` と `setInputValue` は遷移中の `evaluate` の失敗を飲んで再試行するが、**この型は飲まない。**
飲めば、それは原因を調べずに再試行で隠す変更になる。

### 再試行を今は選ばない理由

ポーリングでの再試行は、返らない呼び出しを「無かったこと」にして次の刻みへ進む。
症状は消えるが、**なぜ返らないのかは残る。**
1 回の取りこぼしなら通り、続けて起きれば落ちるという振る舞いは、不安定さの原因を不安定さのまま残す。

再試行を選ぶのは、観測の結果が「Bun 側が応答を 1 回落とした」であり、呼び出し側から避けられないと分かったときだけとする。
そのときは、それが最後の手段であることと、何回諦めたかが失敗に残ることを設計に書く。

### 採用しない案

| 案 | 採用しない理由 |
| --- | --- |
| テストの上限 60 秒を伸ばす | 返らない呼び出しは何秒待っても返らない。失敗までの時間だけが伸びる。 |
| 失敗したテストを自動で再実行する | 不安定さを隠す。動機そのものに反する。 |
| 実行の先頭に暖機の待ちを入れる | 「冷えている」という説明は確かめられていない。確かめずに待ちを足すと、費用だけが残る。 |
| 呼び出しごとに期限付きの関数を作り、呼び出し側を書き換える | 50 箇所以上を書き換えたうえ、素の `view.evaluate` を書けば迂回できる。包み忘れが再び原因を名乗らない失敗を生む。 |
| 先にポーリングで再試行して安定させる | 原因を調べずに症状を消す。返らない理由が残るので、別の待ちで同じことが起きる。観測を入れてから選ぶ。 |

## 診断

### 2026-09-20: 止まっている呼び出しを名指しした

観測を入れて `mise run test-ui-e2e` を連続実行し、11 回通過した後の 12 回目で再現した。

```
WebViewCallExpired: Bun.WebView.evaluate did not answer within 10000 ms:
  document.querySelector('meta[name="idmagic:page"]')?.getAttribute("content") ?? …
  (url=http://localhost:5174/admin)
(fail) admin console scenarios are reachable after admin-audience login [10431.39ms]
Ran 34 tests across 8 files. [50.04s]
```

**止まっていたのは `waitForPage` が投げるページマーカーの `evaluate` であり、応答が返っていない。**

**「間を置いた後の初回だけ」ではない。** 直前に 11 回通過した状態で起きた。
初回に見えていたのは観測数が少なかったためで、暖まっているかどうかとは別である。

### 消えた仮説

| 仮説 | 消えた理由 |
| --- | --- |
| `navigate` が解決しない | 302 リダイレクト、読み込み中の `location.replace`、`history.pushState`、`meta refresh`、3 秒かかるスクリプトのいずれでも解決した (5〜3008 ms)。 |
| `click` が操作可能にならない要素を待つ | 名指しされたのは `evaluate` である。 |
| 遷移中の `evaluate` が失われる | 自前の最小ページへ遷移を重ねながら 5999 回投げて、喪失 0、拒否 0。最小構成では起きない。 |
| Vite の依存再最適化が強制リロードを起こす | `node_modules/.vite/deps` の更新時刻は失敗した実行より前であり、再最適化は起きていない。 |

### 2026-09-20: 原因を確定した

足した観測が答えを出した。

| 観測 | 読み取れること |
| --- | --- |
| 期限切れの直後の追試が `Invalid state: an evaluate() is already pending` | 諦めた呼び出しが「1 view に 1 つ」の枠を握ったままである。**その view は以後使えない。** |
| ページ側の `console` は Vite の HMR 再接続が 4 回だけ。React の警告もエラーも無し | 再レンダリングの暴走ではない。 |
| API のログは全要求 200、遅延 0〜25 ms | サーバー側は健全である。 |
| WebKit のクラッシュレポートは無い | プロセスが落ちた痕跡は残っていない。 |

実アプリのスタックに対してログインを繰り返す計測を書き、**オンデマンドに再現できるようにした。**
待ち方を替えて 60 回ずつ比べた結果が原因を示す。

| 待ち方 | `evaluate` 回数 | wedge |
| --- | --- | --- |
| 25 ms 間隔でポーリング（現状） | 1086 | 7 |
| 200 ms 間隔でポーリング | 364 | 0 |
| 25 ms 間隔、`view.loading` の間は投げない | 1123 | 2 |
| ページ側の `MutationObserver` で 1 回の `evaluate` に集約 | 60 | 30（残り 30 回は別のエラー） |

最後の行が原因を名乗った。
遷移をまたいで pending にする設計にすると、半分が wedge、半分が WebKit のエラーになる。

```
Completion handler for function call is no longer reachable
```

**すなわち、飛んでいる `evaluate` の最中に文書が入れ替わると、完了ハンドラーが呼ばれずに捨てられる。**
半分はこのエラーとして拒否され、半分は何も返らず、そのうえ枠が解放されない。
失敗が「ログイン直後のページマーカー待ち」に集中していたのは、そこが文書の入れ替わりが連続する唯一の場所だからである。
wedge の数が `evaluate` を飛ばした回数に比例することも、同じ説明で説く。

### この欠陥は Bun 側にある

応答を返せなくなった呼び出しは、拒否して枠を解放するべきである。
黙って消えたうえ枠を握り続けるのは、呼び出し側から回復できない。
**したがって最後まで直せるのは upstream だけであり、報告する。**

呼び出し側で潰せるのは「危険な窓へ `evaluate` を投げること」である。
`view.url` と `view.loading` は同期の読み取りで、ページへ問い合わせないので窓を作らない。
遷移が収束してからマーカーを見れば、投げる回数も窓に当たる確率も下がる。
JS 起点の遷移は `loading` が立つまでわずかに遅れるため、窓を完全には閉じられない。
そこは upstream の欠陥として残る。

## Out of Scope

- タイムアウト値を伸ばすこと。伸ばせば失敗は減るが、遅い原因は残り、全体の所要だけが伸びる。
- 失敗したテストの自動再実行。不安定さを隠す手段であり、失敗の意味を壊すという動機そのものに反する。
- サンドボックス内で WebView が描画できない問題。原因も対処も別であり、`frontend/tests/e2e/README.md` の「実行環境の前提」と `openWebView` が既に持つ。

## Plan

段を分けた。
**1 段目は観測だけを入れ、原因を名指しさせる。** 2 段目でその名前に応じて潰す。
1 段目で原因が upstream のものだと確定したので、2 段目は upstream の修正を待つ判断とした（下記「対策を保留する判断」）。

1. 期限付きの包みと期限切れの型を `fixtures.ts` へ入れ、`openWebView` が包んだ WebView を返す。期限切れは飲まずに投げる。
2. 期限切れの扱いを単体テストで固定する。返らない呼び出しを作って、名指しされることと、待ちがそれを飲まないことを確かめる。
3. 連続実行で再現を捕まえ、名指しされた失敗を得る。
4. 原因を確定させる。
5. 既知の欠陥として名乗らせ、サーバー側の記録を残す。

この tooling 変更には製品の規範要件が無い。
Acceptance RED は `mise run test-ui-unit-file -- tests/e2e/webview-deadline.test.ts` で、返らない呼び出しを外側から観測したときにメソッド、引数、期限、URL が欠ける失敗とする。
Unit RED は同じ recipe の `a wait does not swallow an unanswered call` で、待ちヘルパーが専用の期限切れを飲み込む失敗とする。
ブラウザーを通る変更なので、最終確認には `mise run test-ui-e2e` も使う。

### 対策を保留する判断

原因は [oven-sh/bun#43412](https://github.com/oven-sh/bun/issues/43412) であり、呼び出し側では閉じられない。
取れる手はどちらも別の費用を払う。

| 案 | 計測 | 払う費用 |
| --- | --- | --- |
| chrome バックエンドへ移す | 欠陥は webkit 固有で、issue の報告者は chrome では拒否されて view が生き残ることを示している | ローカルが被験ブラウザーを WebKit から Chrome へ替える。Chrome の導入が前提になる |
| ポーリングの刻みを粗くする | `poll-100-idle` で 120 回 0 件、`url-then-poll-25` で 120 回 4 件 | 露出を下げるだけで窓は閉じない。待ちの応答が鈍り、cb75f5f9 が 150 ms から 25 ms へ詰めた分を戻す |

**したがって、この作業は観測と記録までとする。**
失敗は残るが、`WebViewCallExpired` が既知の欠陥として名乗るので、被験コードの回帰とは区別できる。
upstream が直り次第 bun を上げて確かめる。

## Tasks

- [x] T001 [Acceptance] 返らない呼び出しが名指しされることを確かめるテストを RED から始める。待ちが期限切れを飲まないことも固定する。
- [x] T002 [App] 期限付きの包みと期限切れの型を入れ、`openWebView` が包んだ WebView を返すようにする。
- [x] T003 [Diagnose] 連続実行で再現を捕まえ、どの呼び出しがどの画面で返らなかったかを記録する。
- [x] T004 [Diagnose] 原因を確定し、upstream の欠陥として特定する。
- [x] T005 [App] 既知の欠陥を失敗メッセージへ名乗らせ、Go と Vite の出力をファイルへ残す。
- [x] T006 [Verify] `mise run test-ui-e2e` と `mise run verify` を通す。診断中の `test-ui-e2e` では 34 pass / 2 fail を観測し、2 件はいずれも #43412 を名乗った。最終確認は 38 pass、`verify` も通過した。

## Verification

- `mise run test-ui-e2e`（連続実行で失敗率を測る）
- `mise run test-ui-unit`
- `mise run verify`

## Risk Notes

リスクは low。対象はブラウザー E2E の足回りであり、製品の実装には触れない。

**誤った直し方が 1 つだけ危ない。** 待ちの条件を緩める、あるいは失敗した待ちを黙って通す形で「安定」させると、テストは通るようになるが、そのとき失われるのは不安定さではなく検出能力である。直したかどうかは、失敗が消えたことではなく、原因を名指しできたことで判断する。

`reversibility` は reversible。テストの変更であり、公開契約もデータも変えない。

## Completion

- **Completed At**: 2026-09-21
- **Summary**:
  `mise run spec-diff` は、`main` と比較して規範仕様の変更がないと判定した。
  `openWebView` が返す WebView は呼び出し単位の期限を持ち、応答しない呼び出しをメソッド、引数、期限、URL とともに `WebViewCallExpired` として報告する。
  期限切れ後の追試が占有済みの `evaluate` を示す場合は oven-sh/bun#43412 を名指しし、Go、Vite、ページ console の記録場所も失敗へ添える。
  待ちヘルパーは遷移中の通常の `evaluate` 失敗だけを再試行し、`WebViewCallExpired` は隠さず呼び出し元へ返す。
- **Acceptance RED Evidence**:
  - **Test**: `mise run test-ui-e2e` と `a call that never answers names the method, the argument, and the url`。
  - **Requirement**: N/A: 製品要件を変えないブラウザー E2E 診断基盤の tooling 変更である。
  - **Observed Failure**: 実ブラウザーでは 1 件が Bun の 60 秒上限まで止まり、メソッド、引数、URL を示さない `this test timed out after 60000ms` だけを報告した。期限ラッパー導入前の focused check も、返らない `evaluate` をこの診断へ変換できなかった。
  - **Detection Reason**: 返らない呼び出しを利用側から観測し、一般的なテスト上限ではなく、原因調査に必要なメソッド、引数、期限、URL が揃うことを固定する。
- **Unit RED Evidence**:
  - **Test**: `mise run test-ui-unit-file -- tests/e2e/webview-deadline.test.ts` の `a wait does not swallow an unanswered call`。
  - **Requirement**: N/A: 製品要件を変えないブラウザー E2E 診断基盤の tooling 変更である。
  - **Observed Failure**: 期限ラッパー導入前は、返らない `evaluate` が `waitForText` のループを止め、専用の `WebViewCallExpired` を呼び出し元へ返さなかった。
  - **Detection Reason**: 待ちが通常の遷移エラーだけを再試行し、応答しない呼び出しを再試行で隠さないことを、返らない fake WebView で直接区別する。
- **Change-Resistance Results**:
  Low risk の tooling 変更であり、Go の mutation testing は対象外。
  応答しない呼び出し、占有済みの追試、正常に応答する追試を別々に与え、期限切れの診断、既知欠陥の帰属、誤帰属の防止を 6 件の focused test で固定した。
  レビューで、テスト名が主張していた引数の診断を実際には表明していないことを検出し、式そのものを失敗メッセージに要求する assertion を追加した。
- **Verification Results**:
  - `mise run lint-go` - passed (0 issues)
  - `mise run check-work-items` - passed
  - `mise run test-ui-unit-file -- tests/e2e/webview-deadline.test.ts` - passed (6 tests, 13 assertions)
  - `mise run typecheck-ui` - passed
  - `mise run test-ui-e2e` - passed (38 tests)
  - `mise run verify` - passed
  - `mise run spec-diff` - `no normative specification change against main`
