# ブラウザー E2E の実行基盤の比較

2026 年 11 月 14 日時点で、Bun.WebView と Playwright Test を比較した調査資料である。
採用済みの構成は[フロントエンド設計](../design/application/frontend.md#e2e-テスト)を参照する。
ここに示す選択肢と評価案は、採用を決めるものではない。

## 現行構成に対する判断

このリポジトリでは、Playwright Test を Bun 上で少数のシナリオに適用し、診断と状態分離を確認してから移行する案を推奨する。
推奨の理由は、自前のブラウザー共通処理を減らせる見込みがあるためである。
現行の[共通処理](../../frontend/tests/e2e/fixtures.ts)は、Chrome を優先する起動判定、CDP による Cookie と保存領域の消去、操作期限、DOM 操作と待機を所有する。
これらはすべて削除できるわけではなく、Go と Vite の起動、SMTP の受信、外部サービスの代替など、製品固有の準備は引き続き必要になる。

現在の 29 本の spec は、[共通の起動処理](../../frontend/tests/e2e/setup.ts)で一つのスタックを使う。
バックエンドはメモリ永続化であり、ブラウザーごとに Cookie を分けてもサーバーの状態は共有される。
先行評価では、OIDC の認可から callback への遷移、キーボードだけの認証操作、独立したセッションを扱う操作を優先する。
現行の障害回避策は [E2E の説明](../../frontend/tests/e2e/README.md)に記録されているが、今回の調査では再現試験をしていない。

## Bun 対応の範囲

Playwright 1.65 は、Playwright Test のテストランナーを Bun 上で実行できる。
コマンドは `bunx --bun playwright test` であり、`bunx playwright test` だけでは Node.js を使う。
Bun が TypeScript を変換するため、Playwright の Babel と変換キャッシュは使わない。
公式の継続検証は Bun 1.4.3 と Linux で行われている。
このバージョン番号は、最低対応バージョンの宣言とは区別する。
出典：[Playwright 1.65 のリリースノート](https://playwright.dev/docs/release-notes#version-165)。

この構成でも、テストは `@playwright/test` の `test` と `expect` で書き、Playwright Test が実行する。
`bun:test` のランナーへ移植する機能ではない。
Playwright Library を `bun:test` から呼ぶ構成も別に考えられるが、公式は一般的な E2E にはテストランナーを含む `@playwright/test` を推奨している。
出典：[Playwright のライブラリーとテストランナーの違い](https://playwright.dev/docs/library)。

## Bun 上で実行するときの制約

Playwright 1.65 のリリースノートには、次の差が明記されている。
出典：[Bun 上のテスト実行](https://playwright.dev/docs/release-notes#run-tests-with-bun)。

| 対象 | Bun 上の差 |
| --- | --- |
| 変更監視 | watch mode は使えない。UI mode もファイル変更時にテストを再実行しない |
| 変更したテストの選択 | `--only-changed` は import をたどらない |
| モジュールの読み込み | `--tsconfig` と `tsconfig.json` の JSX オプションは効かず、独自の ESM loader は使えない |
| スタックトレース | async frame を含まない |
| API 名 | `locator.click` ではなく `click` のようにクラス名が省かれ、一部のステップ名、位置、エラーメッセージが変わる |
| テスト側の console 出力 | trace と UI mode に入らない |

最後の制約はテストプログラムの console 出力について述べており、ブラウザー内の console 出力全般が取得不能になるという意味には広げない。
変更監視や診断情報の完全性を優先する場合は、Playwright Test を Node.js で実行する案を比較対象にする。
これは制約から導いた判断である。

## 自動化 API とテスト基盤の違い

Bun.WebView は Bun に組み込まれたブラウザー自動化 API で、公式文書は experimental としている。
macOS の標準 backend は OS 付属の WKWebView、Linux と Windows は Chrome 系ブラウザーを使う。
Chrome backend は Bun プロセスごとに一つのブラウザーを使い、各 view をタブとして作る。
永続ストレージのディレクトリも Chrome プロセス全体へ適用される。
出典：[Bun.WebView](https://bun.com/docs/runtime/webview)。

Bun.WebView にも操作前の自動待機がある。
`click(selector)` は要素の存在、大きさ、画面内の位置、安定性、中心点が隠れていないことを確認する。
したがって、「Playwright にすると初めて自動待機が得られる」という比較は正確ではない。
出典：[Bun.WebView のセレクターによるクリック](https://bun.com/docs/runtime/webview#clicking-by-selector)。

Playwright は操作前の検査に加えて、条件成立まで再試行する assertion を用意している。
Playwright Test にはテストごとの `page` と `BrowserContext`、環境の準備と破棄をまとめる fixture がある。
このため、ブラウザーの状態分離と assertion の待機処理を共通基盤へ任せられる。
出典：[操作前の待機と assertion](https://playwright.dev/docs/actionability)、[fixture](https://playwright.dev/docs/test-fixtures)、[状態の分離](https://playwright.dev/docs/browser-contexts)。

失敗時の調査では、Playwright の trace から操作、前後の DOM、スクリーンショット、通信を追える。
現在のテスト基盤で同等の情報を残していない場合、再現できない失敗の原因を調べる手段が増える。
出典：[Trace Viewer](https://playwright.dev/docs/trace-viewer)。

ただし、`BrowserContext` が分離するのはブラウザーの状態である。
共有バックエンドのテナント、ユーザー、メール受信先、callback の状態は別に分離する必要がある。
ブラウザーの分離だけを理由に既存テストを並列化しないという判断は、この区別から導かれる。

## 選択肢の比較

次の評価は、上記の機能と制約から導いた判断である。
速度差、メモリー差、実際の失敗頻度は測定していない。

| 選択肢 | 利点 | 負担と制約 | 適する条件 |
| --- | --- | --- | --- |
| Bun.WebView と `bun:test` を継続 | 移行作業が不要。追加の自動化パッケージを持たずに済む | fixture、待機、状態の初期化、診断を自前で維持する範囲が残る。API は experimental | テストが小規模で、独自基盤の保守や失敗調査に困っていない |
| Playwright Test を Bun で実行 | Bun を使いながら locator、assertion、fixture、trace、複数ブラウザーの実行基盤を使える | テスト API の移行とブラウザーバイナリーの管理が必要。Bun 固有の制約がある | Bun を維持し、独自テスト基盤の保守を減らしたい |
| Playwright Test を Node.js で実行 | ブラウザー操作と assertion を保ち、Bun 固有の変更監視と診断の制約を避けられる | Node.js を実行環境へ加え、既存の `Bun.serve` などを別プロセスへ分けるか置き換える必要がある | UI mode の変更監視や診断情報を優先する |
| Playwright Library を `bun:test` から利用 | 既存ランナーを保ちつつブラウザー操作 API を置き換えられる | page/context の生存期間、fixture、レポートなど、テストランナーが担う仕組みを別途組み立てる | ランナーを変えられない強い制約がある |

Playwright のブラウザーバイナリーは、Playwright のバージョンに対応するものをインストールする。
必要な OS ライブラリーも残るため、Bun 対応によってブラウザーの導入作業がなくなるわけではない。
出典：[ブラウザーの導入](https://playwright.dev/docs/browsers)、[CI の準備](https://playwright.dev/docs/ci)。

Playwright は Chromium、Firefox、WebKit を扱うが、Playwright の WebKit は Safari 製品そのものではない。
WebKit の main branch に由来し、Playwright のパッチを使う。
Safari に近いプラットフォーム依存の挙動まで必要なら、macOS での検証も検討する。
出典：[Playwright の WebKit](https://playwright.dev/docs/browsers#webkit)。

## 移行判断のための評価案

独自の fixture や待機処理を減らせるかを確認するため、認証、状態変更、外部 callback を含む代表的な少数のシナリオを Playwright Test と Bun で試す。
最初は一つの worker で実行し、共有バックエンドによる競合を増やさず比較する。
公式の CI ガイドも、安定性と再現性を優先する構成では `workers: 1` を推奨している。
出典：[CI の worker 数](https://playwright.dev/docs/ci#workers)。

判断には、成功時の実行時間に加えて、意図的な失敗で得られる診断情報、状態が次のテストへ漏れないこと、削除できる独自補助コードを使う。
Bun 固有の制約が必要な作業を妨げる場合は、同じ Playwright Test を Node.js で実行して差を確認する。
これは採用判断に向けた提案であり、この調査では移植、ブラウザー起動、性能測定を実施していない。
