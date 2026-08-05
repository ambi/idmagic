---
status: accepted
authors: [tn]
created_at: 2026-07-04
---

# ADR-039: ユーザープロフィールの形 (thin core + sparse attribute bag)

## コンテキスト

`spec.User` は OIDC Core の最小プロファイル (`sub` / `preferred_username` /
`name` / `given_name` / `family_name` / `email` / `email_verified`) と運用
フラグ (`roles` / `mfa_enrolled` / `disabled_at` / `deleted_at` / timestamps)
しか持たない。本番 IdP (Keycloak / Okta / Google Workspace) と比べると、
OIDC §5.1 の残りの標準クレーム、SCIM `enterprise:User` 拡張相当の組織属性、
ライフサイクル属性、tenant 定義カスタム属性が欠落していた (動機は wi-19)。

当初これらを `spec.User` に **OIDC optional claim を個別フィールドで全部足し、
組織属性/連絡先/検証済み claim も専用構造体で持つ**設計を試みた。しかし:

- どんなテナントでも実際に使う属性は一部だけ。全ユーザーに ~25 個の
  optional フィールドを持たせるのはモデルも DB も無駄に肥大する。
- `last_login_at` 等を `UserLifecycle` に置きつつ `disabled_at` / `deleted_at`
  を `User` 直下に残すのは一貫性が無い。
- 多値連絡先や `verified_claims` (OIDC4IDA、本 WI では out_of_scope) は
  過剰設計だった。

## 決定

wi-19 の基盤 (PR i)。

`User` は thin core (識別・認証・表示名・RBAC・ライフサイクル) だけを型付きで持ち、それ以外の
プロフィール属性は単一の sparse な `attributes: Map<String, AttributeValue>` に格納する
(thin core + sparse attribute bag)。属性は組み込みカタログとテナント定義の 2 階建て schema
(`UserAttributeDef`) で駆動する。ライフサイクルは `status: UserStatus` + `status_changed_at` に
一本化し、旧 `disabled_at` / `deleted_at` フィールドは廃止する。後方互換は不要とし (pre-release)、
既存 DB 構造は作り変える。

全ユーザーに ~25 個の optional フィールドを型として持たせるとモデルも DB も無駄に肥大すること、
`disabled_at` / `deleted_at` を `User` 直下に残しつつ他のライフサイクル属性を別に持つのは一貫性が
無いことが、この形を選んだ理由である。型付き core と attribute bag の内訳、`address` claim の
フラット化、属性 schema の詳細は [[ADR-040]] に、tombstone 時の attributes 全消去は [[ADR-036]] に
委ねる。現在の設計は [`backend/idmanagement/ARCHITECTURE.md`](../backend/idmanagement/ARCHITECTURE.md)
に置く。

## 影響

- `spec.User` のフィールド数が大幅に減り、プロフィール拡張は `attributes` の
  key 追加だけで済む。既存 RP のクレーム集合は変わらない (core は据え置き、
  optional claim は値が入った時だけ露出)。
- 本 PR では in-memory adapter + Postgres (JSONB) が追従。多値属性の別テーブル
  化と self API・UI は後続 PR
  ([[wi-19-rich-user-attributes]] の scope_split)。
