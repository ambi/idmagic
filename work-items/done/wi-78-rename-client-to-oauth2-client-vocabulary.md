---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# ドメイン語彙の Client を OAuth2Client にリネームし指示対象を明確にする

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

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
