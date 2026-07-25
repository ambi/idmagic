---
status: pending
authors: [tn]
risk: medium
created_at: 2026-07-25
depends_on: [wi-276-evaluate-vitest-to-bun-test-migration]
change_kind: tooling
spec_impact: { kind: none, reason: "フロントエンドのテストランナーと検証基盤の置換のみで、製品の外部契約・振る舞い・保証は変更しないため。" }
---

# フロントエンド単体・コンポーネントテストを Vitest から Bun test へ全面移行する

## Motivation
wi-276 の実証で、`frontend/` の 77 test files / 427 tests を Bun test + Happy DOM へ機械変換した結果、
小さな preload シムのみ（テスト無改変）で **425/427 が pass**、同一 green セットで **Bun は Vitest の約 3.6 倍速**
（75 files / 420 tests: Bun 4.0s vs Vitest 14.7s）であることが確認された。判断は Go。本 Work Item は
その全面移行を、rollback 容易な段階的計画で実施し、Vitest / jsdom を検証経路から除去する。

wi-276 が特定した要点:
- 実使用の `vi` API は 6 種（`fn`/`stubGlobal`/`unstubAllGlobals`/`mocked`/`spyOn`/`restoreAllMocks`）のみ。
  **`vi.mock`（モジュールモック）は不使用**で、機械変換が現実的。
- **決定的真因**: Happy DOM の `Location` はアクセサを prototype 側に持ちスプレッド不可。
  `vi.stubGlobal('location', {...originalLocation, assign: vi.fn()})` が `pathname`/`origin` を失い、
  アプリの try/catch に握り潰されてサイレント失敗する。→ preload でスプレッド可能な location snapshot を
  注入して解消（jsdom 既定 origin `http://localhost:3000` に一致させる）。
- RTL 自動 cleanup は Bun に global `afterEach` が無いため登録されない。→ preload で `afterEach(cleanup)` を明示。
- Happy DOM 登録は `@happy-dom/global-registrator`（DOM 登録）と jest-dom/RTL cleanup を **2 段 preload に分離**
  （ESM の import 巻き上げ回避）。

## Scope
- `frontend/vite.config.ts`: `test` ブロック（Vitest 設定）除去、または Bun 用設定への置換。
- `frontend/package.json`: `test:unit` / `test:unit:coverage` スクリプトの Bun test 化、依存の入替
  （remove: vitest / jsdom / @vitest/coverage-istanbul、add: happy-dom / @happy-dom/global-registrator）。
- `frontend/bunfig.toml`（新規）: preload 登録。
- `frontend/src/test/` 配下: 2 段 preload（`register-dom` / `setup`）と、`vi.stubGlobal`/`unstubAllGlobals` 相当の
  bun ネイティブな小ヘルパー（例 `src/test/globals.ts` の `stubGlobal`/`restoreGlobals`）。**`vitest` 命名の shim は残さない。**
- `justfile`: `test-ui-unit` / `test-ui-cover` の実体を Bun test へ。
- `frontend/tsconfig*.json`: `vitest/globals` 型参照の除去、`bun-types` 追加。
- `wi-131` 由来の coverage governance（path group / diff threshold / CI artifact）の再配線。
- CI 定義（macOS / Linux）でのランナー切替。
- SCL 影響なし（`spec/scl.yaml` は触れない）。

## Out of Scope
- 製品コード（`src/` の非 test）・SCL・API・UI 挙動の変更。
- UI E2E（`tests/e2e/`、既に `bun test`）の構成変更。
- テストの assertion（オラクル）を弱める形の書き換え。互換性差は setup / preload 側で吸収する。

## Plan
- **移行方式**: wi-276 の spike 手法を本番化する。ただし **`vitest` という歴史的名称を残さず bun test ネイティブな形へ
  リファクタする**。各 test の `from 'vitest'` を `from 'bun:test'` に置換し、`vi.fn`→`mock`、`vi.spyOn`→`spyOn`、
  `vi.restoreAllMocks`→`jest.restoreAllMocks`（または `mock.restore`）、`vi.mocked(x)`→そのまま `x`（型ヘルパー不要）へ
  機械置換する。`bun:test` に相当が無い `vi.stubGlobal`/`unstubAllGlobals`（計 185 箇所）は、`vitest` を想起させない
  用途名のヘルパー（例 `src/test/globals.ts` の `stubGlobal`/`restoreGlobals`）へ寄せる。互換シムをそのまま常設しない。
- **DOM**: Happy DOM を 2 段 preload で登録。location スプレッド互換 snapshot を preload に恒久化。
- **残 2 tests の手当て**（wi-276 で切り分け済み・オラクル不変で対応）:
  - `AdminLifecycleWorkflowPages`（削除）・`ApiTokensTab`（発行/取消）は「`await` 後の `setState` が
    `waitFor` ポーリング中に flush されない」React 19 + Happy DOM のタイミング差。`findBy*` への置換や
    `act` 境界の付与など、assertion を変えない範囲で解消する。
- **カバレッジ**: 全 src 強制 import preload で分母を復元。Bun の exclude が貧弱（preload/helper 混入）なため
  reporter 出力を後処理するか、対象を絞る。wi-131 の JSON 依存は LCOV へ移す。分母が現行 Istanbul より
  狭くなる構成は採らない（未 import file を 0% で算入する）。
- **段階と rollback**: 1 コミットで Vitest を残したまま Bun 経路を追加 → 全 green 確認 → Vitest 除去、の順で
  各段階を独立 revert 可能にする。
- **未決定事項**: `vi.stubGlobal` の置換先ヘルパーの粒度・命名、coverage reporter 後処理の要否、CI マトリクスの範囲。

## Tasks
- [ ] T001 [Helper] `vi.stubGlobal`/`unstubAllGlobals` の置換先を bun ネイティブな用途名ヘルパー
      （例 `src/test/globals.ts`）として用意し、`vitest` 命名の shim を常設しない方針を確定する。
- [ ] T002 [Preload] `register-dom`（Happy DOM + location snapshot + `IS_REACT_ACT_ENVIRONMENT`）と `setup`
      （jest-dom matchers + `afterEach(cleanup)`）を `src/test/` に用意し `bunfig.toml` で登録する。
- [ ] T003 [Convert] 77 test files の `from 'vitest'` を `bun:test` へ、`vi.*` を bun 相当（`mock`/`spyOn`/`jest.restoreAllMocks`/
      ヘルパー）へ機械置換し、リポジトリから `vitest` import を根絶する。
- [ ] T004 [Residual] 残 2 tests（lifecycle 削除 / ApiTokens 発行取消）をオラクル不変で green にする。
- [ ] T005 [Coverage] 強制 import で分母復元、reporter を LCOV 化、wi-131 の path group / diff threshold /
      CI artifact を再配線し、Istanbul と対象 file 集合・指標を突き合わせる。
- [ ] T006 [Wire] `package.json` / `vite.config.ts` / `justfile` / `tsconfig` を Bun test へ切替、Vitest / jsdom /
      coverage-istanbul を除去する。
- [ ] T007 [CI] macOS / Linux CI でランナーを切替え、両環境で全 green・coverage artifact 生成を確認する。
- [ ] T008 [Verify] `just verify-ui`、`just test-ui-cover`、`just check-work-items`、`just check-ids` を成功させる。

## Verification
- `just test-ui-unit`（Bun test、427 tests green）
- `just test-ui-cover`（分母が現行 Istanbul 相当、LCOV 生成）
- `just verify-ui`
- Linux CI での再現（全 green・artifact）
- 同一 test set の warm wall time が Vitest 比で有意に短いこと（wi-276: 約 3.6x）
- `just check-work-items` / `just check-ids`

## Risk Notes
- 最大のリスクは、残 2 tests のような非同期 flush タイミング差が他にも潜在し、移行時に別の test で顕在化すること。
  wi-276 の全 77 files 実行で 425/427 まで確認済みだが、修正過程で assertion を弱めない規律を保つ。
- カバレッジは分母が現行 Istanbul より狭くならないことを必須条件とする。強制 import は import 時副作用を誘発し得る
  ため、side-effect を持つ entrypoint（`main.tsx` 等）を exclude する。
- jsdom は bun 下で清潔に登録できず Happy DOM 前提となるため、jsdom 特有挙動に依存する test が将来追加された
  場合の drift 検出方針（例: Happy DOM で落ちる形の CI ゲート）を用意する。
- 段階コミットで各ステップを rollback 可能にし、Vitest 除去は全 green 確認後に限定する。
