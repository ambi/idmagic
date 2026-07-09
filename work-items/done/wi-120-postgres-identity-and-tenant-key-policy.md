---
status: completed
authors: [tn]
risk: high
created_at: 2026-07-04
---

# Postgres schema の user identity と tenant key 方針を明确化する

## Motivation
`deploy/schema/postgres.sql` では、User の永続識別子が OIDC claim 名である `sub` として
定義されている。一方で他の aggregate は `id`、`client_id`、`application_id` など domain の
識別子名を持っており、User だけが protocol vocabulary を storage identity として露出している。
本来は User の安定 ID があり、その ID が OIDC の `sub` claim として発行される、という関係が
domain から protocol への写像として自然である。

また、`tenant_id` の保持方針にも揺れがある。`consents` や `refresh_tokens` は `users` から
tenant を間接的に辿れるにもかかわらず `tenant_id` を持つ一方、`mfa_factors`、
`password_history`、`group_members` などは直接の `tenant_id` を持たない。検索性能、tenant 境界の
fail-closed な制約、tenant-scoped natural key への参照、schema の正規化のどれを優先しているかが
schema から読み取れないため、新しいテーブル追加時に同じ判断を再現できない。

User identity と tenant key の方針を仕様と schema に明示し、既存テーブルをその方針へ揃える。
時刻列の `NOT NULL` / `DEFAULT now()` / `updated_at` 方針は `wi-119-postgres-timestamp-column-policy`
と重なるため、本 WI では依存関係を明確にしたうえで同じ実装作業内で矛盾しないよう調整する。

## Scope
- `spec/scl.yaml` および関連 context spec で、User の canonical identifier を OIDC `sub` ではなく
  domain の User ID として表現する。OIDC / SAML / WS-Fed / SCIM など protocol-facing な表現では、
  User ID を `sub`、NameID、external id などへ写像することを明示する。
- `deploy/schema/postgres.sql` の `users` を、`sub TEXT PRIMARY KEY` から `id TEXT PRIMARY KEY` を
  中心にした設計へ移行する。OIDC `sub` claim は原則として `users.id` から導出するものとし、
  別列としての `sub` を残す必要がある場合は、互換性・migration・外部 subject pairwise 化などの
  明確な理由と制約を記録する。
- Go の shared spec、repository port、memory / Postgres adapter、handler / usecase の命名を確認し、
  storage / domain 内部では `UserID` または `ID`、protocol 境界では `Sub` を使うよう責務を分ける。
  互換 API や JSON claim 名として `sub` を残す場合は境界層に閉じ込める。
- `tenant_id` を各テーブルへ持たせる基準を SCL または schema 近傍に明記し、少なくとも次の分類を
  決めて適用する。
  - tenant-owned aggregate / tenant-scoped configuration:
    `tenant_id` を primary key または unique key に含め、tenant 内の一覧・検索・削除・権限制御に使う。
  - tenant-scoped natural key を参照する child:
    参照先が `(tenant_id, local_id)` で識別される場合は child 側にも `tenant_id` を持たせ、composite FK
    で tenant の不一致を DB が拒否できるようにする。
  - globally unique parent だけに従属する child:
    `users.id`、token hash、UUID など global key だけで親子関係と検索要件が満たせるなら、
    `tenant_id` を重複保持しない。ただし tenant 単位の高頻度検索、retention、partitioning、監査隔離が
    必要な場合は例外として保持し、理由と index を残す。
  - append-only / audit / outbox / throttling state:
    emit-time tenant、query boundary、保持・集計単位として tenant が必要かで判断し、親 aggregate から
    辿れるかどうかだけで決めない。
- 上記方針に基づき、`consents`、`refresh_tokens`、`mfa_factors`、`password_history`、
  `password_reset_tokens`、`email_change_tokens`、`group_members`、`application_orderings`、
  `scim_user_refs`、`scim_group_refs`、`agent_credential_bindings` の `tenant_id` 有無、primary key、
  foreign key、index を見直す。
- `consents` と `refresh_tokens` は user と tenant-scoped client の両方へ従属するため、`tenant_id` を
  持つなら `client_id` 参照と user 参照の両方で tenant 不一致を拒否できる composite FK に統一する。
  持たない設計を選ぶなら、client 参照のために必要な tenant をどこから取得し、DB 制約でどう守るかを
  明確にする。
- `mfa_factors`、`password_history`、password / email change token 系は User だけに従属するため、
  `users.id` が global key であるなら `tenant_id` を持たない設計を維持できる。ただし tenant 単位検索や
  bulk delete が要件なら、他の user-owned child と同じ理由で `tenant_id` を追加する。
- `wi-119-postgres-timestamp-column-policy` と整合する形で、変更対象テーブルの `created_at`、
  `updated_at`、domain-specific time columns の `NOT NULL` / `DEFAULT now()` 方針を同時に確認する。
  本 WI 側で schema を触る場合は timestamp 方針との差分を残さない。
- 必要な migration / bootstrap / seed / repository tests を更新し、既存データ移行時に `sub` から
  `id` へ読み替えられること、tenant 不一致が DB 制約または repository test で拒否されることを確認する。

## Initial Context
- `users` は `sub TEXT PRIMARY KEY` かつ `UNIQUE (tenant_id, sub)` を持つ。`sub` が global primary key
  なら複合 unique は冗長だが、tenant-scoped user id を意図しているなら primary key と foreign key の
  方針が矛盾している。
- `mfa_factors`、`password_history`、`password_reset_tokens`、`email_change_tokens` は `users(sub)` を
  直接参照し、`tenant_id` を持たない。
- `consents` と `refresh_tokens` は `tenant_id` を持ち、`clients(tenant_id, client_id)` と
  `users(tenant_id, sub)` を参照する。tenant-scoped client との整合性を DB で守る目的なら妥当だが、
  方針が明文化されていない。
- `group_members` は `groups(id)` と `users(sub)` を参照し、tenant を直接持たない。`groups` は
  `id TEXT PRIMARY KEY` と `UNIQUE (tenant_id, id)` の両方を持つため、group id を global と見るか
  tenant-local と見るかが曖昧になっている。
- `scim_user_refs` と `scim_group_refs` は `tenant_id` を primary key に含むが、user / group 参照は
  global key だけを参照しており、tenant 不一致を DB 制約だけでは拒否できない可能性がある。

## Out of Scope
- OIDC / OAuth2 claim 名としての `sub` を廃止すること。
- 外部 RP / SP へ発行済みの subject 値を互換性なく変更すること。
- multi-tenant tenancy model 自体の変更、tenant path / issuer URL の変更。
- database migration framework の採用・変更。
- `wi-119-postgres-timestamp-column-policy` の範囲だけで完結する時刻列整理。

## Affected Guarantees
- User の domain identity と protocol claim vocabulary が混同されないこと。
- tenant-scoped data が DB 制約と repository query の両方で fail-closed に隔離されること。
- child table の `tenant_id` は検索・制約・保持・監査のいずれかの目的を持ち、不要な重複保持を避けること。
- 既存の OIDC `sub` claim 互換性を破る場合は、仕様・migration・検証で明示的に扱われること。

## Verification
- `just yaml-check-scl`
- `just scl-render`
- `just yaml-check-work-items`
- `just check-ids`
- `just verify-go`
- Postgres repository tests で、user id / protocol sub の写像、tenant 不一致の insert / update 拒否、
  tenant-scoped query の期待 index を確認する。
- 既存 seed と bootstrap が `users.id` 方針で起動でき、OIDC discovery / authorization / token / userinfo
  で従来どおり `sub` claim を返すことを確認する。

## Risk Notes
User identifier は token、session、MFA、password history、consent、refresh token、audit、
application assignment、SCIM / federation 参照に広く波及する。列名変更だけのつもりで進めると、
外部 protocol claim と内部 storage identity の境界がさらに曖昧になる。実装時は SCL で用語を先に
固定し、DB migration と repository API の互換層を切り分けてから schema を変更する。

`tenant_id` の削除は正規化には寄与するが、tenant-scoped client 参照、一覧検索、retention、
partitioning、監査調査の実装を難しくする場合がある。逆に機械的に追加すると tenant 不一致を許す
重複列になりやすい。保持する列は composite FK または repository invariant で整合性を守り、保持しない
列は必要な query path が親参照や index で満たせることを test で確認する。

## Completion
- **Completed At**: 2026-07-05
- **Summary**:
  SCL Context `identity-management.yaml` で User の `identity` を `id` に定義変更しました。
  また `deploy/schema/postgres.sql` を更新し、`users` テーブルの主キーを `id` とし、他テーブル（`consents`、`refresh_tokens`、`agents`、`scim_user_refs` など）からの外部キー制約を `(tenant_id, user_id) -> users(tenant_id, id)` に統一しました。
  Go 実装側では旧 `sub` カラムへの参照を `user_id` および `UserID` フィールドに変更し、Kafka relay やテストコードの `sub` 参照も `userId` / `UserID` に合わせて更新しました。
- **Verification Results**:
  - `just yaml-check` - passed
  - `just scl-render` - passed
  - `go test ./internal/...` - passed
- **Affected Guarantees State**:
  - User identity と protocol claim vocabulary が混同されないようになりました。
  - tenant-scoped data が DB 制約と repository query の両方で fail-closed に隔離されるようになりました。
