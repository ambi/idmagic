---
status: in_progress
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-20
priority: p1
depends_on: []
change_kind: tooling
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: "テストの実行基盤だけを変える。利用者が観測できる差分が無く、リリースの読者へ知らせるものが無い。"
  references: []
spec_impact:
  kind: none
  reason: "ブラウザー E2E の起動方法と失敗時の報告だけを変え、製品の観測可能な振る舞いも TypeSpec の契約も変えない。"
initial_context:
  source:
    - frontend/tests/e2e/fixtures.ts
    - frontend/tests/e2e/webview-deadline.ts
    - frontend/tests/e2e/webview-process-lease.ts
    - frontend/tests/e2e/README.md
    - .github/workflows/idmagic-ci.yaml
  tests:
    - frontend/tests/e2e/webview-deadline.test.ts
    - frontend/tests/e2e/webview-process-lease.test.ts
    - frontend/tests/e2e/ui-scenario-actions.spec.ts
  stop_before_reading: [backend, spec]
---

# ブラウザー E2E を GitHub Actions の Linux runner で実行できるようにする

## 動機

CI の `ui-e2e` ジョブは、2026-09-06 に追加されてから一度も成功していない。

| 期間 | 観測される失敗 |
| --- | --- |
| 2026-09-06 から 2026-09-19 まで | spec ごとに `timeout waiting for page kind=login` |
| 2026-09-19 の dbbd2631 以降 | 28 件が `Bun.WebView could not render a blank page.` |

**メッセージは原因を名乗っていない。**
現在の文面は macOS の Seatbelt だけを挙げ、「サンドボックス外で `mise run test-ui-e2e` を実行せよ」と案内する。
Linux runner ではどちらも当たらないため、2 週間にわたって読んでも直せない失敗が残り続けた。

原因は `Bun.WebView` のバックエンドがプラットフォームで異なることにある。

| プラットフォーム | バックエンド | 画面サーバー |
| --- | --- | --- |
| macOS | WKWebView | 必要 |
| Linux、Windows | インストール済みの Chrome を CDP で駆動 | 不要。Bun が `--headless` を渡す |

`ubuntu-latest` (24.04) には Google Chrome が同梱されるので、Bun の探索は当たる。
当たらない場合の Bun のエラーは `Failed to spawn Chrome (set BUN_CHROME_PATH, backend.path, or install Chrome/Chromium)` であり、CI の症状とは別物である。

落ちているのは Chrome の起動である。
Ubuntu 23.10 以降は `kernel.apparmor_restrict_unprivileged_userns=1` が既定で、Chrome の sandbox が user namespace を作れずに即座に死ぬ。
Linux コンテナー (`oven/bun:1.4.2` に Chromium 153 を追加、uid 0 と uid 1000 の両方) で再現した。

| 起動方法 | 結果 |
| --- | --- |
| 既定 | `Error: Chrome process closed the pipe` |
| `backend.argv` に `--no-sandbox` | `evaluate` が `2` を返す |
| `backend.argv` に `--no-sandbox --disable-dev-shm-usage` | `evaluate` が `2` を返す |

コンテナーには `DISPLAY` が無いので、この成功は Bun が `--headless` を渡し続けていること、すなわち `backend.argv` が既定のフラグを置き換えるのではなく足すことも同時に示す。

**`detectWebViewSupport` が下位のエラーを捨てているために、この判別が CI のログからできなかった。**
`probe.catch(() => undefined)` は失敗の理由を落とし、残るのは「描画できなかった」という一文だけである。
原因を名乗らせるために置いた仕組みが、原因を隠していた。

## 対象範囲

- `frontend/tests/e2e/fixtures.ts` の WebView 起動を、実行するホストで成立する起動方法を選ぶ形にする。
- `detectWebViewSupport` の失敗メッセージへ、試した起動方法と Bun が返したエラー本文を載せる。
- `frontend/tests/e2e/README.md` の「実行環境の前提」を、macOS と Linux で書き分ける。
- CI の `ui-e2e` ジョブが緑になることを確認する。

## 対象外

- `mise run test-ui-e2e` をローカルの標準ゲートへ戻すこと。wi-497 の判断を変えない。
- E2E が実行ごとに結果を変える問題。wi-624 が扱う。
- Windows での実行。CI にも開発環境にも対象が無い。
- Playwright やブラウザーバイナリーのダウンロードの導入。Bun 組み込みのバックエンドで足りることが分かっている。

## 設計

### 採用する形

`detectWebViewSupport` を、起動方法の候補を順に試す探索にする。
最初に空白ページを描いて評価を返せた候補を実行全体で記憶し、`openWebView` が同じ候補で開く。

| 順 | 候補 | 成立するホスト |
| --- | --- | --- |
| 1 | 既定 (オプションを渡さない) | macOS、sandbox が働く Linux |
| 2 | `backend: { type: 'chrome', argv: ['--no-sandbox', '--disable-dev-shm-usage'] }` | AppArmor が user namespace を制限する Linux、コンテナー |

候補 1 を先に試すので、Chrome の sandbox が働くホストでは sandbox が有効なまま実行される。
`--no-sandbox` が効くのは、それ以外では実行そのものができないホストに限られる。

プラットフォームで分岐しないのは、`fixtures.ts` が既に述べている理由と同じである。
`process.platform` は「どのバックエンドが動くか」に答えず、コンテナーや WSL のような組み合わせを取りこぼす。
描けるかどうかは描かせて確かめる。

候補の探索は 1 実行につき 1 回であり、`setup.ts` がスタックの起動と並列に走らせる。
候補 1 が失敗するホストでも、Chrome は `closed the pipe` を即座に返すので追加の待ちは生じない。

### Chrome のセッション分離

2026-09-20 の CI run `35455470818` で、候補 2 は WebView の起動を成功させ、36 件を実行できた。
しかし最初の認証後、後続 26 件は新しい WebView でも前のログイン状態を引き継ぎ、`timeout waiting for page kind=login` で失敗した。

Bun の Chrome バックエンドは 1 Bun プロセスにつき Chrome を 1 つだけ起動し、後続の WebView は `Target.createTarget` で同じ Chrome を使う。
`dataStore: 'ephemeral'` はディスクへ保存しないという意味であり、同じ Chrome プロセスに属する WebView 間の Cookie を分けない。
WKWebView で成立していた「`openWebView` ごとに新しいブラウザーセッション」というテストの前提を、Chrome ではプロセスの寿命で作る必要がある。

`openWebView` が返す期限付き WebView に Chrome プロセスの lease を重ねる。
各 view の `close` を数え、最後の view が閉じたときだけ `Bun.WebView.closeAll()` を呼ぶ。
`closeAll()` の直後は閉じた pipe が残るため、非同期の `openWebView` は子プロセスの終了を待ってから新しい一時 data store の Chrome を起動する。
前のテストの Cookie は引き継がない。
複数 view が同時に生きる間はプロセスを落とさない。

同一 Chrome プロセス内の複数 view を別 Cookie jar にする機能は Bun.WebView のインターフェースに無い。
2 つの独立ブラウザーセッションを要求する 1 件は、別 WebView で先にセッション A を作って閉じ、Chrome をリセットしてから WebView B でログインする。
サーバーには A と B が残るので、B の画面から A を失効できることを引き続き観測できる。

### 失敗メッセージ

全候補が失敗したときは、候補ごとに「何を試したか」と Bun のエラー本文を並べ、macOS と Linux それぞれで確かめるべきものを述べる。
現在のように 1 つの原因を決め打ちしない。

### 採用しない案

| 案 | 採用しない理由 |
| --- | --- |
| CI の手順へ `sudo sysctl -w kernel.apparmor_restrict_unprivileged_userns=0` を足す | runner の設定だけを直す。コンテナーや devcontainer では相変わらず描画できず、原因も名乗らない。 |
| Linux では常に `--no-sandbox` を渡す | sandbox が働くホストでも無効化する。候補順にすれば失う必要が無い。 |
| `BUN_CHROME_PATH` を CI で設定する | Chrome は探索で見つかっている。見つからない場合のエラーは別の文面であり、CI の症状と一致しない。 |
| Xvfb を CI へ導入する | Linux のバックエンドは Chrome であり、画面サーバーを必要としない。効かない対処で CI の手順が増える。 |
| E2E を macOS runner へ移す | WKWebView なら動くが、Linux で動かない理由を残したまま実行費用を上げる。 |

## 計画

1. 候補の探索と失敗メッセージを `fixtures.ts` へ入れる。`openWebView` は唯一の入口のまま残す。
2. Linux コンテナーで、変更前は候補 1 で止まり、変更後は候補 2 で通ることを確認する。
3. README の「実行環境の前提」を書き直す。macOS では画面サーバーが要るためサンドボックス内では実行できないこと、Linux では Chrome の起動が条件であることを分ける。
4. CI の `ui-e2e` ジョブを観測する。
5. Chrome の共有プロセスを最後の view の close で破棄し、テスト間の Cookie を分離する。
6. 2 セッションを要するシナリオは、別 WebView と Chrome プロセスで順番に作る。

未解決の問いは無い。
原因の判別と対処の成立は、起票前にコンテナーでの実測で終えている。

## タスク

- [x] T001 [Acceptance] 変更前の失敗を観測する。Linux で既定の起動が `Chrome process closed the pipe` で落ち、現在のメッセージがそれを述べないことを記録する。
- [x] T002 [Tooling] `detectWebViewSupport` を候補順の探索にし、採用した候補を `openWebView` が使う。失敗時は候補ごとのエラー本文を述べる。
- [x] T003 [Docs] `frontend/tests/e2e/README.md` の「実行環境の前提」を macOS と Linux で書き分ける。
- [x] T004 [Acceptance] CI run `35455470818` で Chrome の起動後に 36 件が走り、10 pass / 26 fail となる RED を観測する。26 件はすべて後続 WebView が前のログイン状態を引き継いだため `page kind=login` を待って失敗した。
- [x] T005 [Unit] `webview-process-lease.test.ts` を module 不在と `waitUntilReady` 不在で RED にし、最後の view だけが process reset を所有することと、終了待ちが次の view を止めることを固定する。
- [x] T006 [Tooling] `openWebView` を非同期にし、Chrome プロセスの lease と終了待ちを返してテスト間の Cookie を分離する。
- [x] T007 [Acceptance] 複数 view の共有 Cookie に依存せず、別ブラウザーセッションを画面から失効できることを固定した。Chrome 強制の対象テストと全 E2E 38 件が通過した。
- [ ] T008 [Verify] macOS と Linux (Bun 1.4.2 / Chromium 153) の `mise run test-ui-e2e` 相当は 38 pass、`mise run verify-ui` と `mise run verify` は通過した。実 GitHub Actions の `ui-e2e` は、この変更を push した後に確認する。

## 検証

- `mise run test-ui-e2e`
- `mise run verify-ui`
- `mise run verify`
- CI の `ui-e2e` ジョブ

## リスク

リスクは low。
対象はブラウザー E2E の足回りであり、製品の実装には触れない。

`--no-sandbox` は Chrome 自身の sandbox を無効にする。
E2E が開くのは、この実行が自分で起動した `localhost` の Vite 開発サーバーと API だけであり、外部から与えられた内容を読み込む経路は無い。
それでも無効化を既定にはせず、候補 1 が成立しないホストに限る。

**誤った直し方が 1 つ危ない。**
候補の探索を、失敗を黙って飲み込む形で書くと、描画できないホストで再び原因の分からないタイムアウトへ戻る。
探索が候補を落とした理由は、最終の失敗メッセージまで運ぶ。

`reversibility` は reversible。
テストの足回りの変更であり、公開契約もデータも変えない。
