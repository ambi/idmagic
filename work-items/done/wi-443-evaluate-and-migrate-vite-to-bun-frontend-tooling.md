---
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-08-29
priority: p2
depends_on: []
change_kind: tooling
evidence_policy: risk-based-v2
initial_context:
  specification:
    - docs/contexts/system/scenarios.md#REQ-SYSTEM-005
    - docs/contexts/system/scenarios.md#REQ-SYSTEM-006
    - docs/contexts/system/scenarios.md#REQ-SYSTEM-007
    - docs/contexts/system/glossary.md
  typespec: []
  source:
    - frontend/vite.config.ts
    - frontend/package.json
    - frontend/index.html
    - frontend/bunfig.toml
    - frontend/knip.jsonc
    - frontend/Caddyfile
    - frontend/Dockerfile
    - frontend/scripts/generate-routes.ts
    - frontend/src/vite-env.d.ts
    - frontend/src/lib/i18n/locale.ts
    - frontend/src/routes/index.tsx
    - frontend/tsconfig.app.json
    - frontend/tsconfig.check.json
    - frontend/tsconfig.node.json
    - mise.toml
    - dev.sh
  tests:
    - frontend/src/devProxy.test.ts
    - frontend/tests/e2e/fixtures.ts
  stop_before_reading:
    - backend
    - spec
    - infra/k8s
affected_spec:
  - { path: docs/contexts/system/scenarios.md, requirement: REQ-SYSTEM-005 }
  - { path: docs/contexts/system/scenarios.md, requirement: REQ-SYSTEM-006 }
  - { path: docs/contexts/system/scenarios.md, requirement: REQ-SYSTEM-007 }
---

# Vite を Bun のフロントエンド開発サーバーとバンドラーへ置き換えられるか実証し、成立する場合だけ移行する

## Motivation

`frontend/` は実行環境、パッケージ管理、単体テストに Bun 1.4.0 を使っている一方、開発サーバーと本番用静的資産の生成には Vite 8.2.2 を残している。Bun 自身の HTML import、`Bun.serve`、`Bun.build`、CSS バンドラーを使えれば、フロントエンド道具立てを減らし、開発時とビルド時の変換経路を Bun に統一できる可能性がある。

ただし Vite は現在、React Fast Refresh、Tailwind CSS、TanStack Router のファイルベースルート生成と自動コード分割、`@` alias、`import.meta.env`、開発時 API 代理、HTML と静的資産の出力に関与している。特に TanStack Router の自動コード分割はバンドラープラグインによるソース変換であり、既存の `scripts/generate-routes.ts` だけでは同等にならない。ビルドが成功することだけを根拠に置き換えると、初期 JavaScript の肥大化、設定の欠落、開発経路だけの代理漏れを見逃すため、隔離した試作と明示的な移行可否判定が必要である。

## Scope

- Bun 1.4.0 の[フルスタック開発サーバー](https://bun.com/docs/bundler/fullstack)、HTML import、`Bun.serve`、`Bun.build`、`bun-plugin-tailwind` を用いた隔離試作を、現行 Vite 経路と併存する形で作る。
- React/TSX、React Fast Refresh と HMR、Tailwind CSS、フォントと画像、`@` alias、HTML、ソースマップ、内容ハッシュ付き静的資産を現行画面で検証する。
- TanStack Router のルートツリー生成と、現在 `autoCodeSplitting: true` および `defaultBehavior: [['loader', 'component']]` が提供する遅延コード分割を維持できる方式を実証する。
- `VITE_DEV_PORT` と `VITE_API_TARGET` によるポートと代理先の上書き、realm 配下の公開済みエンドポイントを含む開発時 API 代理、SPA フォールバック、厳密なポート確保を維持する。
- `VITE_DEFAULT_LOCALE`、`VITE_DEMO_LOGIN_ENABLED`、開発サーバー判定のビルド時注入を維持し、`REQ-SYSTEM-005`、`REQ-SYSTEM-006`、`REQ-SYSTEM-007` の観測可能な振る舞いを変えない。
- 本番用 `dist/` を Caddy から配信する現在の境界、CSP、キャッシュ方針、Docker ビルド、CI、`mise run dev-ui`、`mise run build-ui`、`mise run verify-ui` を維持する。
- 下記の移行可否基準をすべて満たした場合だけ既定の開発・ビルド経路を Bun へ切り替え、`vite.config.ts`、Vite の型参照、Vite と Vite 専用プラグインを除去する。
- 移行不可の場合は試作コード、試作用スクリプト、試作用依存を撤去し、測定値と阻害条件だけを本 Work Item の完了記録に残す。

## Out of Scope

- 本番の静的資産配信を Caddy から Bun サーバーへ変更すること。
- `VITE_DEFAULT_LOCALE`、`VITE_DEMO_LOGIN_ENABLED`、`VITE_DEV_PORT`、`VITE_API_TARGET` の名前変更。Vite 由来の接頭辞は既存設定との互換性のため本変更では維持し、名称整理は別 Work Item とする。
- TanStack Router、Tailwind CSS、React、Caddy、Bun 自体の採否やメジャーバージョンを見直すこと。
- TanStack Router のバンドラープラグインが行う変換を、リポジトリ固有の大規模な独自プラグインとして再実装すること。
- 製品の API、認証、画面の意味、利用者向け文言、対応ブラウザー範囲を変更すること。
- 移行不可と判断した場合に、要件を弱めたり Vite と Bun の二つを恒久的な既定経路として保守したりすること。

## Design

### 現行基準線

比較対象は `frontend/vite.config.ts` と `frontend/package.json` の現在の `dev` と `build` の経路とする。試作前に、開発サーバーが最初の HTML を返すまでの時間、本番ビルド時間、出力ファイル一覧、初期画面で取得する圧縮前 JavaScript と CSS の合計、ルート遷移時に追加取得するチャンク、代表画面の HMR を同一マシンで記録する。性能値はキャッシュなしとキャッシュありを各 5 回測り、中央値で比較する。

### 試作の境界

既定の `dev` と `build` は移行可否を確定するまで Vite のまま保つ。Bun 側は一時的な `dev:bun` と `build:bun` から起動し、開発時は HTML import を `Bun.serve` の静的ルートとして渡す小さな TypeScript エントリ、本番ビルド時は `index.html` を入力する `Bun.build` の JavaScript API として試す。二つの経路は Tailwind プラグインと公開定数の設定を共有する。開発時の API 代理判定は URL を受けて「Bun が処理する静的ルート」「Go バックエンドへ転送するリクエスト」のどちらかを返す純粋な関数へ分離し、その関数と実 HTTP 境界の双方をテストする。製品 API を Bun 側へ移さない。

Tailwind CSS は Bun 公式の `bun-plugin-tailwind` を第一候補とする。TanStack Router は既存の `@tanstack/router-generator` によるルートツリー生成を残すが、[自動コード分割の公式資料](https://tanstack.com/router/latest/docs/guide/code-splitting)が示すとおり CLI 生成だけでは自動コード分割にならないため、上流が対応するバンドラープラグイン、またはルートのソースを限定的かつ機械的に `.lazy.tsx` または `Route.lazy()` へ分ける方式で同じ遅延境界を作る。TanStack Router の内部変換を模倣する独自 Bun プラグインが必要な場合は移行不可とする。

`VITE_DEFAULT_LOCALE` と `VITE_DEMO_LOGIN_ENABLED` は公開してよい変数だけをビルド時の `define` で明示的に埋め込み、環境全体をクライアントへ埋め込まない。開発中かどうかも Bun のサーバープロセスが明示的な公開定数として注入し、ブラウザー側で Bun のプロセス環境を直接参照しない。これにより既存設定名と規範上の振る舞いを保ちながら `vite/client` と Vite 固有の `import.meta.env.DEV` 型を除去する。

### 移行可否基準

次を一つでも満たせない場合は移行不可とし、Vite を維持する。

- `mise run verify-ui` と `mise run test-ui-e2e` が、試作の Bun 開発サーバーおよび Bun 生成物に対して既存のオラクルを弱めず成功する。
- `REQ-SYSTEM-005`、`REQ-SYSTEM-006`、`REQ-SYSTEM-007` の設定済み、未設定、開発時、本番ビルドの各分岐が同じ結果になる。
- realm 配下のエンドポイントを含む `frontend/src/devProxy.test.ts` の全経路が実 HTTP リクエストでも Go バックエンドへ転送され、クエリ、メソッド、ヘッダー、本文、状態、応答ヘッダーを失わない。該当しない SPA ルートと静的資産はバックエンドへ転送しない。
- React のコンポーネントと CSS の編集が HMR で反映され、代表的なフォームの入力状態を React Fast Refresh が維持する。HMR が失敗した場合は明示的にページ全体を再読込し、古い資産を表示し続けない。
- 本番出力で初期ルートに不要な `component` と `loader` が初期 JavaScript から分離され、画面遷移時に対応するチャンクが遅延取得される。上流対応または限定的なソース分割で実現でき、TanStack Router の内部変換を複製する独自プラグインを必要としない。
- `dist/` は Caddy の SPA フォールバック、`/assets/*` の不変キャッシュ、HTML の `no-store`、現在の CSP で配信できる。インラインのスクリプトやスタイル、外部 CDN、実行時の Bun サーバーを本番に要求しない。
- フォント、画像、CSS の `url()`、動的 import を含む全資産が内容ハッシュ付きで解決され、ブラウザーコンソールに読込エラー、ソースマップエラー、hydration error を出さない。
- キャッシュなしとキャッシュありの開発起動時間と本番ビルド時間の中央値、および初期画面の JavaScript と CSS の合計が、Vite 基準線からそれぞれ 10% を超えて悪化しない。
- 切替後の依存宣言、ソース、型設定、lockfile に `vite`、`@vitejs/plugin-react`、`@tailwindcss/vite`、`@tanstack/router-plugin/vite`、`vite/client`、`vite.config` の参照が残らない。Vite 専用パッケージを残さず、Bun と共有するパッケージまで誤って除去しない。
- Bun 固有実装は開発サーバー用エントリ、ビルド用エントリ、共有する公開設定と代理規則に限定され、現行 Vite 設定より保守対象を増やさない。

### 事前に定める RED 検査

本 Work Item は `change_kind: tooling` であり、製品の観測可能な振る舞いを変えないことが目標である。したがって製品要件を新設した RED は作らず、次を事前に宣言する。

- Acceptance RED: `N/A: 製品要件を追加しないため`。代替として、Bun 経路の実 HTTP 開発サーバーに対する `frontend/tests/e2e/` の既存 E2E と、Bun 生成 `dist/` に対する `mise run build-ui` を、Bun 経路が未実装の段階で走らせて失敗を観測する。
- Unit RED: `frontend/src/devProxy.test.ts` を、Vite 設定オブジェクトの走査ではなく代理判定の純粋関数と実 HTTP 転送に対する検査へ移し、判定関数が存在しない段階で失敗を観測する。移行不可と判定した場合は、この検査自体を撤去する。

### 選ばない案

- 開発サーバーだけ、または本番バンドラーだけを Bun にして Vite を残す案は、二つの変換経路を恒久保守するため選ばない。いずれか一方しか成立しない場合は移行不可とする。
- ルート単位の遅延取得を失って単一バンドルにまとめる案は、現在の性能特性を静かに変えるため選ばない。
- 動作させるために既存 E2E のアサーション、CSP、設定分岐、API 代理範囲を弱める案は選ばない。
- Bun の `env: "inline"` で全環境変数をブラウザーのバンドルへ埋め込む案は、秘密値を混入させ得るため選ばない。

## Plan

1. Vite 基準線と、Vite が担う変換、開発サーバー、出力契約を棚卸しする。
2. 既定経路を変えず、Bun の HTML ルート、Tailwind プラグイン、公開変数の `define`、API 代理、本番ビルドを最小構成で試作する。
3. TanStack Router のルートツリー生成と遅延コード分割について、上流対応を優先し、次に限定的なソース分割を試す。独自の変換器が必要と判明した時点で移行不可とする。
4. 既存検証、実 HTTP 代理試験、HMR 手動試験、本番 Caddy 配信、バンドル構成、性能を同じ入力で比較し、移行可否基準を表として記録する。
5. 移行不可なら試作を撤去して Vite 経路を再検証し、阻害条件と再評価可能になる上流条件を Completion に記録して終了する。
6. 移行可能なら、実装を切り替える前に `REQ-SYSTEM-006`、`REQ-SYSTEM-007` と `glossary.md` の Vite 固有表現を「リポジトリのフロントエンド開発サーバー」に更新する。`REQ-SYSTEM-005` と既存 `VITE_*` 設定名の意味は変えない。
7. `dev` と `build`、`mise` タスク、E2E フィクスチャ、Docker ビルドを Bun 経路へ切り替え、Vite の設定、型、依存を除去する。
8. Git の追跡対象外である TypeSpec と HTML の生成物を必要に応じて再生成し、全検証と移行後の測定を行う。

## Tasks

- [x] T001 [Baseline] Vite が担う機能、依存、設定、代理経路、出力資産を棚卸しし、キャッシュなしとキャッシュありを各 5 回測って性能とバンドルの基準線を記録する。
- [x] T002 [Prototype] 既定経路と併存する Bun の HTML ルート、開発サーバー、Tailwind CSS、本番ビルド、公開環境変数注入を実装する。
- [x] T003 [Proxy] API 代理規則を純粋な判定から Bun の実 HTTP 転送までテストし、並列 E2E 用のポートと代理先の上書きを維持する。
- [x] T004 [Routing] ルートツリー生成と `component` および `loader` の遅延コード分割を実証し、独自変換器なしで維持できるか確定する。
- [x] T005 [Acceptance] HMR / React Fast Refresh、設定分岐、Caddy 配信、静的資産、既存 UI E2E、性能を比較し、移行可否基準の各行に証跡を残す。
- [x] T006 [Decision] 移行可否を確定する。移行不可なら試作を完全に撤去して Vite 経路を再検証し、T010 へ進む。
- [x] T007 [Spec] 移行不可のため実施しない。`REQ-SYSTEM-006`、`REQ-SYSTEM-007`、`glossary.md` の Vite 固有表現は現行実装をそのまま指しており、変更しない。
- [x] T008 [Cutover] 移行不可のため実施しない。
- [x] T009 [Cleanup] 移行不可のため実施しない。Vite 依存は現行の既定経路として維持する。
- [x] T010 [Verify] 試作を撤去した作業ツリーで Vite 経路を再検証し、全検証を通す。

## Findings

### 基準線と試作の測定値

同一マシン、同一入力での測定。Vite はキャッシュなし (`node_modules/.vite` 削除) とキャッシュありを各 5 回、Bun は永続キャッシュを持たないため 5 回で、いずれも中央値を採る。

| 指標 | Vite 8.2.2 (基準線) | Bun 1.4.0 (試作) | 差 |
|---|---|---|---|
| 開発サーバーが最初の HTML を返すまで (キャッシュなし) | 357 ms | 226 ms | -37% |
| 開発サーバーが最初の HTML を返すまで (キャッシュあり) | 288 ms | 226 ms | -22% |
| 本番ビルド (キャッシュなし) | 775 ms | 183 ms | -76% |
| 本番ビルド (キャッシュあり) | 773 ms | 184 ms | -76% |
| 初期画面の JavaScript | 337,244 B (`index` + modulepreload) | 1,557,081 B | +362% |
| 初期画面の CSS | 79,844 B | 397,389 B | +398% |
| JavaScript チャンク数 | 189 | 1 | 遅延取得なし |
| 内容ハッシュ付きフォント | 7 ファイル 218,512 B | 0 (CSS へ data URI として埋め込み) | 資産として出力されない |

Bun 側の本番ビルドは `process.env.NODE_ENV` を明示しないと React の開発ビルドを取り込む。上表は `define` で `production` を与えた後の値である。

### 移行可否基準の判定

| 基準 | 判定 | 証跡 |
|---|---|---|
| `mise run verify-ui` と `mise run test-ui-e2e` が Bun 経路で既存オラクルのまま成功する | 不成立 | Bun 開発サーバーでは `tests/e2e/ui-scenario-smoke.spec.ts` の 3 シナリオすべてが失敗した (Vite では 3 件成功)。画面が空のまま起動しない。 |
| `REQ-SYSTEM-005` / `-006` / `-007` の各分岐が同じ結果になる | 条件付き成立 | 本番ビルドでは `define` により `VITE_DEMO_LOGIN_ENABLED` 未設定で `demoEnabled:!1`、`true` で `demoEnabled:!0` になった。開発サーバーでは `import.meta.env.DEV` が `true` に解決され `demoEnabled: true` になった。ただし `import.meta.env.VITE_*` は Bun がどの設定でも埋め込まず、`hmr.importMeta.env.VITE_DEFAULT_LOCALE` という実行時参照のまま残る。`process.env.VITE_*` へ書き換えれば `bunfig.toml` の `[serve.static] env = "VITE_*"` で埋め込まれることは別途確認した。 |
| realm 配下を含む代理経路が実 HTTP で欠落なく転送される | 成立 | 判定の純粋関数と Bun 開発サーバーの実 HTTP 転送の双方で、公開エンドポイント 13 経路がメソッド、クエリ、ヘッダー、本文、状態、応答ヘッダーを保って転送され、SPA ルート 4 経路はバックエンドへ送られなかった。ただし Bun の `routes` は `fetch` ハンドラーより先に一致するため、`"/*"` の HTML ルートだけでは全 API が SPA へ吸われる。公開セグメントごとに明示ルートを登録して初めて成立した。 |
| HMR と React Fast Refresh が効き、失敗時は全体を再読込する | 判定不能 | 開発バンドルに `hmr.reactRefreshAccept` が 229 箇所あり Fast Refresh 自体は組み込まれるが、画面が起動しないため入力状態の維持を観測できなかった。 |
| 初期ルートに不要な `component` と `loader` が初期 JavaScript から分離される | 不成立 | 下記「TanStack Router のコード分割」を参照。 |
| `dist/` を現在の Caddy 設定と CSP で配信できる | 不成立 | 生成物を実際の `frontend/Caddyfile` で配信したところ、SPA フォールバック、`/assets/*` の不変キャッシュ、HTML の `no-store`、CSP ヘッダーはそのまま機能した。しかし CSS に `url(data:font/woff2;base64,...)` が 7 件含まれ、同じ応答が返す `font-src 'self'` はこれを許可しない。 |
| 全資産が内容ハッシュ付きで解決される | 不成立 | Bun の CSS バンドラーは `url()` の参照先を必ず data URI へ埋め込む。`Bun.build` の `loader` で `.woff2` を `file` にしても変わらず、`BuildConfig` に埋め込みしきい値に相当する設定は無い。 |
| 開発起動時間と本番ビルド時間、初期画面の JavaScript と CSS が基準線から 10% を超えて悪化しない | 不成立 | 時間は改善するが、初期 JavaScript は +362%、CSS は +398%。 |
| 切替後に Vite 参照が残らない | 判定不要 | 移行しないため該当しない。 |
| Bun 固有実装が現行 Vite 設定より保守対象を増やさない | 不成立 | 代理の明示ルート登録に加え、コード分割には上流の内部フックを叩くアダプターが要る。 |

### TanStack Router のコード分割

`@tanstack/router-plugin@1.168.35` は `./vite`、`./webpack`、`./rspack`、`./esbuild` を配布するが Bun 用の入口を持たない。上流の esbuild 版プラグインを `Bun.build` の `plugins` へそのまま渡すと、例外は出ないまま完全な no-op になった。プラグインの有無で出力は 1 チャンク 4,167,755 B と一致した。

自前で橋渡しする経路も、公開 API では閉じている。`createRouterPluginContext` は公開されている一方、それを受け取る `createRouterGeneratorPlugin` と `createRouterCodeSplitterPlugin` は `exports` マップで塞がれており、公開されている `unpluginRouterGeneratorFactory` と `unpluginRouterCodeSplitterFactory` はそれぞれ内部で別々のコンテキストを作る。両者が `routesByFile` を共有できないため、分割側の `transform` はどのルートに対しても `null` を返す。加えて分割側の設定初期化はバンドラー別フック (`vite` / `rspack` / `webpack`) の中だけで走るので、Bun から使うには擬似 compiler を渡して私的なフックを起こす必要がある。これは Design が移行不可と定めた「内部変換を複製する独自プラグイン」に当たる。

Design が代替として挙げた限定的なソース分割も成立しない。`src/routes` 配下で `createFileRoute` を使うファイルは 95 個あり、そのすべてが `component` か `loader` を宣言している。`.lazy.tsx` へ機械的に割るとファイル数はおよそ倍になり、「限定的」の範囲に収まらない。

### 起動しない原因

Bun 開発サーバーで配信した画面は `TypeError: null is not an object (evaluating 'import_load_client2.replaceRouteChunk')` で停止する。`@tanstack/router-core` の `router.js` と `load-client.js` は相互に import する循環で、Bun の HMR モジュールラッパーはこの循環を解決できず名前空間が `null` のまま評価される。同じ循環は Vite でも `Bun.build` の本番ビルドでも問題にならないため、開発サーバー固有の制約である。

### 副次的な発見

- `bun-plugin-tailwind@0.1.2` は Tailwind CSS 4.1.14 を同梱した自己完結のコンパイラーであり、リポジトリが固定している `tailwindcss@4.3.3` を使わない。採用すると宣言した依存と実際に CSS を生成する実装が食い違う。
- `Bun.build` は `target: "browser"` でも `process.env.NODE_ENV` を置き換えない。明示しないと React の開発ビルドが本番成果物に入る。

### 再評価できる条件

次がすべて満たされたときに再評価する価値がある。

- `@tanstack/router-plugin` が Bun 向けの入口を配布するか、少なくとも生成器とコード分割器がプラグインコンテキストを共有できる公開 API を出す。
- Bun の CSS バンドラーが `url()` の参照先をファイルとして出力する選択肢を持つ。
- Bun の開発サーバーが循環 import を含む依存を正しく評価する。

## Verification

- `mise run check-spec`
- `mise run spec-render`（移行可能と判断して仕様を変更した場合）
- `mise run build-ui`
- `mise run test-ui-unit`
- `mise run test-ui-e2e`
- `mise run verify-ui`
- `mise run check-ui-dependencies`
- `mise run check-work-items`
- `mise run check-ids`
- `mise run verify`
- 手動: 同じ React フォームと CSS を編集し、Vite と Bun で HMR、Fast Refresh による入力状態維持、失敗時のページ全体の再読込を比較する。
- 手動: `VITE_DEFAULT_LOCALE` と `VITE_DEMO_LOGIN_ENABLED` の設定および未設定と、開発および本番ビルドの組み合わせを実ブラウザーで確認する。
- 手動: 初期画面と代表ルート遷移のネットワークログから、初期バンドルと遅延チャンクの取得境界を比較する。
- 手動: Bun 生成 `dist/` を本番 Caddy 構成で配信し、ディープリンク、更新、静的資産キャッシュ、CSP、コンソールエラーを確認する。
- 測定: キャッシュなしとキャッシュありを各 5 回測った開発起動時間と本番ビルド時間、および初期画面の JavaScript と CSS の合計を基準線と比較する。

## Risk Notes

リスクは `medium`。変更自体は Vite 経路へ戻せるが、バンドラー差により未読込のルート、Tailwind の生成 CSS、フォントや画像、環境変数置換、動的 import が一部だけ壊れてもビルドと単体テストは成功し得る。開発時 API 代理の漏れは OAuth/OIDC、SAML、SCIM など一部のエンドポイントだけを壊し、別オリジンへの誤転送はリクエスト本文や認可情報を意図しない宛先へ送る危険がある。

対策として、既定経路を判断前に置き換えず、ルートの遅延取得、実 HTTP 代理、本番 Caddy 配信、設定分岐を観測境界で比較する。全環境変数の埋め込み、独自 TanStack 変換器、要件やアサーションの緩和は許可しない。移行可否基準を一つでも満たせない場合は、部分移行を残さず Vite を維持する。

## Completion
- **Completed At**: 2026-08-30
- **Summary**:
  `mise run spec-diff` は `no normative specification change against main` を返す。移行可否を実証した結果は移行不可であり、既定の開発サーバーと本番バンドラーは Vite 8.2.2 のまま変えていない。製品の観測可能な振る舞い、公開契約、設定名、CSP、配信境界に差分は無い。リポジトリに残るのは本 Work Item の測定値と阻害条件の記録だけで、試作のコード、スクリプト、依存、`bunfig.toml` の追記、E2E フィクスチャの切替は作業ツリーから完全に撤去した。
- **Acceptance RED Evidence**:
  - **Test**: `bun test tests/e2e/ui-scenario-smoke.spec.ts`（`IDMAGIC_UI_SERVER=bun` で Bun 開発サーバーを起動）
  - **Requirement**: N/A: 製品要件を追加しない tooling の Work Item であるため。代替として既存の UI E2E を Bun 経路に対して走らせた。
  - **Observed Failure**: 3 シナリオすべてが `timeout waiting for page kind=... body=` で失敗した。ブラウザーコンソールには `TypeError: null is not an object (evaluating 'import_load_client2.replaceRouteChunk')` が出ており、画面がまったく描画されない。同じ検査は Vite 経路では 3 件とも成功する。
  - **Detection Reason**: この検査は「ビルドが成功したか」ではなく「実ブラウザーで画面が所定の種別まで到達したか」を見る。バンドラーの差でモジュール評価順だけが壊れた場合、`Bun.build` は成功し単体テストも通るため、観測境界をブラウザーの描画に置かなければ見逃す。実際に本 Work Item ではビルド成功と E2E 失敗が同時に起きた。
- **Unit RED Evidence**:
  - **Test**: 試作した代理判定の検査（純粋関数 `devRoute` の 17 経路と、Bun 開発サーバーへ実 HTTP を張った 17 経路の転送）
  - **Requirement**: N/A: 製品要件を追加しない tooling の Work Item であるため。
  - **Observed Failure**: 判定関数を実装する前の初期形では、`Bun.serve` の `routes` に `"/*": index` だけを置いたため `/api/branding`、`/token`、`/realms/default/scim/v2` などの公開エンドポイントがすべて 200 の SPA HTML を返し、バックエンドへ到達しなかった。公開セグメントごとの明示ルートを加えて初めて 13 経路すべてが 203 とエコー本文を返した。
  - **Detection Reason**: 状態コードだけを見る検査では SPA の 200 と転送成功を区別できない。この検査はバックエンドのエコー応答からメソッド、パス、クエリ、ヘッダー、本文、応答ヘッダーを突き合わせ、さらに SPA ルートがバックエンドへ漏れないことも見るため、「全部転送する」「全部 SPA にする」のどちらの誤実装も落ちる。
- **Change-Resistance Results**:
  移行不可という結論が、設定の詰め方次第で覆る種類のものかを確かめるため、代表的な「うまくいったように見える」実装を作って測り直した。
  - フォントを内容ハッシュ付きで出す設定があるはずだという想定で `Bun.build` の `loader` に `{'.woff2': 'file'}` を与えたが、CSS 出力は 397,389 B のまま変わらず data URI が 7 件残った。最小の CSS 1 ファイルと 1 つの woff2 だけの再現でも同じで、`BuildConfig` に該当する設定が無いことを確認した。判定は覆らない。
  - 上流の esbuild 版プラグインを渡せばコード分割が効くはずだという想定を、プラグインの有無で同じ入力をビルドして比較した。どちらも 1 チャンク 4,167,755 B で完全に一致し、no-op であることが分かった。判定は覆らない。
  - `process.env.NODE_ENV` を与えないままの測定は Bun 側に不利な数値を出す。`define` で `production` を明示して測り直したところ初期 JavaScript は 1,957,509 B から 1,557,081 B へ下がったが、基準線 337,244 B に対してなお +362% であり判定は覆らない。この一件だけは誤った測定が誤った結論を導き得たため、修正後の値を採用している。
- **Verification Results**:
  - `mise run verify` - passed
  - `mise run test-ui-e2e` - passed (24 tests across 4 specs)
  - `mise run spec-diff` - no normative specification change against main
