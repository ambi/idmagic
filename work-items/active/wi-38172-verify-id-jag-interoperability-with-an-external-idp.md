---
status: pending
authors: [tn]
risk: medium
created_at: 2026-10-31
priority: p2
depends_on: []
change_kind: feature
affected_spec:
  - { path: docs/modules/oauth2/features/token.md, requirement: REQ-OAUTH2-066 }
  - { path: docs/modules/oauth2/features/token.md, requirement: REQ-OAUTH2-069 }
---

# 実在の外部 IdP と ID-JAG の相互運用を確かめる

## 動機

[Okta の Agent SSO](https://www.okta.com/newsroom/press-releases/okta-brings-first-class-identity-to-ai-agents-with-agent-sso/) は Cross App Access を一般提供し、多数の SaaS と接続している。
IdMagic は `draft-ietf-oauth-identity-assertion-authz-grant-05` に固定して ID-JAG を実装したが、外部製品との相互運用を確認していない。
既存の企業 IdP と共存するには、資源側の認可サーバーとして受け入れられることが前提になる。

## 対象範囲

- Okta を IdP、IdMagic を資源側の認可サーバーとする償還の確認。
- 確認で見つかった差分の修正と、対応範囲の明示。
- 実際の `jwks_uri` の取得を含む経路の確認。

## 対象外

- IdMagic が IdP として Okta 側へ発行する構成。対象の製品が見つかってから扱う。
- Identity Continuation Assertion の実装。draft の状況を T001 で記録するにとどめる。
- 認定や互換の対外的な主張。

## 設計

Okta のテナントが要るので、利用できる環境を T001 で確かめる。
利用できない場合、公開の適合試験や他の実装で代替できる範囲を決め、確認できない範囲を未検証として残す。
差分は draft の解釈の違いか製品固有かを分け、製品固有の挙動を規範へ持ち込まない。

## 計画

1. 検証環境と、確認する draft の版を決める。
2. 償還を実行して差分を一覧にする。
3. 必要な修正を行い、記録する。

## タスク

- [ ] T001 [Decision] 検証環境、対象の draft、代替の範囲を確定する。
- [ ] T002 [Verify] 実環境で発行と償還を実行し、差分を記録する。
- [ ] T003 [Spec] 必要なら REQ-OAUTH2-069 の周辺と標準の採用表を更新する。
- [ ] T004 [App] 差分を修正する。

## 検証

- 実環境の確認の手順と結果。外部環境を要するので、通常の CI では実行しない。
- 修正があれば `mise run test-go-package -- ./backend/oauth2/internal/token/usecases`。
- 最終検査は `mise run verify-changed -- <着手時の基準リビジョン>` とする。

## リスク

draft の改訂で再び差が生じ得る。
確認した版と日付を標準の採用表へ残す。
