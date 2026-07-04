---
id: wi-119-postgres-timestamp-column-policy
title: "Postgres schema の時刻列ポリシーを統一する"
created_at: 2026-07-04
authors: [tn]
status: completed
risk: medium
---

# Motivation
`deploy/schema/postgres.sql` では `created_at` と `updated_at` の有無、`DEFAULT now()` の有無、
および `issued_at` / `granted_at` / `occurred_at` / `added_at` などの用途別時刻名が
テーブルごとに揺れている。現状でも多くは domain の意味に沿った名前だが、永続化層の
共通方針が明文化されていないため、新しいテーブル追加時に「更新可能な aggregate なのに
`updated_at` がない」「履歴レコードに不要な `updated_at` を足す」などの判断ぶれが起きやすい。

Identity provider の永続化では、監査・認証・トークン・設定変更の時系列がセキュリティ調査、
運用復旧、同期、キャッシュ無効化に使われる。時刻列の意味と必須性を仕様と schema の両方で
揃え、created / updated / domain event time を混同しない状態にする。

# Scope
- `spec/scl.yaml` および関連 context spec で、永続化 model の時刻列方針を確認し、必要なら
  model / invariant / glossary 相当の記述を更新する。
- `deploy/schema/postgres.sql` の各テーブルを次の分類に整理する。
  - mutable aggregate / tenant configuration / admin-editable resource:
    `created_at TIMESTAMPTZ NOT NULL` と `updated_at TIMESTAMPTZ NOT NULL` を原則持つ。
  - append-only event / audit / outbox / history:
    発生時刻または記録時刻を表す `occurred_at`、`created_at`、`published_at` などを持ち、
    原則として `updated_at` は持たない。
  - credential / token / grant lifecycle:
    `issued_at`、`granted_at`、`expires_at`、`revoked_at`、`last_used_at` など domain 固有名を
    優先し、`created_at` / `updated_at` の追加が意味を曖昧にしないかを個別判断する。
  - pure join / reference mapping:
    監査・同期・UI 表示に必要な場合だけ `created_at` または `added_at` を持ち、更新されない
    join には `updated_at` を持たせない。
- `tenants`、`clients`、`users`、`groups`、`agents`、`authorization_detail_types`、
  `applications`、`application_sign_in_policies`、`tenant_default_sign_in_policies`、
  `application_categories`、`saml_service_providers`、`wsfed_relying_parties`、
  `scim_configs` など、更新可能に見えるテーブルの `created_at` / `updated_at` の要否と
  `NOT NULL` / `DEFAULT now()` 方針を揃える。
- `consents`、`refresh_tokens`、`audit_events`、`authentication_event_buckets`、`outbox`、
  `password_history`、`password_reset_tokens`、`email_change_tokens`、`application_icons`、
  `application_assignments`、`agent_credential_bindings`、`scim_tokens`、`scim_user_refs`、
  `scim_group_refs` など、例外または domain 固有時刻でよいテーブルは、その理由を仕様または
  schema 近傍で確認できる形にする。
- Go repository / usecase / seed data が DB schema の時刻列方針と一致するように更新する。
- 必要なら schema bootstrap や repository tests に、時刻列の保存・更新・非更新の期待を追加する。

# Initial Context
- `clients` は `created_at` を持つが `updated_at` を持たず、admin で更新可能な resource に見える。
- `tenants`、`groups`、`agents`、`saml_service_providers`、`wsfed_relying_parties` は
  `updated_at` が nullable で、`users` や `authorization_detail_types` と方針が揃っていない。
- `tenant_user_attribute_schemas`、`application_sign_in_policies`、`tenant_default_sign_in_policies`、
  `application_orderings` は `updated_at` だけを持ち、作成時刻を不要とする理由を明確にしたい。
- `applications`、`application_categories` は `created_at` / `updated_at` を持つが
  `DEFAULT now()` がなく、repository 側で時刻を入れる方針か DB default 方針かが混在している。
- `audit_events`、`authentication_event_buckets`、`refresh_tokens`、`consents`、token 系テーブルは
  domain 固有時刻名が妥当な可能性が高く、機械的な `created_at` / `updated_at` 追加は避ける。

# Out of Scope
- DB migration framework の変更、または sqldef 採用方針の変更。
- 履歴・監査・outbox の保持期間や削除ポリシーの変更。
- domain 上不要な時刻列を、見た目の統一だけを目的に追加すること。
- 既存レコードの意味を変える backfill や監査時刻の再解釈。

# Affected Guarantees
- tenant-owned mutable resource の変更時刻が一貫して追跡できること。
- append-only / audit / token lifecycle data の発生時刻と更新時刻を混同しないこと。
- Postgres schema と repository 実装が同じ時刻列契約を守ること。

# Verification
- `just yaml-check-scl`
- `just scl-render`
- `just yaml-check-work-items`
- `just check-ids`
- `just verify-go`
- 必要に応じて Postgres repository tests で insert / update 時刻列の期待を確認する。

# Risk Notes
時刻列の `NOT NULL` 化や `DEFAULT now()` 追加は、既存 seed、fixture、repository insert に影響する。
また、`updated_at` を自動更新する trigger を入れる場合は全更新経路へ効く一方で domain 側の
明示時刻と競合し得る。実装時は、DB default、repository-managed timestamp、trigger のどれを
採るかを先に決め、履歴・token・audit 系の例外を機械的な統一から守る。

# Completion
- **Completed At**: 2026-07-04
- **Summary**:
  Postgres schema に timestamp policy コメントを追加し、mutable aggregate / tenant-scoped configuration の
  `updated_at` を `NOT NULL DEFAULT now()` に統一した。`clients` と `scim_configs` には不足していた
  `updated_at` を追加し、Go model / repository / usecase / bootstrap / HTTP response の契約を合わせた。
  `consents`、`refresh_tokens`、`audit_events`、history / token / credential lifecycle 系は domain 固有時刻を
  維持し、汎用 `updated_at` を追加しない方針を schema test で確認した。
- **Verification Results**:
  - `just yaml-check-scl` - passed
  - `just scl-render` - passed
  - `GOCACHE=/tmp/idmagic-go-cache go test ./...` - passed
  - `just yaml-check` - passed
  - `just verify-go` - passed
  - `just verify-ui` - passed
- **Affected Guarantees State**:
  tenant-owned mutable resource の変更時刻は schema / Go model / repository の契約として必須化された。
  append-only / audit / token lifecycle data は domain 固有時刻のまま残し、created / updated / event time の
  混同を避ける状態になった。
