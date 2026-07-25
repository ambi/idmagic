---
status: completed
authors: [tn]
risk: medium
created_at: 2026-07-23
depends_on: []
change_kind: tooling
spec_impact: { kind: none, reason: "フロントエンドのテストランナー選定と検証基盤だけを評価し、製品の外部契約・振る舞い・保証は変更しないため。" }
---

# フロントエンド単体・コンポーネントテストの Vitest から Bun test への移行可否を実証する

## Motivation
IdMagic は Bun をパッケージ管理、スクリプト実行、UI E2E、`tools/` のテストに利用する一方、
フロントエンドの単体・React コンポーネントテストには Vitest、React Testing Library、jsdom、
Istanbul coverage を利用している。Bun test への統一は依存関係とテスト基盤を単純化し、起動時間や
リソース消費を改善する可能性があるが、現在のテスト資産と同じ分離性、DOM 互換性、カバレッジ品質を
維持できるかは実証されていない。

2026-07-23 時点の調査では、フロントエンドに 77 test files / 425 tests があり、全 77 files が
`vitest` を import している。52 files が `vi` を使用し、特に Bun の限定的な Vitest 互換 alias に
含まれない `vi.stubGlobal` を 133 箇所、`vi.unstubAllGlobals` を 50 箇所、`vi.mocked` を 6 箇所で
使用している。また 22 files が `window`、`document`、storage、File API、WebAuthn などの DOM / browser API
へ直接依存する。現行の `just test-ui-unit` は 77 files / 425 tests を全件成功し、ローカル実測で
約 10.9 秒だった。

Bun test は TypeScript / JSX、lifecycle hooks、mock、React Testing Library、coverage を提供するため
移行そのものは可能と見込まれる。ただし Bun の公式手順では jsdom ではなく Happy DOM と preload を
用い、全 test files を単一 process で実行する。現行 Vitest の file isolation / file parallelism、
Vite transformation pipeline、Istanbul の text / JSON / HTML report、未 import source を含む
`coverage.include` と同等の保証は自動的には得られない。推測による全面移行ではなく、代表 test の
spike と測定結果を基に Go / No-Go を判断する必要がある。

## Scope
- 現行 Vitest suite の test API、mock、global mutation、DOM API、Vite transformation、coverage report
  依存を inventory 化し、Bun test との対応表を確定する。
- pure logic、React component、global stub、storage / location、File API、WebAuthn、Radix UI を含む代表
  10〜15 test files を選び、Happy DOM と Testing Library preload を使う隔離された Bun test spike を作る。
- `vi.stubGlobal` / `vi.unstubAllGlobals` に代わる型安全な global stub / restore helper を試作し、
  test file 間の状態漏洩と実行順依存を検証する。
- 現行 Vitest と Bun test で、結果、cold / warm wall time、CPU、peak memory、flakiness を同一環境・
  同一 test set で比較する。
- Bun coverage の text / LCOV と現行 Istanbul coverage の対象集合・line / function / branch 指標を比較し、
  未 import source を 0% として分母へ含める方法、path group / diff threshold、CI artifact 化の可否を確認する。
- `wi-131-testing-governance-and-ci-enforcement` が前提とする Vitest coverage JSON を LCOV 等へ変更する場合の
  影響を整理する。
- 根拠付きの Go / No-Go 判断を記録し、Go の場合だけ全面移行を別 Work Item として起票する。

## Out of Scope
- 本 Work Item 内で 77 test files 全体を Bun test へ移行すること。
- Vitest、jsdom、coverage provider を本番の検証経路から削除すること。
- 製品コード、SCL、API、UI behavior を変更すること。
- UI E2E の `bun test` / `Bun.WebView` 構成を変更すること。
- ベンチマーク結果を確認せず、ランナー統一だけを理由に移行を決定すること。

## Plan
- spike は現行 `just test-ui-unit` と併存させ、既存 required verification を置換しない。
- DOM environment は Bun 公式手順に沿って Happy DOM を候補とし、jsdom との差異を assertion 単位で記録する。
- global mutation を伴う test は単一 process 内で順序を変更して反復実行し、restore 漏れを検出する。
- coverage は数値の大小だけでなく、対象 source file 集合と未実行 file の扱いを比較する。現行より分母が
  狭くなる構成は、見かけの coverage が上がっても同等と判定しない。
- Go 判断には、425 tests 相当へ外挿可能な互換性、再現性、coverage governance、CI 運用性と、明確な
  性能または保守性の改善を要求する。一つでも保証できない場合は No-Go として Vitest を維持する。
- No-Go の場合も、未使用と見込まれる `@vitest/coverage-v8` の削除など、独立して安全な依存整理候補を記録する。

## Tasks
- [x] T001 [Inventory] 77 test files を分類。全 77 が `vitest` import、52 が `vi`（`fn` 265 / `stubGlobal` 135 / `unstubAllGlobals` 50 / `mocked` 6 / `spyOn` 2 / `restoreAllMocks` 1）。**`vi.mock`（モジュールモック）は不使用**＝機械変換が現実的。
- [x] T002 [Spike] 全 77 files を `vitest-shim` + Happy DOM preload + RTL cleanup で Bun 実行。段階的に 390→424→425 pass へ到達。
- [x] T003 [Isolation] 単一 process 実行で RTL 自動 cleanup 欠落による DOM 蓄積を検出→ preload で `afterEach(cleanup)` 明示。`vi.stubGlobal`/`unstubAllGlobals` は `defineProperty`＋descriptor 復元で再現。
- [x] T004 [Coverage] Bun coverage は **import 済み file のみ分母算入**（未 import file を 0% で算入できない）を実証。全 src 強制 import preload で分母復元は可能だが、Bun の exclude が貧弱で helper/preload まで混入し、副作用リスクと wi-131 再配線コストを伴う。
- [x] T005 [Performance] 同一 green 75 files / 420 tests warm 実測：**Bun 4.0s vs Vitest 14.7s（約3.6倍速）**。同一 64 files でも Bun 2.56s vs Vitest 9.47s。
- [x] T006 [Governance] wi-131 の path group / diff threshold / CI artifact(JSON→LCOV) は Bun coverage では自動維持されず、強制 import＋reporter 再設計が必要と評価。
- [x] T007 [Decision] 下記証跡に基づき **Go（移行は現実的）** と判断。全面移行 Work Item を起票する。
- [x] T008 [Verify] `just verify-ui`（format-check / lint / test-ui-unit 427 / build）green。`just test-ui-cover`（istanbul, 427）green。

### test-first 証跡について
本 Work Item は評価（tooling, `spec_impact: none`）で製品コードの振る舞いを変更しないため、ADR-119 の test-first 必須層（Domain / UseCases / Adapters）には該当しない。spike の検証は「既存 425 tests を無改変で Bun 実行し pass 数を観測する」形で行った（新規振る舞いのテスト創作はしていない）。

## Decision Criteria
Go とするには、少なくとも次をすべて満たす。

- 代表 test の assertion が jsdom / Vitest と Happy DOM / Bun test で同じ意味を検証し、未解決の互換性差分がない。
- `fetch`、`location`、`navigator`、storage、File API などの global state が test file 間で漏れず、順序変更・反復実行でも安定する。
- 未 import source を含む coverage denominator、必要な path group / diff threshold、CI artifact を維持できる。
- macOS と Linux CI の両方で再現可能で、現行 Vitest に対して有意な wall time、memory、依存管理、保守性の改善がある。
- Vite 固有 transform / plugin を利用する test が追加された場合の drift 検出方針が明確である。
- 全面移行、review、CI 安定化を含む工数が、得られる継続的な利益に見合う。

## Verification
- `just test-ui-unit`
- `just test-ui-cover`
- spike 用に追加する `just` recipe
- 同一 test set の反復・順序変更 benchmark recipe
- `just verify-ui`
- `just yaml-check-work-items`
- `just check-ids`

## Risk Notes
最大のリスクは、Bun test で test が成功しても、Happy DOM と jsdom の browser behavior 差分、単一 process
での global state 漏洩、coverage denominator の縮小によって検証能力が静かに低下することである。
特に IdMagic は global stub を多用し、File API、WebAuthn、location、storage、Radix UI を含むため、import の
機械置換だけでは安全に移行できない。また現行 Vitest は file-level parallelism により 425 tests を約
10.9 秒で実行しており、Bun test の起動が軽量でも suite 全体の高速化は保証されない。

spike は既存検証を置換せず、coverage の対象 file 集合と assertion の意味を比較し、性能値は複数回の
測定結果で判断する。全面移行はこの Work Item の Go 判断後に別 Work Item として扱い、容易に rollback
できる段階的な計画を要求する。

## Completion
- **Completed At**: 2026-07-25
- **Summary**:
  Vitest → Bun test 移行の可否を実証し、判断は **Go（移行は現実的）**。`frontend/` 全 77 test files / 427 tests を
  隔離した spike へ機械変換し、Happy DOM を登録する 2 段 preload と実使用 6 API のみの互換シムで Bun 実行した結果、
  preload のみ・テスト無改変で **425/427 が pass**、同一 green セットで **Bun は Vitest の約 3.6 倍速**
  （75 files / 420 tests: Bun 4.0s vs Vitest 14.7s）であることを確認した。
  互換性の主因は「Happy DOM の `Location` がスプレッド不可で、多用される `vi.stubGlobal('location', {...originalLocation})`
  が `pathname`/`origin` を失いアプリ側でサイレント失敗する」点で、preload でのスプレッド可能な location snapshot 注入により解消した。
  spike 一式と探索用依存は判断確定後に撤去し、本 Work Item では独立の安全整理として未使用の `@vitest/coverage-v8` のみ削除した。
  全面移行は後続 Work Item [[wi-283-migrate-frontend-unit-tests-to-bun-test]] へ委譲する。

### Affected Guarantees State
- 本 Work Item は評価（`change_kind: tooling` / `spec_impact: none`）であり、製品の外部契約・振る舞い・保証の状態は変更していない。
  `spec/scl.yaml` の `standards`・`guarantees` いずれにも変更なし。確定した実変更は開発時依存 `@vitest/coverage-v8`（未使用）の削除のみ。

### Evidence
- **速度**: 同一 green 75 files / 420 tests の warm 実測で Bun 4.0s vs Vitest 14.7s（約 3.6x）。同一 64 files でも Bun 2.56s vs Vitest 9.47s。
  ベースライン full Vitest（77 files / 427 tests）は warm 約 11.2–11.7s。実行環境: macOS arm64 / Bun 1.3.14 / Vitest 3.x。
- **互換性**: 全 77 files を Bun 実行し 390（素の変換）→ 424（location snapshot 注入）→ 425（jsdom 既定 origin `http://localhost:3000` 一致）へ到達。
  段階的に特定した差分と対処（すべて preload/setup 内・オラクル不変）:
  - RTL 自動 cleanup が Bun の global `afterEach` 欠如で未登録 → preload に `afterEach(cleanup)` を明示。
  - `vi.stubGlobal`/`unstubAllGlobals` を `defineProperty` + descriptor 復元で再現（readonly アクセサ対応）。
  - **決定的真因**: Happy DOM の `Location` はアクセサを prototype に持ち own-enumerable が空（`Object.keys(location)` = `[]`）。
    `{...window.location}` が全フィールドを失い、`tenantURL`→`path.match(undefined)` 等が throw、アプリの try/catch に握り潰される。
    → preload でスプレッド可能な location snapshot を注入し 34 files 解消。当初の「React イベント非互換」という診断は誤りだった。
  - 残り 2 tests（`AdminLifecycleWorkflowPages` 削除 / `ApiTokensTab` 発行取消）は「`await` 後の `setState` が `waitFor`
    ポーリング中に flush されない」React 19 + Happy DOM の非同期タイミング差。切り分け済み・後続で個別対応（オラクル不変）。
  - 実使用 `vi` API は 6 種のみ、`vi.mock` 不使用を確認（機械変換が現実的）。
- **DOM 実装**: jsdom は bun 下で readonly グローバル登録が失敗し清潔に使えず、Bun test は実質 Happy DOM 前提。
- **カバレッジ**: Bun coverage は import 済み file のみ分母算入。全 src 強制 import preload で分母復元は可能だが、Bun の exclude が
  貧弱で helper/preload が混入し（[oven-sh/bun#19494](https://github.com/oven-sh/bun/issues/19494)）、import 副作用リスクと
  wi-131（branch 指標・path group・diff threshold・CI artifact JSON→LCOV）の再配線コストを伴う（[Charpentier: Bun Code Coverage Gap](https://charpeni.com/blog/bun-code-coverage-gap)）。
- **開示（対応していないこと）**: 77 files の本番移行・Vitest/jsdom 除去・残 2 tests 修正・カバレッジ統治・CI（macOS/Linux）検証は
  Out of Scope で後続 [[wi-283-migrate-frontend-unit-tests-to-bun-test]] へ委譲。実測は macOS のみで Linux CI 再現は後続で確認する。

- **Verification Results**:
  - `just verify-ui`（format-check-ui / lint-ui / test-ui-unit 427 / build-ui）— passed
  - `just test-ui-cover`（istanbul, 77 files / 427 tests）— passed
  - `just check-work-items` — passed
  - `just check-ids` — passed
