---
status: pending
authors: [tn]
risk: high
created_at: 2026-07-18
depends_on: [wi-254-backend-feature-vertical-slice-convention]
change_kind: refactor
spec_impact:
  kind: none
  reason: "context 境界・context_map を動かさない純粋な物理配置変更。SCL 規範振る舞いは不変で spec/scl.yaml 編集も scl-render も不要。"
initial_context:
  source: [backend/oauth2, ARCHITECTURE.md]
  tests: [backend/oauth2]
  stop_before_reading: [frontend, spec]
---

# oauth2 を client/consent/authorization/token/device の feature 垂直スライスへ再配置する

## Motivation

oauth2 は ≈10.3k LOC でリポジトリ最大の context。`domain/`・`usecases/`・`adapters/http`・
`adapters/persistence/*` に client・consent・authorization(authorize+code+par)・
token(exchange+refresh+revoke+introspect)・device という複数の sub-domain が同居する。
wi-254 で確立した feature 垂直スライス規約を適用して可読性と変更局所性を高める。ただし
authorize と token は相互結合（`complete_login` の token 発行、authorization_code↔exchange）が
あり、他 2 context より境界設計の判断を要するため risk: high とする。

## Scope

- `backend/oauth2/` を `client/` `consent/` `authorization/`（authorize+code+par）
  `token/`（exchange+refresh+revoke+introspect）`device/` の feature 層へ再配置（全層）。
  `git mv` で履歴を保持。
- authorize/token 間で共有される domain 型（`token`/`authorization_code`/`authorization_request`/
  `authorization_details`/`pkce` 等）の帰属を移動前に確定する。
- Go import path の一括置換と、context 横断ハブ（`backend/oauth2/adapters/http/routes.go` 等）の
  named import 修正。
- `ARCHITECTURE.md` frontmatter の `modules[].path` の oauth2 分を feature 粒度へ同期
  （`new-architecture` skill）。

## Out of Scope

- SCL（`spec/scl.yaml`）の規範定義・context_map の変更。
- `REGENERATIVE_ARCHITECTURE.md` §3.8 と `ARCHITECTURE.md` の規約散文の再編集
  （wi-254 で確定済み。本 wi は frontmatter の path 同期のみ）。
- `module.go`（DI 束）と `backend/cmd/internal/bootstrap` の組み立て構造の変更（据え置き）。
- OAuth2/OIDC の振る舞い・エンドポイント・wire 契約の変更。

## Plan

feature 案:

- `client/`: `client`/`client_secret`、`admin_clients`/`register_client`、client persistence。
- `consent/`: `consent`、`admin_consents`/`account_consents`、consent persistence。
- `authorization/`: `authorization_code`/`authorization_request`/`authorization_details`/`pkce`、
  `authorize`/`exchange_code`/`complete_login`、`push_authorization_request`(par)、
  authorize 系 handler（最大の `authorize_*.go` 10+ ファイルの塊）。
- `token/`: `refresh_token`/`token`、`exchange_token`/`refresh_tokens`/`revoke_token`/
  `introspect_token`、`userinfo`、token/denylist persistence。
- `device/`: `device_authorization`、`device_flow`、device code persistence。
- **境界設計を先に確定**: authorize↔token 間の共有 domain 型（`token`/`authorization_code` 等）を
  どちらの feature に帰属させるか、あるいは context ルート共有 `domain/` に残すかを移動前に決める。
  `complete_login` の token 発行は authorization 側にオーケストレーションを置き、token 発行の
  実体は token feature の usecase/port を named import して呼ぶ方針を基本線とする。
- package 名は各層名のまま。context 横断ハブで named import が必要（wi-254 と同方針）。

## Tasks

- [ ] T001 [Design] authorize↔token の共有 domain 型の帰属と cross-feature 呼び出し方向を確定し、
      Plan に追記する（移動前の設計判断）。
- [ ] T002 [Move] `backend/oauth2/` を `git mv` で 5 feature 配下へ再配置（全層）。共有型は
      T001 の決定に従い配置。
- [ ] T003 [Go] import path を一括置換し、context 横断ハブと cross-feature 参照の named import を修正。
- [ ] T004 [Docs] `ARCHITECTURE.md` frontmatter `modules[].path` の oauth2 分を `new-architecture`
      skill で feature 粒度へ同期。
- [ ] T005 [Verify] 下記 Verification を実行し全緑を確認。

## Verification

- `just verify-go` / `just build-go` / `just test-go` — format/lint/typecheck/build/テスト緑。
- `just yaml-check` / `just check-ids` — RA/SCL の ID・YAML 整合（SCL 不変を確認）。
- `just verify` — 全体スイートの最終確認。
- `git log --follow` で `git mv` の履歴保持を確認、旧配置への import 残存ゼロを grep で確認。

## Risk Notes

- **authorize↔token の結合**が最大リスク。移動前の境界設計（T001）で cross-feature import の
  方向を一方向（authorization → token）に整理し、循環 import を防ぐ。整理できない共有型は
  context ルート共有 `domain/` に残す。
- **最大の context**でファイル数・import 数が多い。`just build-go` / `just test-go` が網羅検証する。
- wi-254 完了（規約・パイロット確定）に依存。規約が固まる前に着手しない。
- module.go / bootstrap 据え置きにより DI 面の破壊的変更を回避する。
