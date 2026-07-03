---
id: idp-wi-31-scim2-provisioning
title: "SCIM 2.0 provisioning を user / group lifecycle の外部契約として実装する"
created_at: 2026-06-20
authors: ["tn"]
status: pending
risk: high
---
# Motivation
Okta / Google IAM / Entra ID と連携する production IdP では、SCIM 2.0 による
user / group provisioning、deprovisioning、group sync が重要になる。
手作業の admin CRUD だけでは、入退社・異動・グループ更新を安全に同期できない。

# Scope
- **decision**: 新規 ADR: idmagic が SCIM server として振る舞う範囲を定義する。 User / Group schema mapping、active=false の扱い、hard delete を受けた場合の soft-delete/anonymization との関係、Bearer token 認証方式を決める。
- **scl**: ScimUser / ScimGroup / ScimServiceProviderConfig / ScimResourceType / ScimSchema を追加する。, CreateScimUser / PatchScimUser / DeleteScimUser / CreateScimGroup / PatchScimGroup / DeleteScimGroup を追加する。, provisioning event と permission を追加する。
- **go**: `/scim/v2/Users`, `/scim/v2/Groups`, `/scim/v2/ServiceProviderConfig`, `/scim/v2/ResourceTypes`, `/scim/v2/Schemas` を realm 配下に公開する。, SCIM bearer token / provisioning client を tenant-scoped client として管理する。, PATCH Operations を仕様準拠で処理し、user/group aggregate に変換する。, `active=false` は disable/deprovision として扱い、hard delete は policy に従う。, group membership は `wi-9` の Group aggregate と同期する。
- **ui**: admin settings に SCIM endpoint、token 発行/rotation、last sync、error history を表示する。, group/user detail に SCIM source を表示し、source-of-truth 属性は直接編集不可にする。
- **documentation**: README に Okta / Google / Entra ID からの SCIM 設定例を書く。

# Out of Scope
- idmagic が SCIM client として外部アプリへ push provisioning すること。
- password sync。
- HRIS connector。
- custom enterprise schema の完全汎用 mapping。初期は core + enterprise user 最小 subset。

# Verification
- [object Object]
- [object Object]
- [object Object]
- [object Object]
- [object Object]
- [object Object]
- 手動: SCIM Create User → Patch active=false → Reactivate → Delete の流れが internal user lifecycle と一致することを確認する。
- 手動: SCIM Group member 更新で effective roles が更新されることを確認する。

# Risk Notes
SCIM は外部 system of record との契約なので、内部 admin 操作よりデータ破壊の
影響が大きい。`active=false` と delete の意味を ADR で先に固定し、hard delete
は既存 ADR-036 と矛盾させない。
