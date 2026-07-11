---
status: completed
authors: [tn]
risk: high
created_at: 2026-07-11
depends_on: []
---

# shared 永続化アダプタに残る OAuth2 / Authentication 固有実装をコンテキストローカリティ化する

## Motivation

[[wi-172]]〜[[wi-179]]・[[wi-181]]・[[wi-182]] のコンテキストローカリティ横展開で
`backend/shared` は大きく整理されたが、`backend/shared/adapters/persistence`
配下には今なお特定の境界づけられたコンテキストに紐づくリポジトリ実装が残っている。
各ファイルの見出しコメント自体が `// AccessTokenDenylist (OAuth2)` /
`// SessionStore (Authentication)` のように帰属コンテキストを明記しており、
移設漏れであることは実装コメントからも裏付けられる。

調査の結果、経緯は一様ではないことが分かった。

- `memory`（access_token_denylist / authorization_codes /
  authorization_requests / device_codes / par / replay）と
  `postgres`（password_reset_token）は、当該コンテキスト側に窓口すら
  存在せず、`bootstrap` から `shared` の型を直接参照している。
- `memory/refresh_tokens.go` と `memory/consents.go` は
  `oauth2/adapters/persistence/memory` 側に型エイリアス／別実装が
  作られているが、実装本体は `shared` に残ったままか（refresh_tokens：
  `oauth2/adapters/persistence/memory/clients.go` が
  `sharedmem.NewRefreshTokenStore()` を呼ぶだけの薄いラッパー）、実装が
  完全に重複して `shared` 側が死んでいるか（consents：`bootstrap` は
  `oauth2memory.NewConsentRepository()` のみを使い、`shared` の
  `ConsentRepository` は本番配線・テストのどこからも参照されていない）。
- `valkey.go`（560 行）は [[wi-181]] の Scope で「oauth2 の token/grant
  backed store を `oauth2/adapters/persistence/valkey` へ同居」と明記
  されていたが、実際に生まれたのは型エイリアスの再輸出窓口
  （`oauth2/adapters/persistence/valkey/token_stores.go`）のみで、
  `AuthorizationRequestStore` / `AuthorizationCodeStore` / `PARStore` /
  `DeviceCodeStore` / `ReplayStore` / `AccessTokenDenylist` の実装本体は
  依然 `shared/adapters/persistence/valkey/valkey.go` にある。同ファイル内の
  `WebAuthnSessionStore` / `SessionStore`（Authentication）は
  [[wi-181]] の Scope で「`shared` に残置」と明示的に先送りされており、
  窓口すら存在しない。
- `postgres/refresh_tokens.go` は `oauth2/adapters/persistence/postgres`
  の sqlc 版に完全代替済みで、本番配線・テストいずれからも参照がない
  死んだ重複実装であることを確認した。

`postgres/base.go`（DB 接続設定）・`memory/helpers.go`・`valkey.go` の接続
ヘルパ／JSON・tenant key ユーティリティは複数コンテキストが利用する真の
共有インフラであり、[[wi-172]]〜以降も一貫して `shared` に残す方針が
とられてきたためそのまま維持する。

`postgres/keys.go`（`KeyStore`）・`postgres/tenant_salt_store.go`
（`TenantSaltStore`）は [[wi-173]]・[[wi-181]] が「SigningKeys 関連、
SigningKeys 自身の context 化は別途評価」として明示的に保留した論点であり、
本 WI でも同じ理由で Out of Scope とする。

## Scope

- `shared/adapters/persistence/memory` のうち OAuth2 分
  （access_token_denylist.go / authorization_codes.go /
  authorization_requests.go / device_codes.go / par.go / replay.go /
  refresh_tokens.go、各 `_test.go` を含む）を
  `oauth2/adapters/persistence/memory` へ物理移設する。
  `refresh_tokens.go` 移設に伴い `oauth2/adapters/persistence/memory/clients.go`
  の `NewRefreshTokenStore` ラッパーを実体に置き換える。
- `shared/adapters/persistence/memory` のうち Authentication 分
  （password_reset_token.go / sessions.go / webauthn_session.go、各
  `_test.go` を含む）を `authentication/adapters/persistence/memory` へ
  物理移設する。
- `shared/adapters/persistence/memory/consents.go`（死んだ重複実装）を削除する。
- `shared/adapters/persistence/postgres/password_reset_token.go` を
  `authentication/adapters/persistence/postgres` へ物理移設する。
- `shared/adapters/persistence/postgres/refresh_tokens.go`（死んだ重複実装）を
  削除する。
- `shared/adapters/persistence/valkey/valkey.go` のうち OAuth2 の
  token/grant backed store（`AuthorizationRequestStore` /
  `AuthorizationCodeStore` / `PARStore` / `DeviceCodeStore` /
  `ReplayStore` / `AccessTokenDenylist`）の実装本体を
  `oauth2/adapters/persistence/valkey` へ物理移設し、
  `token_stores.go` の型エイリアス窓口を実体定義に置き換える。
- `shared/adapters/persistence/valkey/valkey.go` のうち Authentication 分
  （`WebAuthnSessionStore` / `SessionStore`）と
  `shared/adapters/persistence/valkey/login_attempt_throttle.go`（+test）を、
  新設する `authentication/adapters/persistence/valkey` へ物理移設する。
- 接続ヘルパ（`Open` / `resilienceHook` 等）・`setJSON` / `getJSON` /
  `ttlUntil` / `tenantKey` 等の共通ユーティリティは `shared` に残す。
- `oauth2.Module` / `authentication.Module` へ上記フィールドを統合し、
  `bootstrap/deps.go` / `bootstrap/memory.go` / `bootstrap/postgres_valkey.go` /
  `shared/adapters/http/server/routes.go` の中央 `Deps` から該当フィールドを撤去する。
- 上記移設に伴い、`shared` の型を直接 import しているテスト
  （現状 `memory.NewRefreshTokenStore()` / `memory.NewDeviceCodeStore()` 等を
  直接 import しているテストが 10 箇所超ある）を各コンテキスト側の
  import に更新する。

## Out of Scope

- `postgres/keys.go`（`KeyStore`）・`postgres/tenant_salt_store.go`
  （`TenantSaltStore`）の移設。[[wi-173]]・[[wi-181]] が明示的に保留した
  「SigningKeys 自身の context 化は別途評価」を再度先送りする。signing key
  材料が oauth2 以外からも参照されうるかの判断は別途行う。
- `postgres/base.go`・`memory/helpers.go`・`valkey.go` の接続ヘルパ／共通
  ユーティリティの移設（真に共有インフラのため）。
- [[ADR-090]] の再評価。valkey backed store は従来通り sqlc 対象外の
  エスケープハッチとして扱う。
- 振る舞い・HTTP route・DB schema・公開 API の変更。ストアの型名・
  メソッドシグネチャは維持する。

## Plan

1. [[wi-172]]〜[[wi-179]]・[[wi-181]]・[[wi-182]] が確立した各コンテキストの
   `adapters/persistence` 型紙をそのまま踏襲する。
2. まず OAuth2 分（memory grant store 群・valkey grant store 群・死んだ
   postgres/memory 重複実装の削除）を実施して規模を実測する。[[wi-173]] が
   採った分岐基準（wi-172 実測 95 ファイルを上回れば分割）に照らして、
   超える場合は Authentication 分を別 WI に切り出す。
3. 続けて Authentication 分（memory: password_reset_token / sessions /
   webauthn_session、postgres: password_reset_token、valkey: 新設
   `authentication/adapters/persistence/valkey` への webauthn/session/
   login_attempt_throttle 移設）を行う。
4. 型エイリアス窓口（`oauth2/adapters/persistence/memory` の
   `NewRefreshTokenStore`、`oauth2/adapters/persistence/valkey` の
   型エイリアス群）を実体コードに置換し、shared を直接参照していた
   テストの import を更新する。
5. 各段階で `just test-go` を green に保ったまま進め、OAuth2 分と
   Authentication 分は別コミットに分離する。

## Tasks

- [x] T001 [Measure] 移設対象ファイル数・行数を実測し、wi-172 相当
  （95 ファイル）を上回るか判定する。上回る場合は本 WI を OAuth2 分に
  絞り、Authentication 分は後続 WI に分割する。
- [x] T002 [Persistence] OAuth2 memory 実装（access_token_denylist /
  authorization_codes / authorization_requests / device_codes / par /
  replay / refresh_tokens）を `oauth2/adapters/persistence/memory` へ
  移設し、`clients.go` の `NewRefreshTokenStore` ラッパーを実体に置換する。
- [x] T003 [Persistence] OAuth2 valkey 実装（AuthorizationRequestStore /
  AuthorizationCodeStore / PARStore / DeviceCodeStore / ReplayStore /
  AccessTokenDenylist）を `oauth2/adapters/persistence/valkey` へ移設し、
  `token_stores.go` の型エイリアスを実体定義に置換する。
- [x] T004 [Cleanup] 死んだ重複実装（`shared/adapters/persistence/memory/consents.go`、
  `shared/adapters/persistence/postgres/refresh_tokens.go`）を削除する。
- [x] T005 [Persistence] Authentication memory 実装（password_reset_token /
  sessions / webauthn_session）を `authentication/adapters/persistence/memory`
  へ移設する。
- [x] T006 [Persistence] Authentication postgres 実装（password_reset_token）を
  `authentication/adapters/persistence/postgres` へ移設する。
- [x] T007 [Persistence] `authentication/adapters/persistence/valkey` を
  新設し、WebAuthnSessionStore / SessionStore / LoginAttemptThrottle
  （+test）を移設する。
- [x] T008 [DI] `oauth2.Module` / `authentication.Module` へ該当フィールドを
  統合し、`bootstrap/deps.go` / `bootstrap/memory.go` /
  `bootstrap/postgres_valkey.go` / `shared/adapters/http/server/routes.go`
  の中央 `Deps` から該当フィールドを撤去する。
- [x] T009 [Tests] `shared` の memory/valkey/postgres パッケージを直接
  import しているテストを各コンテキスト側の import に更新する。
- [x] T010 [Verify] `just verify-go` / `just test-go` / `just build-go`
  （memory / postgres_valkey 両バックエンド） / `just dev` でスモークする。

## Verification

- `just verify-go` が green。
- `just test-go` で回帰なし。authorization code / PAR / device code /
  DPoP replay / client assertion replay / access token denylist / refresh
  token（OAuth2）、session / WebAuthn / password reset / login attempt
  throttle（Authentication）の単体・E2E が通ることを確認する。
- `just yaml-check` / `just check-ids` で SCL・双子定義・ID の整合。
- locality 指標：`grep -rn "shared/adapters/persistence" backend/oauth2 backend/authentication`
  が `KeyStore` / `TenantSaltStore`（Out of Scope）由来の `base.go` 参照等を
  除いてゼロに近づくことを確認する。
- `just build-go`（memory / postgres_valkey 両バックエンド起動）と
  `just dev` でスモーク。

## Risk Notes

- **risk: high**。OAuth2 の認可コード・PAR・device code・DPoP replay・
  access token denylist と、Authentication のセッション・WebAuthn・
  パスワードリセット・ログイン試行スロットルという、認可・認証双方の
  中核ストアを同時に動かす移設であり、[[wi-181]]（risk: high）と同水準の
  影響範囲を持つ。
- 軽減：OAuth2 分と Authentication 分を別コミット（規模次第では別 WI）に
  分離し、各段階で E2E を含む `just test-go` を通す。型エイリアス窓口の
  置換は既存の公開型名・メソッドシグネチャを変えない機械的移設に留め、
  独自判断を増やさない。

## Completion

- **Completed At**: 2026-07-11
- **Summary**: OAuth2 と Authentication 固有の memory / Postgres / Valkey 永続化
  アダプタを各コンテキストへ移設し、shared には接続設定と共通ユーティリティのみを残した。
  HTTP の中央 `Deps` から対象ストアを撤去し、各 Module を唯一の配線窓口にした。
- **Affected Guarantees State**: HTTP route、DB schema、公開 API、OAuth2/OIDC と
  Authentication のストア操作シグネチャおよび振る舞いは不変。
- **Verification Results**:
  - `just test-go` — passed
  - `just verify-go` — passed (lint 0 issues, race-enabled Go tests)
  - `just verify` — passed
  - `just yaml-check` / `just check-ids` — passed
- **Evidence**:
  - 実行日: 2026-07-11
  - 実行環境: ローカル開発環境
  - 実行主体: Codex
  - 対象ソース版: main（コミット前）
  - 保存先: 外部成果物なし。上記検証結果を本記録に要約。
