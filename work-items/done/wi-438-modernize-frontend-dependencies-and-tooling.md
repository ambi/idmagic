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
  source:
    - frontend/package.json
    - frontend/bunfig.toml
    - frontend/biome.json
    - frontend/vite.config.ts
    - frontend/components.json
    - frontend/tsconfig.app.json
    - frontend/tsconfig.check.json
    - frontend/src/styles.css
    - frontend/scripts/generate-routes.ts
    - frontend/Dockerfile
    - mise.toml
    - .github/workflows/idmagic-ci.yaml
  tests:
    - frontend/src/test
    - frontend/tests/e2e
    - tools/check/src/mise-config.test.ts
  stop_before_reading:
    - backend
    - spec
    - docs/contexts
spec_impact: { kind: none, reason: "フロントエンドの依存宣言、開発ツール設定、検証経路を保守する変更であり、製品のモデル、公開インターフェース、認証、認可、利用者から観測できる振る舞いを変更しない。" }
---

# フロントエンドの依存宣言と開発設定を現行のツールチェーンへそろえる

## Motivation

`frontend/package.json` は `shadcn` を実行時依存として宣言しているが、生成済みコンポーネントは `frontend/src/components/ui` に置かれ、パッケージを使うのは `shadcn/tailwind.css` を解決するビルドとコンポーネント生成 CLI である。この役割なら `shadcn` は開発依存に置くべきであり、[[wi-313-remove-unused-radix-and-hugeicons-dependencies]] でも後続課題として明記されていた。

依存宣言と設定は個別の更新で積み重なっており、直接依存が実際の import、CSS、設定、スクリプトのどれに必要なのかを一度に検査する仕組みが無い。実際、TanStack Router の実行時パッケージと Vite プラグインは異なる版を指し、リポジトリ全体の `verify` が実行する `typecheck-ui` をフロントエンド専用の `verify-ui` は実行しない。`frontend/README.md` にも移行済みの Radix UI と Vitest が残っているため、マニフェスト、設定、検証経路、文書の間で現在の構成が一致していない。

Biome、Vite、TypeScript、Tailwind CSS、TanStack Router の更新は、パッケージの版だけを変えて完了にはできない。設定スキーマ、プラグイン API、型検査対象、生成ルート、CSS のカスタムバリアント、Bun で CLI を起動する経路が相互に依存するため、ロックファイルと検証入口を含む一つの保守変更として扱う。

## Scope

- `frontend/package.json` の全直接依存を、ブラウザー向けコードまたはスタイル、ビルド、生成、整形、静的解析、型検査、テストのどこから使うかで棚卸しし、`dependencies` と `devDependencies` を整理する。
- `shadcn` を `devDependencies` へ移し、`shadcn/tailwind.css` を使うビルド時依存と、ローカルに固定したコンポーネント生成 CLI という二つの役割を保つ。
- 未使用の直接依存を削除し、推移依存へ暗黙に依存しているパッケージを直接依存へ追加する。依存の過不足を継続的に検査するため、Knip を開発依存として導入し、実際のソース、CSS、設定、テスト、スクリプト、生成入口に合わせて依存検査だけを有効にする。
- 着手時点で利用可能な安定版を基準に、フロントエンドの直接依存と Bun を互換性のある版へ更新し、私用アプリケーションである `idmagic-ui` の全直接依存を範囲指定せず固定して `frontend/bun.lock` を再生成する。React と型定義、TanStack Router と Vite プラグイン、Happy DOM と登録器のように協調して動くパッケージは同じ更新単位で互換性を確認する。
- `frontend/biome.json`、`frontend/vite.config.ts`、`frontend/tsconfig*.json`、`frontend/components.json`、`frontend/src/styles.css` を、更新後のパッケージが定める設定形式へ移行する。必要な場合に限り、更新後の API へ追随する最小限のソースコードとテストの修正、ルートツリーの再生成を含める。
- パッケージスクリプトを読み取り専用検査と書き込み処理に分け、`mise.toml` に依存整合性検査と、固定版の shadcn CLI でコンポーネントを追加する入口を追加する。`mise run verify-ui`、リポジトリ全体の `mise run verify`、CI が同じフロントエンド検査を実行するようにそろえる。
- `mise.toml` の Bun、`frontend/package.json` の `packageManager`、`frontend/Dockerfile` のビルダーイメージを同じ版にそろえ、固定ロックファイルからのインストールがマニフェストを書き換えないことを検査する。
- `frontend/README.md` の採用ライブラリ、テストランナー、Vite の起動方法、shadcn コンポーネントの追加手順を実装後の状態へ更新する。
- `frontend/bunfig.toml` の `preload` が E2E にも適用されて `mise run test-ui-e2e` が起動待ちで必ず失敗する状態を直す。テスト環境の設定が検証入口を壊している事例であり、本項目が E2E を完了条件に置く以上ここで解決する。

## Out of Scope

- 製品仕様、公開 API、認証または認可の変更。
- UI の見た目、操作、アクセシビリティ契約の意図的な変更、および新しい shadcn コンポーネントの追加。
- React、Base UI、TanStack Router、Tailwind CSS、Vite、Biome、Bun を別の技術へ置き換える評価。
- `tools/package.json`、Go モジュール、mise が管理するフロントエンド以外のツールの更新。
- Knip による未使用ファイル、未使用 export、アプリケーション内部の到達不能コードの一括削除。本項目では依存宣言の過不足だけを検査対象にする。
- すべての開発依存を本番コンテナへ残すこと。開発依存はビルド段階だけに存在し、配信用成果物には Vite が生成した静的ファイルだけを残す現在の多段ビルドを維持する。
- Renovate のリポジトリ全体の更新方針変更。更新後もフロントエンド依存を検出できることの確認と、必要最小限の設定修正だけを含める。

## Design

### 依存の所有場所

ブラウザー向けソースから実行時に参照され、Vite が成果物へ組み込むライブラリは `dependencies` に置く。ビルド、生成、整形、静的解析、型検査、テストのためだけに実行するパッケージは `devDependencies` に置く。CSS をビルドへ供給し、CLI も提供する `shadcn` は後者とし、`frontend/src/styles.css` の `@import "shadcn/tailwind.css"` は維持する。

直接依存の要否は TypeScript の import だけでは判定しない。CSS の `@import`、Vite と Biome の設定、Bun のテスト初期化、パッケージスクリプト、生成器も入口として数える。Knip には[依存関係に関する問題種別](https://knip.dev/reference/issue-types)だけを検査させる。未使用ファイルと未使用 export まで同時に有効にする案は、依存マニフェストの整理からソース構造の変更へ範囲が広がるため採らない。

入口そのものは `knip.jsonc` へ書き写さない。Knip のプラグインが `vite.config.ts`、`bunfig.toml`、`package.json` のスクリプト、`tsconfig*.json` の `types`、`components.json` から実在の入口を導出し、書き写した入口を冗長として報告するためである。二重に持てば、設定が動いた側だけが更新されて食い違う。書き写す代わりに、導出元のファイルと各々が与える入口を `knip.jsonc` の注釈に残し、Knip が導出できない入口が現れたときだけ設定へ足す。

shadcn CLI は `@latest` を都度取得せず、`devDependencies` に固定した版を `mise run add-ui-component -- <component...>` から実行する。[公式 CLI](https://ui.shadcn.com/docs/cli)が複数コンポーネントの追加と `components.json` による生成先の設定を提供しているため、この入口から生成されたソースと必要な直接依存を通常の差分としてレビューできる。`npx shadcn@latest` を文書化する案は、Bun に統一した実行環境と固定版による再現性の両方を外れるため採らない。

### バージョンと設定の更新

`idmagic-ui` は公開ライブラリではなく `private: true` のアプリケーションなので、互換範囲を利用者へ伝える必要が無い。全直接依存を範囲指定せず固定し、`bun.lock` で推移依存まで固定する。更新は Renovate または明示的な保守作業で行い、同じ機能を構成するパッケージの版が意図なくずれないようにそろえる。Bun、整形と静的解析、TypeScript、Vite とプラグイン、CSS ツール、UI 実行時依存、テスト環境の順に小分けして更新し、各単位で型検査、単体テスト、ビルドを通す。これにより、最後にまとめて失敗して原因が複数の更新へまたがる状態を避ける。

Biome は更新後の正確な schema URL に合わせ、公式の設定移行で名前が変わった項目や廃止された項目を直す。新しい推奨規則が既存コードを検出した場合は、振る舞いを変えない修正で解消できるものだけを同じ変更に含める。規則を無効化する場合は、対象を狭めて理由を設定の近くに残す。利用可能になった規則を理由なくすべて有効化する案は、ツール更新とコード規約の変更を区別できなくなるため採らない。

Vite とプラグインの設定は、API プロキシの対象、`strictPort`、`cssCodeSplit: false`、`@` alias、TanStack Router の自動コード分割、Bun による Node.js 非依存の起動を不変条件とする。TypeScript は製品ビルド用とリポジトリ全体の型検査用の分離を維持し、`typecheck-ui` が `src`、`tests`、`scripts`、`vite.config.ts` をすべて検査する。重複するコンパイラーオプションは共通設定へ寄せられる場合だけ整理し、ビルド対象と検査対象を再び混ぜない。

Tailwind CSS と shadcn の設定更新では、Base UI が出す属性に対応する `data-open:`、`data-closed:`、`data-disabled:` とアニメーション用ユーティリティが生成後の CSS に残ることを確認する。[公式 CLI の `eject` の説明](https://ui.shadcn.com/docs/cli#eject)どおり、`shadcn/tailwind.css` をリポジトリへ複製してパッケージを削除すると、将来の CLI 更新が共有 CSS へ反映されなくなる。この同期を手作業で所有する案は採らない。

### 検証入口

`verify-ui` に `typecheck-ui` と新しい依存整合性検査を含め、リポジトリ全体の `verify` と CI にも同じ検査を明示する。E2E は実行時間と外部プロセスの起動を伴うため既存どおり標準の `verify` には入れないが、本項目の完了時には一度通し、Vite のプロキシ、生成ルート、ブラウザー上の主要操作が更新前と同じことを確認する。

`mise.toml`、`packageManager`、Dockerfile の Bun の版ずれは、既存の mise 設定検査へ不変条件を追加して検出する。人が三つのファイルを見比べるだけの確認は次の更新で再び漏れるため、完了時の一度きりの確認にはしない。

### Bun test の preload と E2E

`frontend/bunfig.toml` の `[test] preload` は `bun test` の全実行に効くため、E2E も Happy DOM を登録した状態で走る。Happy DOM の `fetch` は登録した文書オリジン（`http://localhost:3000`）を基準に CORS を適用するので、テスト側から API の `http://localhost:8082/health` を読む起動待ちが必ず遮断され、`mise run test-ui-e2e` は 125 秒のタイムアウトで失敗する。これは単体テスト移行時に入った設定の副作用であり、パッケージ版とは独立している。

修正はブラウザーの `fetch` を Happy DOM から奪い返す方向で行う。`GlobalRegistrator.register()` の前に Bun 本来の `fetch` を退避し、登録後に戻す。単体テストが相対 URL を渡す経路は `src/test/setup.ts` の包み込みが `window.location.href` を基準に絶対 URL へ解決してから委譲することで保つ。E2E だけ別の `bunfig.toml` で起動する案は、`bun test` に設定ファイルを差し替える引数が無く、作業ディレクトリを変える回避策がテストの相対パスを壊すため採らない。単体テストで CORS を模す価値は無く、遮断は製品の欠陥ではなくテスト環境の副作用としてしか現れない。

## Plan

1. 現在の `mise run verify-ui`、`mise run test-ui-e2e`、`mise run audit-dependencies` を通し、直接依存ごとの利用入口、更新可能版、マニフェストと設定の不一致を記録する。`shadcn` が `dependencies` にある現在の状態を Acceptance RED として保存する。
2. Knip、依存整合性検査のパッケージスクリプトと mise タスク、固定版 shadcn CLI の mise タスクを追加する。実在する入口を設定して既存コードに対する指摘を分類し、依存以外の指摘は設定で対象外にする。
3. `shadcn` を `devDependencies` へ移し、未使用依存の削除と不足する直接依存の追加を行う。直接依存の一覧とロックファイルだけを先に安定させる。
4. Bun と直接依存を協調する単位ごとに更新し、各単位で `mise run typecheck-ui`、`mise run test-ui-unit`、`mise run build-ui` を通す。Bun を更新する場合は mise、`packageManager`、Dockerfile を同じ差分で変更する。
5. 更新後の Biome、Vite、TypeScript、Tailwind CSS、shadcn、テスト環境の設定形式へ移行し、必要最小限の互換修正とルート再生成を行う。プロキシ、コード分割、CSS バリアント、型検査範囲の不変条件を確認する。
6. `verify-ui`、リポジトリ全体の `verify`、CI に依存整合性検査と型検査を同じ構成で配線し、Bun の版整合を検査で固定する。`frontend/README.md` を実装後の構成とコマンドへ合わせる。
7. 固定ロックファイルから再インストールして差分が生じないことを確認し、単体、ビルド、E2E、脆弱性検査、リポジトリ全体の検証を通す。代表的な誤った依存宣言を一時的に入れて、依存整合性検査が失敗することも確認する。

## Tasks

- [x] T001 [Baseline] 現在のフロントエンド検証、E2E、脆弱性検査を通し、直接依存の利用入口と更新候補を記録する。`jq -e '.dependencies.shadcn == null and .devDependencies.shadcn != null' frontend/package.json` が失敗することを Acceptance RED として記録する。
- [x] T002 [Tooling] Knip と依存整合性設定、読み取り専用のパッケージスクリプト、`check-ui-dependencies` mise タスクを追加し、未使用依存と未宣言依存だけを検査する。
- [x] T003 [Tooling] 固定したローカル版の shadcn CLI を使う `add-ui-component` mise タスクを追加し、`mise run add-ui-component -- button card` の形で複数コンポーネントを指定できるようにする。
- [x] T004 [Deps] `shadcn` を `devDependencies` へ移し、全直接依存を利用実態に合わせて追加、削除、再分類し、`frontend/bun.lock` を更新する。
- [x] T005 [Deps] Bun とフロントエンドの直接依存を着手時点の安定した互換版へ更新する。協調するパッケージ群を同じ単位で更新し、各単位で型検査、単体テスト、ビルドを通す。
- [x] T006 [Config] Biome、Vite、TypeScript、Tailwind CSS、shadcn、テスト環境の設定を更新後の形式へ移行し、必要な互換修正とルート再生成を行う。
- [x] T012 [Tooling] `src/test/register-dom.ts` で Bun 本来の `fetch` を退避して登録後に戻し、`src/test/setup.ts` の包み込みで相対 URL を絶対化してから委譲する。`mise run test-ui-e2e` が起動待ちのタイムアウトで失敗しないことを確認する。
- [x] T007 [Tooling] `verify-ui`、リポジトリ全体の `verify`、CI に型検査と依存整合性検査をそろえて配線し、mise、`packageManager`、Dockerfile の Bun の版一致を既存の mise 設定検査で保証する。
- [x] T008 [Docs] `frontend/README.md` から移行済みの Radix UI と Vitest の記述を直し、現在の Base UI、Bun test、Vite、shadcn の役割と mise 経由のコンポーネント追加手順を記載する。
- [x] T009 [Acceptance] `shadcn` が `devDependencies` だけに存在し、固定ロックファイルからの再インストールで差分が生じず、更新後の静的バンドルと E2E の主要経路が更新前と同じ振る舞いを保つことを確認する。
- [x] T010 [Change Resistance] 使用中の直接依存を一時的にマニフェストから外す誤実装と、未使用の直接依存を一時的に加える誤実装のうち少なくとも一つを注入し、`mise run check-ui-dependencies` が失敗することを確認してから差分を戻す。
- [x] T011 [Verify] フロントエンド、依存脆弱性、ツール検査、リポジトリ全体の検証を通し、生成物、マニフェスト、ロックファイルに意図しない差分が無いことを確認する。

## Verification

### 着手前に宣言する RED

- **Acceptance RED**：`jq -e '.dependencies.shadcn == null and .devDependencies.shadcn != null' frontend/package.json` は exit 1 になる。現在は `shadcn` が `dependencies` にだけ存在するためであり、本項目の主要な依存分類の差を直接検出する。
- **Unit RED の代替**：依存整合性検査の配線後、使用中の直接依存をマニフェストから一時的に外すか、未使用の直接依存を一時的に加える。`mise run check-ui-dependencies` が失敗しなければ検査は依存宣言の過不足を検出していないため、設定を直して失敗を観測してから実装を緑へ戻す。設定ファイルの更新には独立した純粋論理の単体境界が無いため、この故障注入を Unit RED の代替とする。

### 完了時に通すもの

- `mise run install-ui`
- `mise run check-ui-dependencies`
- `mise run verify-ui`
- `mise run test-ui-e2e`
- `mise run audit-dependencies`
- `mise run test-tools`
- `mise run verify`
- `jq -e '.dependencies.shadcn == null and .devDependencies.shadcn != null' frontend/package.json`
- `git diff --check`

## Risk Notes

- **複数の更新が同じ失敗を起こす。** すべての版を一度に変えると、型エラー、設定エラー、ブラウザー上の回帰の原因を切り分けられない。協調するパッケージ群ごとに更新し、各単位で型検査、単体テスト、ビルドを通してから次へ進む。
- **`shadcn` を開発依存へ移した結果、コンテナビルドが CSS を解決できなくなる。** 現在の多段ビルドはビルド段階で開発依存を含む `bun install --frozen-lockfile` を実行する。この順序を維持し、固定ロックファイルからのインストールと静的バンドル生成を確認する。
- **設定移行が製品の出力を変える。** Vite のプロキシ、TanStack Router のコード分割、Tailwind CSS のカスタムバリアント、Biome の自動修正は、設定が受理されるだけでは正しさを示さない。生成ルート、生成 CSS、コンポーネント単体テスト、ブラウザー E2E を使って更新前の不変条件を確認する。
- **Knip が動的な入口を未使用と誤判定する。** ルート生成、CSS import、テスト初期化、Vite プラグイン、shadcn 設定を入口として明示し、抑止はパッケージ単位で広く置かず理由と対象を限定する。依存以外の問題種別は本項目では有効にしない。
- **ロックファイル更新で推移依存の脆弱性が入る。** `mise run audit-dependencies` を更新後に実行し、未抑止の既知脆弱性を残さない。修正版が無い場合は、この保守項目の中で恒久的な抑止を追加せず、既存の期限付き抑止手順に従う。
- **Bun の版が実行環境ごとにずれる。** mise、`packageManager`、Dockerfile の三つを同時に更新し、版一致の検査を残す。CI は mise の固定版と `bun install --frozen-lockfile` を使い、手元と異なる入口を増やさない。

## Completion

- **Completed At**: 2026-08-29
- **Summary**:
  `mise run spec-diff` は `no normative specification change against main` を返す。規範仕様の差分は無く、変わったのは依存の宣言、開発ツールの設定、検証入口である。`shadcn` は `devDependencies` へ移り、未使用の `@tanstack/react-table` と、`@happy-dom/global-registrator` の推移依存でしかなかった `happy-dom` を削除し、`scripts/generate-routes.ts` が暗黙に使っていた `@tanstack/router-generator` を直接依存として宣言した。全直接依存を範囲指定せずに固定し、`@happy-dom/global-registrator` だけが 20.11.13 から 20.11.15 へ上がった。Knip を `devDependencies` に加え、`frontend/knip.jsonc` で依存関係の問題種別だけを有効にして `mise run check-ui-dependencies` を新設し、`verify-ui`、リポジトリ全体の `verify` と `verify-serial`、CI へ配線した。`verify-ui` には `typecheck-ui` も加わり、三つの入口が同じフロントエンド検査を実行する。固定版の CLI を使う `mise run add-ui-component` を追加した。`vite.config.ts` の `__dirname` は `import.meta.dirname` へ移した。`bunfig.toml` の preload が E2E の起動待ちを遮断していた問題を直し、E2E は 4 ファイル 24 件が通るようになった。
  着手時に版が食い違うと見ていた TanStack Router の実行時パッケージと Vite プラグインは、実際には食い違いではなかった。`@tanstack/react-router` 1.170.32 と `@tanstack/router-plugin` 1.168.35 はどちらも公開されている最新版であり、プラグインは `@tanstack/router-generator` 1.167.33 を厳密に固定している。TanStack が各パッケージを独立に採番しているだけなので、版をそろえる変更はしていない。Bun も 1.4.0 が最新で、`mise.toml`、`packageManager`、`Dockerfile` の三つは既に一致していた。上げる差分は無いが、次にずれたときに気づけるよう版一致の検査だけを残した。
- **Acceptance RED Evidence**:
  - **Test**: `jq -e '.dependencies.shadcn == null and .devDependencies.shadcn != null' frontend/package.json`
  - **Requirement**: N/A: 開発ツールと依存宣言の保守であり、対応する規範シナリオを持たない。
  - **Observed Failure**: 実装前に `false` を出力して exit 1。`shadcn` が `dependencies` にだけ存在したためである。実装後は `true` を出力して exit 0 になる。
  - **Detection Reason**: 二つの条件の連言なので、`shadcn` を両方から消す、片方に残したまま両方へ書く、`devDependencies` へ書き足して `dependencies` から消し忘れる、のいずれもが不合格になる。本項目の中心である「どちらの依存集合が所有するか」の差だけを見る。
- **Unit RED Evidence**:
  - **Test**: `mise run check-ui-dependencies` への故障注入。宣言どおりの単体境界が無いため、work item が宣言した代替をそのまま実施した。
  - **Requirement**: N/A: 設定ファイルの配線であり、独立した純粋論理の単体境界を持たない。
  - **Observed Failure**: 使用中の直接依存 `clsx` をマニフェストから外すと `Unlisted dependencies (1) clsx src/lib/utils.ts:1:39` を報告して exit 1。未使用の直接依存 `is-odd` を加えると `Unused dependencies (1) is-odd package.json:36:6` を報告して exit 1。差分を戻すと exit 0。
  - **Detection Reason**: 過不足の両方向を見るので、宣言を丸ごと信じる実装（未使用を見逃す）と、`node_modules` の実在だけを見る実装（推移依存への暗黙の依存を見逃す）のどちらも不合格になる。実際に後者が `@tanstack/router-generator` の未宣言を検出したので、この向きが空振りでないことは注入以外でも確かめられている。
- **Change-Resistance Results**:
  - 使用中の直接依存 `clsx` を `frontend/package.json` から削除 → `mise run check-ui-dependencies` が exit 1（未宣言として検出）。差分を戻して exit 0。
  - 未使用の直接依存 `is-odd` を `frontend/package.json` に追加 → `mise run check-ui-dependencies` が exit 1（未使用として検出）。差分を戻して exit 0。
  - `frontend/Dockerfile` のビルダーイメージを `oven/bun:1.3.14-alpine` へ書き換え → `mise run test-tools` の `Bun version boundary > builds the UI container from the same version` が失敗。差分を戻して 17 件すべて成功。
  - 検出できない変更として、`knip.jsonc` の `include` から問題種別を減らす改変が残る。検査自身の対象を狭める変更は検査自身では検出できない。`mise run test-tools` が `verify-ui` の構成と各入口への配線までは固定するが、Knip に何を検査させるかは固定していない。
- **Verification Results**:
  - `mise run install-ui` - passed（`--frozen-lockfile` で 549 パッケージに変更なし。マニフェストとロックファイルの書き換えも無し）
  - `mise run check-ui-dependencies` - passed
  - `mise run verify-ui` - passed（依存検査、整形、静的解析、型検査、単体テスト 667 件、ビルド）
  - `mise run test-ui-e2e` - passed（4 ファイル 24 件。着手時は最初の起動待ちで失敗していた）
  - `mise run audit-dependencies` - passed（既存の期限付き抑止 1 件を除いて指摘なし）
  - `mise run test-tools` - passed（`mise run verify` の中で実行。`mise-config.test.ts` は 17 件）
  - `mise run verify` - passed
  - `jq -e '.dependencies.shadcn == null and .devDependencies.shadcn != null' frontend/package.json` - passed
  - `git diff --check` - passed
