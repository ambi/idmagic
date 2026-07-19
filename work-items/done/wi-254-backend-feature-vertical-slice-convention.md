---
status: completed
authors: [tn]
risk: medium
created_at: 2026-07-18
depends_on: []
change_kind: refactor
spec_impact:
  kind: none
  reason: "context 境界・context_map の publishes/depends_on を動かさない純粋な物理配置変更。SCL 規範振る舞いは不変で spec/scl.yaml 編集も scl-render も不要。"
initial_context:
  source: [backend/idmanagement, REGENERATIVE_ARCHITECTURE.md, ARCHITECTURE.md]
  tests: [backend/idmanagement]
  stop_before_reading: [frontend, spec]
---

# 大型 bounded context に feature 垂直スライス層を導入する規約を定め、idmanagement をパイロット変換する

## Motivation

`backend/<context>/{adapters,domain,ports,usecases}/` の現行構成は、context が
大きくなると 1 つの層ディレクトリに複数 feature のファイルが平積みされ、複雑になる。
特に大型 context（oauth2 ≈10.3k / idmanagement ≈8.3k / authentication ≈8.3k LOC）で
顕著で、`domain/` や `usecases/` に user・group・agent など無関係な sub-domain が同居する。

RA §3.8 の「ディレクトリ構造はドメイン・仕様の構造をそのまま表現し反映し叫ばなければ
ならない」という要請に照らすと、context 内が複数の sub-domain（feature）で構成される場合、
その垂直軸も物理配置に写すべきである。そこで層 × context の格子に **feature の垂直スライス層**
`backend/<context>/<feature>/{adapters,domain,ports,usecases}/` を足す規約を定め、最も
境界がクリーンな idmanagement をパイロットとして変換する。

## Scope

- **規約の確定（ADR）**: `decisions/ADR-124-*.md` を新設。「大型 bounded context 内の
  feature 垂直スライス」を決定し、下記 Plan の条件付き規約・成長トリガー規約・module.go
  据え置きを明文化する。ADR-089/090/091（context-locality 系）と RA §3.8 を参照する。
- **RA メタ文書の更新**: `REGENERATIVE_ARCHITECTURE.md` §3.8 の構造例コードブロック
  （129–141 行）を更新。`internal/`→`backend/` の訂正に加え、任意の `<feature>/` 層と
  条件付き/成長トリガー規約を追記する。
- **アーキテクチャ地図の同期**: `ARCHITECTURE.md` の `## Go Package Conventions`
  （散文・ツリー、1661–1673 行付近）に feature 層と規約を追記し、frontmatter の
  `modules[].path` の idmanagement 分を feature 粒度へ更新（`new-architecture` skill）。
- **パイロット変換**: `backend/idmanagement/` を `user/` `group/` `agent/` の feature 層へ
  再配置（`domain`/`ports`/`usecases`/`adapters/http`/`adapters/persistence/{memory,postgres,valkey}`
  の全層）。`git mv` で履歴を保持し、Go import path を一括置換する。
- **共有ドメイン型・persistence サブ構造の帰属決定**: `domain/events.go`・`enums.go` など
  複数 feature 横断の型の置き場、および postgres の `queries`/`sqlcgen` を feature 単位に
  割るか context 共有に残すかをパイロットで確定し、doc に反映する。
- **Phase 2（adapters/http・adapters/persistence/postgres の分割）**: Phase 1（`domain`/
  `ports`/`usecases`/`adapters/persistence/memory`）完了後、いったん「Go の言語制約により
  分割不可」と判断したが、レビューで指摘を受けて再検討した結果、次の設計で両方とも分割可能と
  判明したため追加実施する。
  - `adapters/http`: `Deps` 構造体は context ルート共有のまま据え置くが、ハンドラを
    `func (d Deps) handleX(c *echo.Context) error` という **メソッド**から
    `func handleX(d Deps, c *echo.Context) error` という**フリー関数**へ変換し、feature
    ディレクトリへ移す。フリー関数は receiver 型と同一パッケージである必要がないため、
    Deps 型を context ルートに残したまま実装だけを feature 側パッケージへ移せる。
    `routes.go` の登録は `g.GET(path, d.handleListGroups)` から
    `g.GET(path, grouphttp.HandleListGroups(d))` のような呼び出しへ変わるが、
    `Deps{}` を構築する外部 30 箇所以上の呼び出し規約（フラットな field 名）は無変更。
  - `adapters/persistence/postgres`: `sqlc.yaml` の該当 1 エントリを feature 単位の 3
    エントリへ分割し、`queries/*.sql` と生成される `sqlcgen/` を feature ディレクトリへ
    移す。`fixtures_test.go`/`harness_test.go` の feature 横断 fixture ヘルパー
    （`seedTenant`/`seedUser`/`seedGroup`/`seedClient`/`testClock` 等）は Go の `_test.go`
    がパッケージをまたげない制約により、各 feature パッケージへ複製する。

## Out of Scope

- SCL（`spec/scl.yaml`）の規範定義・context_map の変更（本 wi は純物理配置変更）。
- authentication / oauth2 の feature 分割（→ wi-255 / wi-256）。
- 単一 feature の薄い context（signingkeys, tenancy, audit, jobs 等）への feature 層導入。
  条件付き規約により、これらはフラット構造のまま維持する。
- `module.go`（context 単位 DI 束、ADR-091）と `backend/cmd/internal/bootstrap` の
  組み立て構造の変更。feature 層はソース配置のみの変更とし、DI は据え置く。

## Plan

変換後ツリー（パイロット idmanagement、Phase 2 込み）:

```text
backend/idmanagement/
  module.go                 # context ルートに1つ維持（DI 束は据え置き）
  domain/                   # 複数 feature 横断の共有ドメイン型のみ残す（events, enums 等）
  usecases/                 # 複数 feature 横断の共有 usecase ヘルパー・エラー変数のみ残す
  adapters/http/
    routes.go                # Deps 型定義とルート登録の集約点（context ルート共有）
    extra_identity_test.go   # feature 横断の統合テスト（context ルート共有）
  user/
    domain/  ports/  usecases/
    adapters/http/            # フリー関数ハンドラ（Deps を引数で受ける）
    adapters/persistence/{memory,postgres}/
  group/
    domain/  ports/  usecases/
    adapters/http/
    adapters/persistence/{memory,postgres}/
  agent/
    domain/  ports/  usecases/
    adapters/http/
    adapters/persistence/{memory,postgres}/
```

- **条件付き規約**: feature 層は **2 つ以上の feature を持つ context のみ**に導入する。
  単一 feature の context に `backend/signingkeys/signingkeys/` のような context名=feature名の
  stutter を作ることは「何も叫んでいない」ため RA 的に有害として禁止する。
- **成長トリガー規約**: context が 2 つ目の feature を獲得した時点で feature 層を導入する。
  これを doc に明文化し、「導入しそびれ」を防ぐ。構造は将来の仮定ではなく現在のドメインを映す。
- **package 名は各層名のまま**: feature 配下でも `package domain`/`ports`/`usecases`/`http`。
  Go は import パスで区別するため、同一 context の複数 feature の `domain` を同時 import する
  箇所（特に context 横断ハブ `adapters/http/routes.go`）では named import が必要になる
  （例 `userdomain`, `groupdomain`）。既存コードも `idmports` 等の named import を多用しており
  慣習の延長。
- **共有ドメイン型**: feature 横断の型（events, enums, 属性スキーマ等）は context ルートの
  共有 `domain/` に残し、feature 固有型のみ feature 配下へ移す。パイロットで具体的な帰属を
  確定して doc 化する。
- **却下した代替案**:
  - 全 context 一律に feature 層を導入 → 単一 feature context で stutter が発生し RA 原則に反する。
  - `module.go` を feature ごとに分割 → bootstrap の組み立て変更が広範になり、feature 層の
    目的（ソース配置の可読性）に対して費用対効果が低い。
  - `adapters/http` の `Deps` を feature ごとの embedded 部分構造体
    （`type Deps struct { user.UserDeps; group.GroupDeps; agent.AgentDeps }`）へ再設計する案
    → 技術的には可能だが、admin_group_handler.go が `UserRepo`、admin_agent_handler.go が
    `UserRepo`/`ClientRepo`、admin_user_handler.go が `GroupRepo`/`ScimRepo`/`JobRepo` 等
    他 feature の port を横断的に参照しており、embed 化すると各 feature の部分構造体に
    同じ port フィールドを重複定義する必要が生じる。フリー関数化（採用案）は Deps 型を
    そのまま維持しつつハンドラの実装コードだけ移せるため、この重複を避けられる。

## Tasks

- [x] T001 [ADR] `decisions/ADR-130-idmanagement-feature-vertical-slice.md` を `new-adr` skill で
      作成(採番は ADR-130。wi 起票時点で ADR-124 は既に別件
      `ADR-124-scheduled-batch-execution-boundary.md` に採番済みだったため繰り上げ)。条件付き規約・
      成長トリガー規約・module.go 据え置き・SCL 不変を決定として記録し、ADR-089/090/091 と RA §3.8 を
      参照。パイロット変換で判明した2つの分割除外(adapters/http, adapters/persistence/postgres)も
      決定として明記。
- [x] T002 [Docs] `REGENERATIVE_ARCHITECTURE.md` §3.8 のコードブロックを更新。`internal/`→`backend/`
      訂正 + 任意 `<feature>/` 層と条件付き/成長トリガー規約を追記。
- [x] T003 [Move] `backend/idmanagement/` を `git mv` で `user/`/`group/`/`agent/` 配下へ再配置。
      対象は `domain`/`ports`/`usecases`/`adapters/persistence/memory` の4層。共有ドメイン型
      (enums.go/events.go)は context ルートの `domain/` に残した。`adapters/http` と
      `adapters/persistence/postgres` は Go の言語制約(Deps 構造体のハンドラメソッドは receiver 型と
      同一パッケージが要る)と sqlc 単一生成 + feature 横断テスト fixture の制約により分割せず
      context ルート共有のまま維持(ADR-130 で決定として明記)。
- [x] T004 [Go] import path を一括置換し、同一 context 複数 feature を同時 import する箇所
      (`adapters/http/*.go`, `module.go` 等の context 横断ハブ)の named import
      (`userdomain`/`groupdomain`/`agentdomain`/`userports`/…)を修正。usecases 層の
      feature 非依存ヘルパー(`normalizeRoles` 等)と feature 横断エラー変数
      (`ErrUserNotFound`/`ErrInvalidRole`)は新設 `backend/idmanagement/usecases/helpers.go`
      (共有、package usecases)へ集約。
- [x] T005 [Docs] `ARCHITECTURE.md` を `new-architecture` skill で同期。`## Go Package Conventions`
      に feature 垂直スライスの節を追記し、frontmatter `modules[].path` の idmanagement 分を
      `idmanagement-{user,group,agent}-{domain,ports,usecases}` と
      `idmanagement-{user,group,agent}-adapters`(memory persistence)へ feature 粒度分割。
      約 30 の外部 module の `depends_on` も実際の import 先 feature へ更新。
- [x] T006 [Verify] 下記 Verification を実行し全緑を確認（Phase 1 完了時点）。
- [x] T007 [Go] `adapters/http` を feature 分割する（Phase 2）。`Deps` 型定義自体は
      `backend/idmanagement/adapters/http/httpdeps`（新設 leaf package）へ切り出し、
      context ルートの `adapters/http` は `type Deps = httpdeps.Deps`（型 alias）で
      再エクスポートして外部の `idmhttp.Deps{...}` 構築コードを無変更に保った。
      ハンドラメソッド `func (d Deps) handleX(c)` はフリー関数
      `func HandleX(d Deps, c) error` へ変換し、`user/adapters/http`・`group/adapters/http`・
      `agent/adapters/http` へ移動。`routes.go` の登録を `d.handleX` から
      `func(c *echo.Context) error { return featurehttp.HandleX(d, c) }` のクロージャへ更新。
      `RegisterRoutes`・`extra_identity_test.go` は context ルート共有のまま。
      feature 横断のテストヘルパー（`adminCSRF`/`adminJSONRequest`/`mockEmailSender`）は
      Go の `_test.go` パッケージ跨ぎ制約により root と group パッケージへ複製。
- [x] T008 [Go] `adapters/persistence/postgres` を feature 分割する（Phase 2）。`sqlc.yaml` の
      idmanagement 用エントリを feature 単位の 3 エントリ（+ 既存の lifecycle_workflows 用
      context ルートエントリ）へ分割し、`sqlc generate` で `queries/*.sql` と `sqlcgen/` を
      feature ディレクトリへ再生成。`fixtures_test.go`/`harness_test.go`（feature 横断
      fixture ヘルパー）と `textOrNil`（pgtype 変換ヘルパー）は各 feature パッケージへ複製。
      `lifecycle_workflows` テーブルは IdGovernance context 所有（wi-237/ADR-117 以前の
      歴史的経緯）で user/group/agent いずれにも属さないため、context ルートの
      `adapters/persistence/postgres/` に残した。`backend/cmd/internal/bootstrap/
      postgres_valkey.go`・`backend/idgovernance/adapters/persistence/postgres/
      lifecycle_workflow_capture.go`・`backend/shared/adapters/persistence/postgres/
      pgfixtures/pgfixtures.go` 等の外部参照を feature 別 import path へ更新。
- [x] T009 [Docs] `ARCHITECTURE.md` を Phase 2 の物理配置に合わせて再同期。`Deps` 型定義を
      leaf package `idmanagement-httpdeps` として独立 module 化（`idmanagement-adapters` と
      feature 別 `-adapters` module の双方が依存するため、module 依存グラフの循環を避ける
      設計）。`idmanagement-{user,group,agent}-adapters` の path を feature の
      `adapters/` サブツリー全体（http + persistence/memory + persistence/postgres）へ
      拡張。`## Go Package Conventions` の Feature 垂直スライス節を Phase 2 の実態
      （フリー関数化・httpdeps 分離・sqlc feature 分割・lifecycle_workflows の扱い）に
      合わせて全面更新。
- [x] T010 [Verify] 下記 Verification を Phase 2 込みで再実行し全緑を確認。

## Verification

- `just verify-go` — format-check / lint / typecheck / build が緑。
- `just build-go` — 全パッケージビルドで新 import path 解決を確認。
- `just test-go` — テスト緑（idmanagement の全テストを含む）。
- `just yaml-check` / `just check-ids` — RA/SCL の ID・YAML 整合（SCL 不変なので影響なしを確認）。
- `just verify` — 全体スイートの最終確認。
- `git log --follow backend/idmanagement/user/domain/users.go` 等で `git mv` の履歴保持を確認。
- 旧配置への import 残存がゼロであることを grep で確認。

## Risk Notes

- **広範だが機械的**: 再配置と import 置換はファイル数が多いが、module import prefix が一意で
  `just build-go` / `just test-go` が網羅的に検証する。named import 修正のみ手作業判断が要る。
- **共有型の帰属**が設計判断を要する唯一の非機械的部分。feature 横断型を安易に feature 配下へ
  移すと不要な cross-feature import を生むため、context ルート共有 `domain/` に残す方針で軽減。
- **module.go / bootstrap 据え置き**により DI 面の破壊的変更を回避し、リスクを配置変更に限定する。
- **並行ブランチとの衝突**: パス移動は衝突しやすいので、並行 work-item ブランチが少ない
  タイミングで実施する。

## Completion
- **Completed At**: 2026-07-20
- **Summary**:
  feature 垂直スライスの規約を ADR-130 として決定し、idmanagement を `user`/`group`/`agent`
  の 3 feature へ**全層**（`domain`/`ports`/`usecases`/`adapters/http`/
  `adapters/persistence/{memory,postgres}`）変換した。
  - **Phase 1**: `domain`/`ports`/`usecases`/`adapters/persistence/memory` を `git mv` で
    再配置し、import path を一括置換。共有型（`enums.go`/`events.go`）は context ルートの
    `domain/` に、feature 非依存ヘルパーと feature 横断エラー変数は新設
    `backend/idmanagement/usecases/helpers.go` に残した。
  - **Phase 2**（レビューでの指摘を受けて追加実施）: 当初 Go の言語制約・コード生成単位を
    理由に分割対象外としていた `adapters/http` と `adapters/persistence/postgres` も、
    再検討の結果別の設計で分割できると判明し実施した。
    - `adapters/http`: `Deps` 構造体を独立した leaf package `httpdeps` へ切り出し、
      ハンドラをメソッドからフリー関数（`func handleX(d Deps, c) error`）へ変換して
      feature パッケージへ移した。`Deps` 型自体は分割せず、外部の `idmhttp.Deps{...}`
      構築コード（bootstrap・約30のテストファイル）を無変更に保った。
    - `adapters/persistence/postgres`: `sqlc.yaml` を feature 単位の複数エントリへ分割し
      `sqlc generate` で再生成。feature 横断のテスト fixture ヘルパーと `textOrNil` は
      各 feature パッケージへ複製した。`lifecycle_workflows` テーブルは IdGovernance
      context 所有（wi-237/ADR-117 以前の歴史的経緯）で features のいずれにも属さないため、
      context ルート共有のまま維持した。
  - ADR-130・`REGENERATIVE_ARCHITECTURE.md` §3.8・`ARCHITECTURE.md`
    （module ledger + Go Package Conventions 節）を Phase 2 の実態に合わせて更新した。
- **Phase 3**（ユーザー指摘を受けて追加実施）: Phase 2 完了報告で「`lifecycle_workflows` は
  IdGovernance context 所有の歴史的経緯により idmanagement の postgres schema に残した」と
  開示したところ、根本是正（IdGovernance への物理移設）を求められ実施した。ADR-117 が
  「context-local な sqlc パッケージ分割（ADR-090）は後続 WI へ後回しする」と明記していた
  既知の debt そのものだったため、新たな設計判断は不要だった。`lifecycle_workflows.sql` を
  `backend/idgovernance/adapters/persistence/postgres/queries/` へ `git mv` し、`sqlc.yaml`
  の該当エントリを idmanagement から idgovernance へ付け替えて `sqlc generate` で再生成。
  `lifecycle_workflow_runs.go` の sqlcgen import path を更新し、
  `backend/idmanagement/adapters/persistence/` はこれで完全に消滅した（残るのは
  `backend/idmanagement/adapters/http/` の Deps・route 登録・feature 横断統合テストのみ）。
  ADR-117 に一文注記を追加し、`ARCHITECTURE.md` の該当 module（`idmanagement-adapters`・
  `idgovernance-adapters`）と Go Package Conventions 節を更新した。
- **Out of Scope として明示的に対応していないこと**（ADR-121 開示）:
  - authentication / oauth2 の feature 分割は当初から Out of Scope（→ wi-255 / wi-256）。
  - signingkeys, tenancy, audit, jobs 等の単一 feature context への feature 層導入は
    条件付き規約により対象外（規約上、意図的に導入しない）。
- **Verification Results**（Phase 1 + Phase 2 + Phase 3 込みで再実行、すべて green）:
  - `just build-go` — passed。
  - `just test-go` — passed（idmanagement の全 feature サブパッケージ、外部消費者を含む）。
  - `just verify-go`（lint-go + test-go-race） — passed。
  - `just yaml-check`（SCL / work-item / ids / architecture cross-check / traceability） — passed。
  - `just verify`（Go + UI 全体スイート） — passed。
  - `git log --follow` で `git mv` の履歴保持を確認（一部ファイルは import 書き換えによる
    差分率が高く rename ではなく create+delete として記録されたが、内容は同一ロジックの
    移動）。
  - 旧配置（`backend/idmanagement/{domain,ports,usecases,adapters/http,adapters/persistence/
    postgres}` の feature 固有シンボル）への import 残存はゼロ（grep で確認）。共有パッケージ
    （`backend/idmanagement/domain`・`backend/idmanagement/usecases`・
    `backend/idmanagement/adapters/http`）への参照は意図的に残存。
    `backend/idmanagement/adapters/persistence/` は Phase 3 で完全に消滅した
    （lifecycle_workflows も含め、idmanagement 配下に postgres 永続化コードは残らない）。
