---
status: accepted
authors: [tn]
created_at: 2026-07-04
---

# ADR-040: 属性スキーマとカスタム属性ポリシー

## コンテキスト

[[ADR-039]] で、core 以外のプロフィール属性を `User.attributes:
Map<String, AttributeValue>` に sparse に持つと決めた。本 ADR はその属性を
統治する schema の形と運用規約を定める。OIDC 標準クレームの組み込み属性と、
tenant 定義のカスタム属性を **同一の `UserAttributeDef` 機構**で扱う点が要点。
Keycloak UserProfile / Okta Custom Profile / Google customSchemas 相当。

## 決定

wi-19 の基盤 (PR i)。[[ADR-039]] の attribute bag を統治する。

属性は `UserAttributeDef` で定義し、組み込みカタログ (コード、全テナント共通) とテナント定義
`TenantUserAttributeSchema` (独立 aggregate、`tenant_id` をキーとする) の 2 階建てで持つ。実効定義は
組み込み ∪ テナント定義とし、custom key が組み込み key と衝突する schema は拒否する。各定義は
`key` / `type` / `required` / `editable_by_user` / `claim_name` / `oidc_scope` / `visibility`
(`private` / `self_readable` / `admin_readable` / `claim_exposed`) / `pii` (省略時 true) を持つ。
`ValidateAttributes` が実効定義に対して値を検証し、self-service 経路は `editable_by_user=true` の
属性のみ key 単位で merge する。`pii=true` の属性値は [[ADR-018]] に従い SHA-256 hash 化する。

tenant aggregate に embed せず独立 aggregate とするのは、schema 変更が tenant 本体より頻繁で
あること、後続 PR で別テーブル化したいこと、tenant 削除時の cascade を明示したいこと、による。
`pii` を省略時 true とするのは、GDPR 上の sensitive attribute かどうかの判断は tenant の運用責任と
しつつ、システム側は安全側 default で最小限の防御を提供するためである。属性定義フィールドの詳細と
値検証フローは [`backend/idmanagement/ARCHITECTURE.md`](../backend/idmanagement/ARCHITECTURE.md) に
置く。

## 影響

- 本 PR では `AttributeValue` / `UserAttributeDef` / `TenantUserAttributeSchema` の
  spec 型・zog 検証・`BuiltinUserAttributeDefs()` カタログ・in-memory repository
  まで。schema を編集する admin API、custom attribute scope での claim 露出、
  UI は後続 PR。
- `ValidateAttributes` / `ValidateAttributeValue` は usecase 層から呼ぶ前提の
  純関数として spec に置く。admin は全置換、self は editable_by_user の merge と
  権限差を usecase で吸収する。
