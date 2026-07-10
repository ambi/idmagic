---
status: pending
authors: [tn]
risk: high
created_at: 2026-07-11
---

# oauth2 コンテキストの token/grant 系型・永続化をコンテキストローカリティ化する

## Motivation

[[wi-173]]（oauth2 コンテキストローカリティ横展開）は T001 の実測で規模が
wi-172（application パイロット、95 ファイル変更）を上回ることが判明し、
wi-173 自身の Plan に明記された分岐に従い client / token / audit・outbox の
3 分割に切り出した。本 WI はそのうち token/grant（authorization request /
authorization code / PAR / refresh token / device code / DPoP replay /
client assertion replay / access token denylist）を担当する。

oauth2 の認可コア（authorize/token エンドポイント）を扱うため、client/consent
（[[wi-173]]）に比べ振る舞いへの影響範囲が大きい。[[wi-173]] が確立する
`oauth2/domain` パッケージ構成・`oauth2.Module` の型紙をそのまま拡張する
前提のため、[[wi-173]] 完了後に着手する。

[[ADR-091]] は `authorize_handler.go`（1,144 行）の feature 単位分割を
横展開時の随伴タスクとして名指ししており、本 WI が authorize/token フロー
本体を扱うため、このファイル分割も本 WI の Scope に含める。

## Scope

- `internal/shared/spec/oauth2.go` のうち token/grant 系業務型
  （`AuthorizationRequest` / `AuthorizationCodeRecord` / `SenderConstraint` /
  `RefreshTokenRecord` / `PARRecord` / `DeviceAuthorization` /
  `AccessTokenClaims` / `IDTokenClaims`）を `internal/oauth2/domain/` へ移設。
- `shared/adapters/persistence/{postgres,memory}/refresh_tokens.go` を
  `internal/oauth2/adapters/persistence/{postgres,memory}` へ同居し sqlc 化。
- `shared/adapters/persistence/valkey/valkey.go`（560 行）のうち oauth2 の
  token/grant backed store（`AuthorizationRequestStore` /
  `AuthorizationCodeStore` / `PARStore` / `DeviceCodeStore` /
  `DpopReplayStore`（`ReplayStore` の DPoP 用途） /
  `ClientAssertionReplayStore`（同 client assertion 用途） /
  `AccessTokenDenylist`）を切り出し `internal/oauth2/adapters/persistence/valkey`
  へ同居（sqlc 対象外、[[ADR-090]] 適用外を明記するのみ）。
  `SessionStore` / `WebAuthnSessionStore` / `LoginAttemptThrottle` 等の
  authentication 系実装は `shared/adapters/persistence/valkey` に残置。
- `authorize_handler.go`（1,144 行）を feature 単位ファイルへ分割
  （[[ADR-091]] 決定 5）。
- [[wi-173]] で新設された `internal/oauth2/module.go` を拡張し、
  `RequestStore` / `CodeStore` / `PARStore` / `RefreshStore` /
  `DeviceCodeStore` / `DpopReplayStore` / `ClientAssertionReplayStore` /
  `AccessTokenDenylist` / `TokenIssuer` / `TokenIntrospector` / `Authorizer`
  を Module へ移す。`Deps`/`bootstrap` から該当分を撤去。

## Out of Scope

- client / consent / authorization detail type（[[wi-173]] で対応済み前提）。
- audit event / outbox（[[wi-182]] で扱う）。
- `KeyStore` / `TenantSaltStore`（SigningKeys 関連、[[wi-173]] と同じ理由で
  shared に残置。SigningKeys 自身の context 化は別途評価）。
- `TokenIssuer` / `TokenIntrospector` / `Authorizer` の実装ロジック変更
  （所在の移設のみ、振る舞い不変）。
- 振る舞い・HTTP route・DB schema・公開 API の変更。

## Plan

1. [[wi-173]] が確立した `oauth2/domain`・`oauth2/adapters/persistence`・
   `oauth2.Module` の型紙を踏襲し、内側→外側の順で進める。
2. valkey backed store の切り出しは `shared/adapters/persistence/valkey`
   から authentication 系実装を巻き込まないよう、型ごとに慎重に境界を引く。
3. `authorize_handler.go` の分割は振る舞い不変を最優先し、まず現状のテストを
   green に保ったまま機械的にファイルを割るところから始める。
4. [[wi-182]]（audit/outbox）と並行実施する場合は `module.go` /
   中央 `Deps`/`bootstrap` での衝突を避けるため、マージ順を先に調整する。

## Tasks

- [ ] T001 [Domain] token/grant 系業務型を `oauth2/domain/` へ移設し参照更新。
- [ ] T002 [Persistence] `refresh_tokens.go` を
  `oauth2/adapters/persistence/{postgres,memory}` へ同居。
- [ ] T003 [Persistence] `refresh_tokens.go` postgres 実装を sqlc 生成へ置換。
- [ ] T004 [Persistence] valkey backed token/grant store を
  `oauth2/adapters/persistence/valkey` へ切り出し。
- [ ] T005 [Adapters] `authorize_handler.go` を feature 単位ファイルへ分割。
- [ ] T006 [DI] `oauth2/module.go` を拡張し token/grant 分を Module 化。
- [ ] T007 [DI] 中央 `server/routes.go` `Deps` と `bootstrap/deps.go` から
  token/grant 分を撤去。
- [ ] T008 [Verify] `just verify-go` / `just test-go` green、locality 指標を確認。

## Verification

- `just verify-go` が green。
- `just test-go` で回帰なし。authorization code / client credentials /
  refresh / DPoP / PAR / device code フローの E2E・単体が通る。
- `just yaml-check` / `just check-ids` で SCL・双子定義・ID の整合。
- `just sqlc-generate` が冪等。
- `just build-go`（memory / postgres_valkey 両バックエンド起動）と
  `just dev` でスモーク。

## Risk Notes

- **risk: high**。token/grant はプロトコルの認可根幹であり、valkey backed
  store の切り出しと `authorize_handler.go` 分割の 2 つの構造変更が重なる
  ため、単一の型移設より不具合の混入余地が大きい。
- 軽減：valkey 切り出しと `authorize_handler.go` 分割を別コミットに分け、
  各段階で `just test-go`（既存 E2E 含む）を通す。[[wi-173]] の型紙を厳密に
  踏襲し独自判断を増やさない。
