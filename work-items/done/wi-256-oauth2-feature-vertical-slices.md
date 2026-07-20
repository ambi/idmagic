---
status: completed
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
- HTTP handler 自体の feature ディレクトリ移動。ADR-131 により、複数 feature を横断する
  protocol orchestration・wire validation・route 登録は context root shared adapter に維持する。

## Plan

feature 案:

- `client/`: `client`/`client_secret`、`admin_clients`/`register_client`、client persistence。
- `consent/`: `consent`、`admin_consents`/`account_consents`、consent persistence。
- `authorization/`: `authorization_code`/`authorization_request`/`authorization_details`/`pkce`、
  `authorize`/`complete_login`、`push_authorization_request`(par)、
  authorize 系 handler（最大の `authorize_*.go` 10+ ファイルの塊）。
- `token/`: `refresh_token`/`token`、`exchange_token`/`refresh_tokens`/`revoke_token`/
  `introspect_token`、`userinfo`、token/denylist persistence。
- `device/`: `device_authorization`、`device_flow`、device code persistence。
- **境界設計を先に確定**: authorize↔token 間の共有 domain 型（`token`/`authorization_code` 等）を
  どちらの feature に帰属させるか、あるいは context ルート共有 `domain/` に残すかを移動前に決める。
  `complete_login` の token 発行は authorization 側にオーケストレーションを置き、token 発行の
  実体は token feature の usecase/port を named import して呼ぶ方針を基本線とする。
- package 名は各層名のまま。context 横断ハブで named import が必要（wi-254 と同方針）。

### T001: authorize↔token 境界

- `authorization` は `AuthorizationRequest`、`AuthorizationCode`、PKCE と authorize / PAR /
  complete-login を所有する。`token` は token claims、sender constraint、`RefreshTokenRecord` と
  token grant、refresh、revoke、introspect、userinfo を所有する。`device` は
  `DeviceAuthorization` を所有する。
- 現行の `CompleteLogin` は token を発行せず authorization code を発行する。したがって
  authorization→token の呼出しは新設せず、code を消費する token 側が
  `authorization/domain` を named import する一方向依存とする。
- `events.go`、横断的な token issuer/authorizer/replay port、OAuth error・乱数・resource indicator・
  authorization details の共通 helper、HTTP の routes / validation / client-auth / httpdeps、
  共有 persistence テスト基盤は context ルートに残す。authorization detail type と MCP
  resource server も 5 feature のいずれにも単独帰属しないため root shared とする。
- HTTP adapter は client / consent / authorization / token / device の usecase を named import
  して dispatch する context root shared adapter とする（ADR-131）。feature 固有 persistence
  adapter は feature 配下へ移す。

## Tasks

- [x] T001 [Design] authorize↔token の共有 domain 型の帰属と cross-feature 呼び出し方向を確定し、
      Plan に追記した。`CompleteLogin` が token を発行しない現行実装を確認し、token →
      authorization の一方向依存を選択した。共有 helper / port / HTTP hub は root に残す。
- [x] T002 [Move] domain / port / usecase / persistence adapter と既存テストを `git mv` で
      client / consent / authorization / token / device 配下へ再配置した。HTTP handler は
      ADR-131 により root shared adapter に維持した。
- [x] T003 [Go] feature usecase import、cross-feature shared helper、root compatibility facade、
      HTTP dispatch の named import を修正した。振る舞い変更のない refactor のため新規 RED は
      対象外とし、移動した既存 domain / usecase / adapter テストが `just test-go` と
      `just verify-go` で green になることを確認した。
- [x] T004 [Docs] `ARCHITECTURE.md` frontmatter を OAuth2 feature の domain / port / usecase /
      adapter module と共有 sqlc binding に同期し、Architecture cross-check を通した。
- [x] T005 [Verify] 下記 Verification を実行し全緑を確認した。

## Verification

- `just verify-go` / `just build-go` / `just test-go` — format/lint/typecheck/build/テスト緑。
- `just yaml-check` / `just check-ids` — RA/SCL の ID・YAML 整合（SCL 不変を確認）。
- `just verify` — 全体スイートの最終確認。
- `git log --follow` で `git mv` の履歴保持を確認し、移動対象ソースが旧配置に残っていないこと、
  context 横断 hub が feature package を named import していることを grep で確認。

## Risk Notes

- **authorize↔token の結合**が最大リスク。移動前の境界設計（T001）で cross-feature import の
  方向を一方向（token → authorization）に整理し、循環 import を防ぐ。feature 横断イベントと
  公開互換 facade は context ルート共有 package に残す。
- **最大の context**でファイル数・import 数が多い。`just build-go` / `just test-go` が網羅検証する。
- wi-254 完了（規約・パイロット確定）に依存。規約が固まる前に着手しない。
- module.go / bootstrap 据え置きにより DI 面の破壊的変更を回避する。
- **fuzz/property test 判断**: 外部入力の文法・認証認可判定は変更せず物理配置だけを変更したため、
  新規 fuzz/property test は追加しない。既存 HTTP、domain、usecase、adapter テストと race test を
  回帰検証として使用した。

## Completion

- **Completed At**: 2026-07-20
- **Summary**:
  OAuth2 の feature 固有 domain / port / usecase / persistence adapter を client / consent /
  authorization / token / device の垂直スライスへ再配置した。feature 横断 helper と公開 import
  互換 facade は root に維持し、HTTP adapter は ADR-131 に従って feature usecase を dispatch する
  context root shared protocol adapter とした。`ARCHITECTURE.md` と traceability manifest の
  module / test path を新配置へ同期した。
- **Verification Results**:
  - `just verify-go` - passed（lint 0 issues、race test passed）
  - `just build-go` - passed
  - `just test-go` - passed
  - `just yaml-check` - passed（SCL / Work Item / Architecture / traceability）
  - `just check-ids` - passed（390 records）
  - `git diff --check` - passed
  - `just verify` - passed（Go、traceability strict、UI format/lint/typecheck/unit/build）
- **Affected Guarantees State**:
  - SCL、OAuth2/OIDC endpoint、wire contract、認可規則は変更なし。
  - `verification/manifest.yaml` の移動対象4テスト参照を新パスへ同期済み。
- **Evidence**:
  - 手順: repository root で `just verify` を実行。
  - 実行環境: macOS / Go 1.26.5 / Bun、localhost bind を要する race test は sandbox 外で実行。
  - 実行主体: Codex。
  - 対象ソース版: commit 前 working tree（wi-256 の全差分）。
  - 結果: passed。ログは本セッションの command output に保存。
