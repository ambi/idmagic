---
id: idp-wi-78-rename-client-to-oauth2-client-vocabulary
title: "ドメイン語彙の Client を OAuth2Client にリネームし指示対象を明確にする"
created_at: 2026-06-27
authors: ["tn"]
status: completed
risk: medium
---
# Motivation
idmagic には Application (上位集約)、SAML SP、WS-Federation relying party など複数の
「接続先アプリケーション」概念が並ぶ。その中で OAuth2/OIDC の client 集約だけが接頭辞なしの
`Client` という語で表現されており、`Client` が何を指すのか文脈なしには判別できない。

[[wi-69-application-catalog-aggregate-and-assignment]] / [[ADR-064]] で Application を
単一導線に統合し、[[wi-76-fold-advanced-protocol-settings-into-application-editor]] /
[[ADR-066]] で低レベル client 画面を撤去した結果、`Client` 集約は内部 API としてのみ残った。
ユビキタス言語上、この集約を `OAuth2Client` と明示し、派生する admin DTO / interface /
permission / event も OAuth2 接頭辞で揃える。

ただし `client_id` / `client_secret` / `token_endpoint_auth_method` / `ClientCredentials`
(grant) / `ClientSecretBasic` (認証方式) / `InvalidClient` (エラー) などは RFC が定める wire
標準語であり、文脈上曖昧でない。これらは標準との整合のためリネームしない。

# Scope
- **scl**: 集約 Client → OAuth2Client、AdminClient* DTO、ListAdminClients 等 interface、 AdminClientsManage permission、AdminClient* event をリネーム。owns_models / owns_interfaces / owns_permissions、screen/variant の interface 参照、語彙定義の文言を整合。, RFC wire 標準語は対象外として明示的に残す。
- **go**: spec.Client → spec.OAuth2Client とその参照 (~56 箇所)、admin client use case / handler / repository / event 型の改名。OAuth2 bounded context 中心。
- **ui**: AdminClient TS 型と関連参照の改名。HTTP path (/api/admin/clients) の扱いは別途判断。
- **data**: 永続化のテーブル/カラム名・JSON tag・HTTP path を変えるかは互換性の観点で決める。 wire/storage の後方互換を壊さない範囲に留める。

# Out of Scope
- [[wi-76-fold-advanced-protocol-settings-into-application-editor]] の機能変更。本 WI は リネームのみで挙動・フィールドを変えない。
- RFC wire 標準語 (client_id 等) の改名。
- WS-Federation / SAML 側の語彙。

# Verification
- [object Object]
- [object Object]
- [object Object]
- [object Object]
- [object Object]
- [object Object]
- 手動: 認可・トークン・管理 API のふるまいがリネーム前と同一であることを既存テストで確認する。

# Risk Notes
広域リネームは wire 標準語 (client_id 等) を巻き込むと標準乖離・後方互換破壊を招く。対象を
ドメイン集約とその admin 派生名に限定し、wire/storage 名は別判断で温存する。永続化スキーマや
HTTP path を変える場合はマイグレーション/互換層を伴う。挙動は変えない。

# Completion
- **Completed At**: 2026-06-28
- **Summary**:
  SCL は SCL 2.0 移行 (e65a6c0) で集約が既に OAuth2Client 化されていたが、admin 系
  (DTO / interface / permission / event) が旧集約名のまま AdminClient* として残っていた。
  AdminUser / AdminConsent と同じ Admin<集約> 規約に合わせ、admin 系一式を AdminOAuth2Client*
  にリネームした (ListAdminOAuth2Clients、AdminOAuth2ClientsManage、AdminOAuth2ClientCreated 等)。
  Go では集約型 spec.Client → spec.OAuth2Client、port OAuth2ClientRepository、admin usecase
  (CreateAdminOAuth2Client / UpdateAdminOAuth2Client / DeleteAdminOAuth2Client、Deps / Input)、
  event 型、handler、role 束ね、outbox / kafka_relay の event 名マッピングを整合させた。
  UI の AdminClient TS 型も AdminOAuth2Client にリネーム。生成 HTML を再生成した。
  wire/storage は WI 指示通り温存: JSON tag (client_id 等)、HTTP path (/api/admin/clients)、
  RBAC action 文字列 (admin:clients_manage)、RFC 7591 自己登録 (RegisterClient / ClientRegistered)、
  glossary のロール語 Client、wire 列挙 (ClientType / ClientCredentials 等) は不変。差分は
  純粋なリネーム (57 file、+265/-265)。挙動は変えていない。
- **Verification Results**:
  - [object Object]
  - [object Object]
  - [object Object]
  - [object Object]
  - [object Object]
  - [object Object]
