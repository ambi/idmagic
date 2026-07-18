---
status: pending
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

## Out of Scope

- SCL（`spec/scl.yaml`）の規範定義・context_map の変更（本 wi は純物理配置変更）。
- authentication / oauth2 の feature 分割（→ wi-255 / wi-256）。
- 単一 feature の薄い context（signingkeys, tenancy, audit, jobs 等）への feature 層導入。
  条件付き規約により、これらはフラット構造のまま維持する。
- `module.go`（context 単位 DI 束、ADR-091）と `backend/cmd/internal/bootstrap` の
  組み立て構造の変更。feature 層はソース配置のみの変更とし、DI は据え置く。

## Plan

変換後ツリー（パイロット idmanagement）:

```text
backend/idmanagement/
  module.go                 # context ルートに1つ維持（DI 束は据え置き）
  domain/                   # 複数 feature 横断の共有ドメイン型のみ残す（events, enums 等）
  user/
    domain/  ports/  usecases/  adapters/{http,persistence/{memory,postgres,valkey}}/
  group/
    domain/  ports/  usecases/  adapters/{http,persistence/{memory,postgres,valkey}}/
  agent/
    domain/  ports/  usecases/  adapters/{http,persistence/{memory,postgres,valkey}}/
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

## Tasks

- [ ] T001 [ADR] `decisions/ADR-124-*.md` を `new-adr` skill で作成。条件付き規約・成長トリガー
      規約・module.go 据え置き・SCL 不変を決定として記録し、ADR-089/090/091 と RA §3.8 を参照。
- [ ] T002 [Docs] `REGENERATIVE_ARCHITECTURE.md` §3.8 のコードブロック（129–141 行）を更新。
      `internal/`→`backend/` 訂正 + 任意 `<feature>/` 層と条件付き/成長トリガー規約を追記。
- [ ] T003 [Move] `backend/idmanagement/` を `git mv` で `user/`/`group/`/`agent/` 配下へ再配置
      （全層）。共有ドメイン型は context ルートの `domain/` に残す。
- [ ] T004 [Go] import path を一括置換し、同一 context 複数 feature を同時 import する箇所
      （`adapters/http/routes.go` 等の context 横断ハブ）の named import を修正。
- [ ] T005 [Docs] `ARCHITECTURE.md` を `new-architecture` skill で同期。`## Go Package Conventions`
      の散文・ツリーに feature 層と規約を追記し、frontmatter `modules[].path` の idmanagement 分を
      feature 粒度へ更新。
- [ ] T006 [Verify] 下記 Verification を実行し全緑を確認。

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
