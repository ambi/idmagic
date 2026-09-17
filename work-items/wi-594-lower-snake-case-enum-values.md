---
status: pending
authors: [tn]
risk: medium
reversibility: irreversible
created_at: 2026-09-18
priority: p2
depends_on: [wi-585-rewrite-api-rules-as-complete-api-guidelines]
change_kind: feature
affected_spec:
  - { path: spec/contexts/oauth2/models.tsp, symbol: IdMagic.Contract.AuthorizationDetailTypeState }
  - { path: spec/contexts/oauth2/models.tsp, symbol: IdMagic.Contract.McpResourceServerState }
  - { path: spec/contexts/sharedsignals/models.tsp, symbol: IdMagic.Contract.SsfStreamDirection }
  - { path: spec/contexts/application/models.tsp, symbol: IdMagic.Contract.ClientSecretCredentialStatus }
  - { path: spec/contexts/signing-keys/models.tsp, symbol: IdMagic.Contract.KeyProvider }
  - { path: spec/contexts/signing-keys/models.tsp, symbol: IdMagic.Contract.KeyUsage }
  - { path: spec/contexts/signing-keys/models.tsp, symbol: IdMagic.Contract.SigningKeyState }
  - { path: spec/contexts/application/models.tsp, symbol: IdMagic.Contract.RequiredAuthnStrength }
  - { path: spec/contexts/sharedsignals/models.tsp, symbol: IdMagic.Contract.RevocationReason }
---

# パスカルケースの列挙値を小文字のスネークケースに統一する

## Motivation

[API ガイドライン](../docs/design/application/api-guidelines.md)の「列挙値」は、IdMagic 独自の列挙値を小文字のスネークケースで表すと定める。
現行では 9 個の列挙がパスカルケースの値（`Active`、`VaultTransit`、`AgentKilled` など）を返し、同じ API の中で `pending_deletion` と `Active` が混在している。

## Scope

- `AuthorizationDetailTypeState`、`McpResourceServerState`、`SsfStreamDirection`、`ClientSecretCredentialStatus`、`KeyProvider`、`KeyUsage`、`SigningKeyState`、`RequiredAuthnStrength`、`RevocationReason` の値を小文字のスネークケースに改める。
- TypeSpec、Go の定数、PostgreSQL の `CHECK (col IN (...))` と既存行の値、フロントエンド、テストを追従させる。
- API ガイドラインの適用状況を改める。

## Out of Scope

- 標準が定める列挙値（`PS256`、`S256`、`DPoP`、グラントタイプの URN）。
- `RevocationReason` を CAEP の Security Event Token に載せる形式。SET 内の表記は Shared Signals の `standards.md` が定める。

## Design

Go の定数値を改め、TypeSpec の列挙値をそこから転記する。
データベースに保存済みの値は、スキーマの `CHECK` 制約を改めたうえで更新する。
プロダクトは未リリースであるため、旧値を読み替える互換処理は設けない。

`RevocationReason` は CAEP のイベントにも現れる。
SET の `reason_admin` などに値をそのまま載せているかを確認し、載せている場合は外部の受信者が観測する値も変わることを Plan で扱う。

## Plan

1. 各列挙の値が永続化される列と、外部へ送信される経路を洗い出す。
2. [[wi-593-kebab-case-static-path-segments]] と同じく、リリースベースラインの扱いを確認する。
3. Go の定数、スキーマ、TypeSpec の順に改める。

## Tasks

- [ ] T001 [Design] 永続化と外部送信の経路を洗い出す。
- [ ] T002 [Spec] TypeSpec の列挙値を改める。
- [ ] T003 [App] Go の定数とスキーマを改める。
- [ ] T004 [UI] フロントエンドの表示と辞書を追従させる。
- [ ] T005 [Docs] API ガイドラインの適用状況を改める。
- [ ] T006 [Verify] 変更を検証する。

## Verification

- `mise run check-schema`
- `mise run check-contract-drift`
- `mise run check-api-compat`
- `mise run verify`

## Risk Notes

保存済みの値を更新しないままスキーマの `CHECK` 制約を改めると、既存の行が制約に違反する。スキーマの変更と値の更新を同じ手順に置く。
