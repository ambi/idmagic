---
status: completed
authors: [tn]
risk: high
created_at: 2026-07-20
depends_on: []
change_kind: refactor
spec_impact:
  kind: none
  reason: "Go package の物理配置と import path だけを変更し、SCL の振る舞い、HTTP 契約、データモデル、認可、運用保証は変更しない。"
---

# Go backend を Flat Wikipedia Architecture へ再配置する

## Motivation

context-owned adapter と feature vertical slice により所有境界は明確になった一方、
`adapters/persistence/postgres` のような中継ぎ階層が全 feature に反復し、一覧性と検索性を
損なっている。Adapter の役割と技術詳細を package 名だけで識別できる構造へ統一し、
依存性逆転を保ったまま backend の物理構造を浅くする。

## Scope

- `backend/` の context / feature 配下にある Core と Adapter package の再配置。
- Adapter package の `<role>_<technology>` 命名、Go import alias、composition root の同期。
- `sqlc.yaml`、current-state 文書、検証 manifest、現役設定の path 同期。
- `ARCHITECTURE.md`、`REGENERATIVE_ARCHITECTURE.md` のディレクトリ構成例、配置判断 ADR の同期。

## Out of Scope

- SCL、HTTP / OAuth / OIDC / SAML / SCIM wire contract、DB schema、環境変数の変更。
- `frontend/`、embedded `tools/`、`infra/` の内部アーキテクチャ変更。
- 完了済み Work Item と過去 ADR 本文に残る歴史的 path の一括書き換え。
- 旧 import path を維持する compatibility shim。

## Plan

- Core package は feature 直下の `domain` / `ports` / `usecases` を維持する。
- `adapters` / `persistence` 中継ぎ階層を除去し、`handlers_http`、`db_postgres` 等を feature
  直下へ移す。shared technical context は capability feature を切り、技術別 Adapter を分離する。
- package 衝突は利用側の lowerCamelCase named import で解決する。
- context-local PostgreSQL query と `sqlcgen` は `db_postgres` 内に維持し、`sqlc.yaml` を同期する。
- repository 全体の挙動テストを回帰 oracle とし、構造変更後に Architecture dependency graph と
  production import を再検証する。

## Tasks

- [x] T001 [Decision] ADR-133 に Flat Wikipedia Architecture の採用理由と置換関係を記録する。
- [x] T002 [Structure] context / feature local Adapter を flat package へ移動し、package 宣言と import alias を更新する。
- [x] T003 [Shared] shared technical capabilities を feature 化し、技術別 Adapter と transport-neutral port を分離する。
- [x] T004 [Codegen] `sqlc.yaml` と生成 package を `db_postgres` path へ同期する。
- [x] T005 [Architecture] `ARCHITECTURE.md`、`REGENERATIVE_ARCHITECTURE.md`、現役 manifest、設定、利用者向け current-state 文書を同期する。
- [x] T006 [Verify] 旧 `adapters` directory 不在、依存方向、codegen 冪等性、全検証 green を確認する。
- [x] T007 [Complete] completion evidence を記録し `done/` へ移動して commit する。

## Verification

- `just sqlc-generate`
- `just format-go`
- `just yaml-check`
- `just verify`
- `git diff --check`

## Risk Notes

Go backend のほぼ全 package import path に影響する高リスク refactor である。挙動は変更せず、
context 単位の機械的移動、codegen 同期、Architecture import graph、race test をゲートにする。
未信頼入力の解釈や認証・認可判断は変更しないため、新しい fuzz/property test は追加しない。

## Completion

- **Completed At**: 2026-07-20
- **Summary**:
  Go backend の context / feature adapter を Flat Wikipedia Architecture に再配置し、
  shared technical capability、文書、codegen、検証 path を同期した。
- **Affected Guarantees State**:
  - HTTP / SCL / DB schema の振る舞いは変更せず、既存の回帰テストと全 repository 検証が成功した。
  - Core から Adapter への依存を導入せず、Architecture import graph と production import 検査が成功した。
- **Verification Results**:
  - `just sqlc-generate` — passed
  - `just format-go` — passed
  - `just yaml-check` — passed（Architecture cross-check、traceability、YAML validation）
  - `just verify-go` — passed
  - `just verify` — passed（Go、UI、SCL、Architecture、derived artifacts）
  - `git diff --check` — passed
- **Evidence**:
  `backend` 配下の旧 adapters / persistence directory と旧 import path を検索し、残存がないことを確認した。
