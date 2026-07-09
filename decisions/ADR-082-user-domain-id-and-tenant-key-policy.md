---
status: accepted
authors: [tn]
created_at: 2026-07-05
---

# ADR-082: User の domain identity と Postgres tenant key 方針

## コンテキスト

`users` テーブルは永続識別子を OIDC claim 名である `sub TEXT PRIMARY KEY` として
持っていた。他の aggregate は `id` / `client_id` / `application_id` など domain の
識別子名を持つのに対し、User だけが protocol vocabulary を storage identity として
露出させ、domain identity と protocol claim の境界が曖昧だった。本来は「User の安定
domain ID があり、それを OIDC `sub` / SAML NameID / WS-Fed subject / SCIM 参照へ
写像する」という domain → protocol の一方向写像が自然である。

加えて `tenant_id` の保持方針に一貫した根拠が無かった。`consents` /
`refresh_tokens` は `tenant_id` を持ち composite FK で tenant 整合を守る一方、
`mfa_factors` / `password_history` / `group_members` は直接の `tenant_id` を持たず、
`scim_user_refs` / `scim_group_refs` は `tenant_id` を PK に含みながら user / group
参照は global key のみで、tenant 不一致を DB 制約で拒否できなかった。新テーブル追加
時に同じ判断を再現できる基準が無い状態だった。

schema は ADR-071 により sqldef で宣言的に管理され、versioned migration framework は
持たない (wi-120 Out of Scope)。したがって列リネームは望ましい最終状態の宣言を
編集することで表現し、既存データの `sub` → `id` 読み替えは移行ノートとして残す。

## 決定

IdentityManagement context の `models.User` を `identity: id` へ変更し
(`id` フィールドに protocol 写像の記述を追加)、`AdminUserResponse` /
`AccountProfileResponse` / `AccountSummary` の識別子を `id` に、User を参照する
全 model / event / interface の識別子を `user_id` (event payload は camelCase の
`userId`) に統一。protocol 表現である OAuth2/OIDC の `AccessTokenClaims.sub` /
`IdTokenClaims.sub` / `UserInfoResponse.sub` / `IntrospectionResponse.sub`、
RFC 8693 token exchange の `act.sub`、Subject / DiscoveryDocument の `sub` は
protocol claim としてそのまま残す。`deploy/schema/postgres.sql` を本方針へ整列。
wi-120 で導入。ADR-071 (declarative schema with sqldef) を前提とする。

1. **User の canonical identifier は domain の `id`。** `users.sub TEXT PRIMARY KEY`
   を `users.id TEXT PRIMARY KEY` に改める。`id` は global 一意で tenant 跨ぎでも
   衝突しない。composite FK の標的として `UNIQUE (tenant_id, id)` を残す。

2. **`sub` は protocol 境界の写像結果に降格する。** OIDC `sub` claim は原則
   `user.id` から導出する。access token / id_token / userinfo / introspection の
   claim 構造体と RFC 8693 の `act.sub`、discovery の `claims_supported` に載る
   `sub`、SAML NameID / WS-Fed subject の source は protocol vocabulary として残し、
   値として `user.id` を写像する。pairwise subject など写像を差し替える場合も
   `user.id` 自体は不変とする。

3. **命名の責務分離。** storage / domain 内部では User 自身の識別子を `id`、User への
   参照を `user_id` (event payload は既存の camelCase 慣習に合わせ `userId`) とする。
   admin action の actor は `actor_user_id` / `actorUserId`、Agent 所有者は
   `owner_user_id` とする。Go でも `spec.User.Sub` を `spec.User.ID` (`json:"id"`) に
   改め、`UserSub` / `OwnerSub` / `ActorSub` を `UserID` / `OwnerUserID` /
   `ActorUserID` へ、SQL 列 `sub` / `user_sub` / `owner_sub` を `user_id` /
   `owner_user_id` へ揃える。`Sub` は OAuth2/OIDC の claim 直列化と inbound
   federation の外部 subject だけに閉じ込める。

4. **`tenant_id` 保持は 4 分類で判断する。** 検索・制約・保持・監査のいずれかの
   目的を持つ列だけを保持し、親から辿れるという理由だけで機械的に足さない。

   - **tenant-owned aggregate / tenant-scoped configuration** — `tenant_id` を PK
     または unique key に含め、tenant 内の一覧・検索・削除・権限制御に使う
     (`users`, `groups`, `clients`, `applications` 等)。
   - **tenant-scoped natural key を参照する child** — 参照先が
     `(tenant_id, local_id)` で識別されるなら child にも `tenant_id` を持たせ
     composite FK で tenant 不一致を DB が拒否する (`consents`, `refresh_tokens`,
     `application_orderings`, `agents.owner_user_id`)。
   - **globally unique parent だけに従属する child** — `users.id` / token hash /
     UUID など global key で親子関係と検索が満たせるなら `tenant_id` を重複保持
     しない (`mfa_factors`, `password_history`, `password_reset_tokens`,
     `email_change_tokens`, `group_members`)。ただし tenant 単位の高頻度検索・
     retention・partitioning・監査隔離が要件なら例外として保持し理由と index を
     残す。
   - **append-only / audit / outbox / throttling state** — emit-time tenant・query
     境界・保持/集計単位として tenant が必要かで判断する (`audit_events`,
     `authentication_event_buckets`, `outbox`)。

5. **`scim_user_refs` / `scim_group_refs` の tenant 整合を DB で守る。** 両者は
   `tenant_id` を PK に含むので、user / group 参照を composite FK
   (`users(tenant_id, id)` / `groups(tenant_id, id)`) に変更し、tenant 不一致な
   参照を DB が拒否できるようにする。global-key-only FK は tenant 越境参照を
   許す穴なので採らない。

6. **`audit_events.sub` を `user_id` に改める。** append-only の監査行だが、格納する
   のは domain の user 識別子であり protocol claim ではない。列名を `user_id` に
   揃え、AuditEventQuery の絞り込みキーも `user_id` にする。`tenant_id` は emit-time
   tenant として保持し、既存の `(tenant_id, occurred_at)` index を維持する。

## 却下した代替案

- **`users` に `sub` 列を別途残す。** 互換や pairwise 化の余地として `id` と併存
  させる案。現時点で `sub = id` の 1:1 写像であり別列は冗長。将来 pairwise が必要に
  なった時点で protocol 境界に写像を導入すればよく、storage に protocol claim 列を
  先取りしない。
- **schema 列は `sub` のまま Go 側だけ `ID` にする。** adapter で `id` 列 ↔ `Sub`
  フィールドを写像する低リスク案。domain と storage の語彙が食い違ったまま残り、WI
  が解消しようとした「protocol vocabulary の storage への露出」を schema 層に残す
  ため不採用。
- **全 child に機械的に `tenant_id` を足す。** 検索・監査を一律に楽にするが、
  global key だけで足りる child では tenant 不一致を許す重複列になりやすい。目的の
  ある列だけを持たせる 4 分類を採る。
- **`scim_*_refs` の tenant 整合を repository 実装だけで守る。** DB は global FK の
  ままアプリ層で tenant を検証する案。fail-closed を DB 制約で担保できず、別経路の
  書き込みで越境を許すため composite FK を採る。

## 影響

- `deploy/schema/postgres.sql`: `users` PK を `id` に、User 参照列
  (`mfa_factors` / `password_history` / `password_reset_tokens` /
  `email_change_tokens` / `consents` / `refresh_tokens` / `group_members` /
  `application_orderings` / `scim_user_refs` / `agents.owner_sub` /
  `audit_events.sub`) を `user_id` / `owner_user_id` に改名。`scim_user_refs` /
  `scim_group_refs` の user / group FK を composite 化。tenant_id 4 分類ポリシーを
  schema 冒頭コメントに記載。
- Go 全層で `Sub` → `ID` / `UserID` のリネームを行い、`Sub` は OAuth2/OIDC の
  claim 直列化と inbound federation の外部 subject に限定する。`json:"sub"` claim と
  外部 protocol 契約 (OIDC discovery / token / userinfo, SAML, WS-Fed, SCIM) は
  従来どおり `sub` / NameID を返す。
- admin / self-service の JSON 契約 (`AdminUserResponse` /
  `AccountProfileResponse` / `AccountSummary` の識別子、admin API パス
  `/api/admin/users/{user_id}`, `/admin/consents/{user_id}/{client_id}` 等) が
  `sub` から `id` / `user_id` に変わり、React UI もこれに追随する。
- 既存データは `sub` 値をそのまま `id` / `user_id` として読み替える (値は不変、列名
  のみ変更)。versioned migration は持たないため移行ノートとして扱う。
